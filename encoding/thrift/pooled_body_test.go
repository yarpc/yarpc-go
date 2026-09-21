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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/bufferpool"
)

func newTestPooledBody(tb testing.TB, payload []byte) (*pooledBody, func()) {
	tb.Helper()

	buf := bufferpool.Get()
	_, err := buf.Write(payload)
	require.NoError(tb, err, "failed to fill the test buffer")
	return newPooledBody(buf)
}

// The happy path: read the body to EOF, then release.
func TestPooledBodyReadsWholePayload(t *testing.T) {
	payload := bytes.Repeat([]byte("abcd"), 1024)

	body, release := newTestPooledBody(t, payload)
	defer release()

	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	require.NoError(t, body.Close())
}

// Close must not release the buffer: a replay still needs it.
func TestPooledBodyCloseDoesNotRelease(t *testing.T) {
	payload := bytes.Repeat([]byte("xyz"), 4096)

	body, release := newTestPooledBody(t, payload)
	defer release()

	require.NoError(t, body.Close())

	// The buffer is still ours, so a rewind can still read it.
	_, err := body.Seek(0, io.SeekStart)
	require.NoError(t, err)

	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, got, "the replay must carry the whole request")
}

func TestPooledBodyReadAfterCloseIsRefused(t *testing.T) {
	body, release := newTestPooledBody(t, []byte("payload"))
	defer release()

	require.NoError(t, body.Close())

	n, err := body.Read(make([]byte, 8))
	assert.Zero(t, n)
	// Not io.EOF: that would send END_STREAM on a truncated request.
	require.ErrorIs(t, err, ErrRequestBodyClosed)
	assert.NotErrorIs(t, err, io.EOF)
}

func TestPooledBodyCloseIsIdempotent(t *testing.T) {
	body, release := newTestPooledBody(t, []byte("payload"))
	defer release()

	require.NoError(t, body.Close())
	require.NoError(t, body.Close())
	require.NoError(t, body.Close())
}

// A rewind after a partial read must still carry the whole request.
func TestPooledBodySeekRewindsAfterPartialRead(t *testing.T) {
	payload := bytes.Repeat([]byte("abcd"), 1024)

	body, release := newTestPooledBody(t, payload)
	defer release()

	_, err := io.CopyN(io.Discard, body, 64)
	require.NoError(t, err)

	_, err = body.Seek(0, io.SeekStart)
	require.NoError(t, err)

	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

// More than one GOAWAY in a row.
func TestPooledBodyCanReplayMoreThanOnce(t *testing.T) {
	payload := []byte("payload")

	body, release := newTestPooledBody(t, payload)
	defer release()

	for i := 0; i < 3; i++ {
		require.NoError(t, body.Close())

		_, err := body.Seek(0, io.SeekStart)
		require.NoError(t, err, "replay %d", i)

		got, err := io.ReadAll(body)
		require.NoError(t, err)
		require.Equal(t, payload, got, "replay %d", i)
	}
}

// Read and Close race each other, with pool churn running.
func TestPooledBodyCloseDuringReads(t *testing.T) {
	const (
		iterations = 200
		payloadLen = 64 << 10
	)

	payload := bytes.Repeat([]byte{0x5A}, payloadLen)

	stopChurn := startPoolChurn(payloadLen)
	defer stopChurn()

	for i := 0; i < iterations; i++ {
		body, release := newTestPooledBody(t, payload)

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
		release()
	}
}

// A cancelled call can release while the writer goroutine is still reading.
// release must not free the buffer out from under it.
func TestPooledBodyReleaseDuringReads(t *testing.T) {
	const (
		iterations = 200
		payloadLen = 64 << 10
	)

	payload := bytes.Repeat([]byte{0x5A}, payloadLen)

	stopChurn := startPoolChurn(payloadLen)
	defer stopChurn()

	for i := 0; i < iterations; i++ {
		body, release := newTestPooledBody(t, payload)

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
			release() // no Close first, as on a cancelled call
		}()

		wg.Wait()
	}
}

// After release, a rewind must not hand the buffer back out.
func TestPooledBodySeekAfterReleaseIsRefused(t *testing.T) {
	body, release := newTestPooledBody(t, []byte("payload"))

	release()

	_, err := body.Seek(0, io.SeekStart)
	require.ErrorIs(t, err, ErrRequestBodyClosed)

	n, err := body.Read(make([]byte, 8))
	assert.Zero(t, n)
	require.ErrorIs(t, err, ErrRequestBodyClosed)
}
