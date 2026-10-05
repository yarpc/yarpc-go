// Copyright (c) 2026 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package thrift

import (
	"bytes"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/bufferpool"
)

func newTestPooledBody(tb testing.TB, payload []byte) *pooledBody {
	tb.Helper()

	return newPooledBody(fillTestBuffer(tb, payload), func() (*bufferpool.Buffer, error) {
		return fillTestBuffer(tb, payload), nil
	})
}

func fillTestBuffer(tb testing.TB, payload []byte) *bufferpool.Buffer {
	tb.Helper()

	buf := bufferpool.Get()
	_, err := buf.Write(payload)
	require.NoError(tb, err, "failed to fill the test buffer")
	return buf
}

func (p *pooledBody) isReleased() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf == nil
}

// The happy path: read the body to EOF, then close it. A drained body must
// release at once. The transport closes it only after the whole response has
// arrived, so waiting for Close would hold the buffer for the response
// latency as well.
func TestPooledBodyReadsWholePayload(t *testing.T) {
	payload := bytes.Repeat([]byte("abcd"), 1024)

	body := newTestPooledBody(t, payload)
	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	require.True(t, body.isReleased(),
		"reaching EOF must return the buffer without waiting for Close")

	// Close on an already drained body is a no-op.
	require.NoError(t, body.Close())
	require.True(t, body.isReleased())
}

// A drained body must keep answering io.EOF. The transport probes one Read
// past the declared ContentLength and an error there fails a good request.
func TestPooledBodyReadAfterDrainReportsEOF(t *testing.T) {
	body := newTestPooledBody(t, []byte("payload"))

	_, err := io.ReadAll(body)
	require.NoError(t, err)
	require.True(t, body.isReleased())

	for i := 0; i < 3; i++ {
		n, err := body.Read(make([]byte, 8))
		assert.Zero(t, n)
		require.ErrorIs(t, err, io.EOF)
		require.NotErrorIs(t, err, ErrRequestBodyReleased)
	}
}

// A body that is never read still has to release. A middleware can reject the
// request before the transport touches the body, so Close is the only event.
func TestPooledBodyCloseWithoutReadReleases(t *testing.T) {
	body := newTestPooledBody(t, []byte("payload"))

	require.NoError(t, body.Close())
	require.True(t, body.isReleased(), "Close must release a body that was never read")
}

func TestPooledBodyReadAfterCloseIsRefused(t *testing.T) {
	body := newTestPooledBody(t, []byte("payload"))
	require.NoError(t, body.Close())

	n, err := body.Read(make([]byte, 8))
	assert.Zero(t, n)
	// Not io.EOF: io.EOF would tell the HTTP/2 writer the body ended
	// cleanly and it would send END_STREAM on a truncated request.
	require.ErrorIs(t, err, ErrRequestBodyReleased)
	assert.NotErrorIs(t, err, io.EOF)
}

func TestPooledBodyCloseIsIdempotent(t *testing.T) {
	body := newTestPooledBody(t, []byte("payload"))

	// A second Close must not return the buffer to the pool twice.
	// bufferpool panics on a double release.
	require.NoError(t, body.Close())
	require.NoError(t, body.Close())
	require.NoError(t, body.Close())
}

// The reason GetBody encodes again instead of sharing the buffer: the
// transport closes the body before it asks for a replay, so the first buffer
// is already back in the pool by then.
func TestPooledBodyGetBodyReplaysAfterRelease(t *testing.T) {
	payload := bytes.Repeat([]byte("xyz"), 4096)
	body := newTestPooledBody(t, payload)

	require.NoError(t, body.Close())
	require.True(t, body.isReleased(), "the first buffer must be back in the pool")

	replay, err := body.GetBody()
	require.NoError(t, err, "GetBody must work after the first body is released")

	got, err := io.ReadAll(replay)
	require.NoError(t, err)
	assert.Equal(t, payload, got, "the replay must carry the whole request")

	require.NoError(t, replay.Close())
	assert.True(t, replay.(*pooledBody).isReleased(), "the replay must release its own buffer")
}

