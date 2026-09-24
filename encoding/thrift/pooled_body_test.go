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

// The happy path: read the body to EOF, then close it. A drained body
// releases its buffer before Close.
func TestPooledBodyReadsWholePayload(t *testing.T) {
	payload := bytes.Repeat([]byte("abcd"), 1024)

	body := newTestPooledBody(t, payload)
	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	require.True(t, body.isReleased(),
		"reaching EOF must return the buffer without waiting for Close")

	// Close on a drained body does nothing.
	require.NoError(t, body.Close())
	require.True(t, body.isReleased())
}

// A drained body keeps answering io.EOF. The transport reads once more after
// ContentLength bytes. An error there fails a good request.
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

// A body that nobody reads releases its buffer on Close.
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
	// Not io.EOF. With io.EOF, the HTTP/2 writer thinks the body ended
	// cleanly. It then sends END_STREAM on a truncated request.
	require.ErrorIs(t, err, ErrRequestBodyReleased)
	assert.NotErrorIs(t, err, io.EOF)
}

func TestPooledBodyCloseIsIdempotent(t *testing.T) {
	body := newTestPooledBody(t, []byte("payload"))

	// A second Close must not return the buffer to the pool again.
	// bufferpool panics on a double release.
	require.NoError(t, body.Close())
	require.NoError(t, body.Close())
	require.NoError(t, body.Close())
}

// GetBody encodes again. A replay works after the first body is closed.
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

// A replay owns its buffer. Closing one body must not release the buffer of
// the other body.
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

// GOAWAY more than once in a row. Every body has the same encode function.
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

// Read and Close race while pool churn runs. The last one to finish must
// release the buffer. The buffer is released exactly once.
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
				// Touch the bytes that were read. The race detector can then
				// see a race.
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

		// The last one to finish must have released the buffer.
		body.mu.Lock()
		released := body.buf == nil
		reading := body.reading
		body.mu.Unlock()

		require.True(t, released, "the buffer must be released once both goroutines are done")
		require.False(t, reading, "no Read may remain in flight")
	}
}

// startPoolChurn takes buffers from the global pool, writes to them, and puts
// them back. It runs until the returned function is called. If a body buffer
// is released too early, this loop reuses it and go test -race reports a data
// race.
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

	// One worker for each P. sync.Pool keeps a free list for each P. One worker
	// can miss a buffer that another P released.
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