// A replay owns its buffer. Closing one body must not release the other's.
func TestPooledBodyGetBodyBodiesAreIndependent(t *testing.T) {
	payload := []byte("payload")
	body := newTestPooledBody(t, payload)

	replay, err := body.GetBody()
	require.NoError(t, err)

	require.NoError(t, replay.Close())
	assert.False(t, body.isReleased(), "closing the replay must not release the first body")

	require.NoError(t, body.Close())
	assert.True(t, body.isReleased())
}

// More than one GOAWAY in a row: every body carries the same encode
// function.
func TestPooledBodyGetBodyCanReplayAReplay(t *testing.T) {
	payload := []byte("payload")
	body := newTestPooledBody(t, payload)
	require.NoError(t, body.Close())

	for i := 0; i < 3; i++ {
		next, err := body.GetBody()
		require.NoError(t, err, "replay %d", i)

		got, err := io.ReadAll(next)
		require.NoError(t, err)
		require.Equal(t, payload, got, "replay %d", i)
		require.NoError(t, next.Close())

		body = next.(*pooledBody)
	}
}

// A body built without an encode function cannot be replayed, and says so
// rather than handing back a nil body.
func TestPooledBodyGetBodyWithoutEncodeFails(t *testing.T) {
	body := newPooledBody(fillTestBuffer(t, []byte("payload")), nil)
	defer func() { require.NoError(t, body.Close()) }()

	replay, err := body.GetBody()
	assert.Nil(t, replay)
	require.ErrorIs(t, err, errRequestBodyNotRepeatable)
}

// Read and Close race each other, with pool churn running. Close does not
// wait for an in-flight Read, so whichever finishes last must release the
// buffer: exactly once, and never while the other still reads it.
func TestPooledBodyCloseDuringReads(t *testing.T) {
	const (
		iterations = 200
		payloadLen = 64 << 10
	)

	payload := bytes.Repeat([]byte{0x5A}, payloadLen)

	stopChurn := startPoolChurn(payloadLen)
	defer stopChurn()

	for i := 0; i < iterations; i++ {
		body := newTestPooledBody(t, payload)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			dst := make([]byte, 4<<10)
			for {
				n, err := body.Read(dst)
				if err != nil {
					return
				}
				// Touch what was read, so the race detector can catch it.
				for _, b := range dst[:n] {
					_ = b
				}
			}
		}()

		go func() {
			defer wg.Done()
			assert.NoError(t, body.Close())
		}()

		wg.Wait()

		// Whoever finished last must have released the buffer.
		body.mu.Lock()
		released := body.buf == nil
		reading := body.reading
		body.mu.Unlock()

		require.True(t, released, "the buffer must be released once both goroutines are done")
		require.False(t, reading, "no Read may remain in flight")
	}
}

// startPoolChurn keeps taking buffers from the global pool, writing to them,
// and giving them back, until the returned function is called. It stands in
// for all the other outbound traffic in a real process.
//
// If the request body's buffer goes back to the pool while the HTTP/2 writer
// still reads it, this loop takes that buffer and writes over those bytes.
// go test -race then reports the read and the write as a data race.
func startPoolChurn(size int) func() {
	var (
		stop    atomic.Bool
		wg      sync.WaitGroup
		workers = runtime.GOMAXPROCS(0)
	)
	if workers < 2 {
		workers = 2
	}
	if workers > 4 {
		workers = 4
	}

	// One worker per P. sync.Pool keeps per-P free lists, so a single worker
	// often never sees the buffer that another P released.
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pattern := bytes.Repeat([]byte{0xAB}, size)
			for !stop.Load() {
				buf := bufferpool.Get()
				_, _ = buf.Write(pattern)
				bufferpool.Put(buf)
			}
		}()
	}

	return func() {
		stop.Store(true)
		wg.Wait()
	}
}
