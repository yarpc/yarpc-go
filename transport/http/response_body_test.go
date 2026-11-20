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

package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// setResponseDrainLimits changes the drainingBody limits for the rest of the
// test. Tests that call it must not run in parallel.
func setResponseDrainLimits(t *testing.T, maxBytes int64, timeout time.Duration) {
	origMaxBytes, origTimeout := _responseDrainMaxBytes, _responseDrainTimeout
	_responseDrainMaxBytes, _responseDrainTimeout = maxBytes, timeout
	t.Cleanup(func() {
		_responseDrainMaxBytes, _responseDrainTimeout = origMaxBytes, origTimeout
	})
}

// fakeBody is a response body that reads from a Reader and counts Close calls.
type fakeBody struct {
	io.Reader

	closeErr error
	closed   int
}

func (b *fakeBody) Close() error {
	b.closed++
	return b.closeErr
}

// countingReader counts Read calls.
type countingReader struct {
	io.Reader

	reads int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.Reader.Read(p)
}

// endlessReader returns one byte from every Read and never reaches the end.
type endlessReader struct{}

func (endlessReader) Read(p []byte) (int, error) {
	return copy(p, "x"), nil
}

// emptyReader returns nothing and no error from every Read.
type emptyReader struct{}

func (emptyReader) Read([]byte) (int, error) {
	return 0, nil
}

// blockingBody is a response body whose Read blocks until Close. It counts
// Close calls, which can come from the drain timer's goroutine.
type blockingBody struct {
	closed    chan struct{}
	closeOnce sync.Once
	closes    atomic.Int32
}

func (b *blockingBody) Read([]byte) (int, error) {
	<-b.closed
	return 0, errors.New("read on closed body")
}

func (b *blockingBody) Close() error {
	b.closes.Add(1)
	b.closeOnce.Do(func() { close(b.closed) })
	return nil
}

func TestDrainingBodyClose(t *testing.T) {
	t.Run("reads the rest of the body", func(t *testing.T) {
		setResponseDrainLimits(t, _responseDrainMaxBytes, time.Hour)
		r := strings.NewReader("message, then the rest of the response")
		fake := &fakeBody{Reader: r}
		body := &drainingBody{ReadCloser: fake}

		// Read only the message, like a decoder that stops when it ends.
		_, err := io.ReadFull(body, make([]byte, len("message")))
		require.NoError(t, err)

		require.NoError(t, body.Close())
		assert.Zero(t, r.Len(), "unread bytes left in the body")
		assert.Equal(t, 1, fake.closed, "Close calls")
	})

	t.Run("does not read after a Read reached the end", func(t *testing.T) {
		setResponseDrainLimits(t, _responseDrainMaxBytes, time.Hour)
		r := &countingReader{Reader: strings.NewReader("message")}
		fake := &fakeBody{Reader: r}
		body := &drainingBody{ReadCloser: fake}

		_, err := io.ReadAll(body)
		require.NoError(t, err)
		reads := r.reads

		require.NoError(t, body.Close())
		assert.Equal(t, reads, r.reads, "Read calls after the end of the response")
		assert.Equal(t, 1, fake.closed, "Close calls")
	})

	t.Run("does not read after a Read error", func(t *testing.T) {
		setResponseDrainLimits(t, _responseDrainMaxBytes, time.Hour)
		r := &countingReader{Reader: iotest.ErrReader(errors.New("stream reset"))}
		fake := &fakeBody{Reader: r}
		body := &drainingBody{ReadCloser: fake}

		_, err := body.Read(make([]byte, 1))
		require.Error(t, err)

		require.NoError(t, body.Close())
		assert.Equal(t, 1, r.reads, "Read calls")
		assert.Equal(t, 1, fake.closed, "Close calls")
	})

	t.Run("stops at the byte limit", func(t *testing.T) {
		setResponseDrainLimits(t, 1<<10, time.Hour)
		r := strings.NewReader(strings.Repeat("x", 4<<10))
		body := &fakeBody{Reader: r}

		require.NoError(t, (&drainingBody{ReadCloser: body}).Close())
		assert.Equal(t, 3<<10, r.Len(), "unread bytes left in the body")
		assert.Equal(t, 1, body.closed, "Close calls")
	})

	t.Run("stops at the timeout", func(t *testing.T) {
		setResponseDrainLimits(t, _responseDrainMaxBytes, 10*time.Millisecond)
		body := &blockingBody{closed: make(chan struct{})}

		done := make(chan error, 1)
		go func() { done <- (&drainingBody{ReadCloser: body}).Close() }()
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(testtime.Second):
			t.Fatal("Close did not return after the drain timeout")
		}
		assert.Equal(t, int32(1), body.closes.Load(), "Close calls")
	})

	t.Run("stops at the timeout when the body never ends", func(t *testing.T) {
		setResponseDrainLimits(t, 1<<62, 10*time.Millisecond)
		body := &fakeBody{Reader: endlessReader{}}

		done := make(chan error, 1)
		go func() { done <- (&drainingBody{ReadCloser: body}).Close() }()
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(testtime.Second):
			t.Fatal("Close did not return after the drain timeout")
		}
		assert.Equal(t, 1, body.closed, "Close calls")
	})

	t.Run("gives up on an empty read", func(t *testing.T) {
		setResponseDrainLimits(t, _responseDrainMaxBytes, time.Hour)
		body := &fakeBody{Reader: emptyReader{}}

		require.NoError(t, (&drainingBody{ReadCloser: body}).Close())
		assert.Equal(t, 1, body.closed, "Close calls")
	})

	t.Run("ignores a read error", func(t *testing.T) {
		setResponseDrainLimits(t, _responseDrainMaxBytes, time.Hour)
		body := &fakeBody{Reader: iotest.ErrReader(errors.New("stream reset"))}

		assert.NoError(t, (&drainingBody{ReadCloser: body}).Close())
		assert.Equal(t, 1, body.closed, "Close calls")
	})

	t.Run("returns the close error", func(t *testing.T) {
		setResponseDrainLimits(t, _responseDrainMaxBytes, time.Hour)
		closeErr := errors.New("close failed")
		body := &fakeBody{Reader: strings.NewReader(""), closeErr: closeErr}

		assert.Equal(t, closeErr, (&drainingBody{ReadCloser: body}).Close())
	})
}

// BenchmarkDrainingBodyClose measures what Close adds when nothing is left to
// read: when the caller has not read the end of the response yet, so Close
// arms the timer and reads it, and when the caller already has.
func BenchmarkDrainingBodyClose(b *testing.B) {
	b.Run("end not yet read", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			body := &drainingBody{ReadCloser: io.NopCloser(strings.NewReader(""))}
			_ = body.Close()
		}
	})

	b.Run("end already read", func(b *testing.B) {
		b.ReportAllocs()
		var buf [1]byte
		for i := 0; i < b.N; i++ {
			body := &drainingBody{ReadCloser: io.NopCloser(strings.NewReader(""))}
			_, _ = body.Read(buf[:])
			_ = body.Close()
		}
	})
}

func TestCallResponseCloseWaitsForResponseEnd(t *testing.T) {
	// The handler sends the whole payload, then holds back the end of the
	// response (END_STREAM on HTTP/2, the last chunk on HTTP/1.1) for
	// endDelay. A callee's server does the same when the body is larger than
	// its write buffer: the end goes out only after the handler returns.
	const endDelay = 50 * time.Millisecond
	payload := bytes.Repeat([]byte("x"), 8<<10)

	// Wait well past endDelay, so that the test fails only on an early close.
	setResponseDrainLimits(t, _responseDrainMaxBytes, testtime.Second)

	tests := []struct {
		name  string
		http2 bool
	}{
		{name: "http1"},
		{name: "http2", http2: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canceled := make(chan bool, 1)
			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				// Read the request to the end, so that the HTTP/1.1 server
				// notices when the client drops the connection.
				_, _ = io.Copy(io.Discard, req.Body)
				_, _ = w.Write(payload)
				w.(http.Flusher).Flush()

				// Closing the response early cancels the request's context:
				// with RST_STREAM on HTTP/2, by dropping the connection on
				// HTTP/1.1.
				select {
				case <-req.Context().Done():
					canceled <- true
				case <-time.After(endDelay):
					canceled <- false
				}
			})

			var opts []OutboundOption
			if tt.http2 {
				handler = h2c.NewHandler(handler, &http2.Server{})
				opts = append(opts, UseHTTP2())
			}
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)

			httpTransport := NewTransport()
			t.Cleanup(func() {
				if err := httpTransport.Stop(); err != nil {
					t.Logf("failed to stop transport: %v", err)
				}
			})

			out := httpTransport.NewSingleOutbound(server.URL, opts...)
			require.NoError(t, out.Start(), "failed to start outbound")
			t.Cleanup(func() {
				if err := out.Stop(); err != nil {
					t.Logf("failed to stop outbound: %v", err)
				}
			})

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			res, err := out.Call(ctx, &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  raw.Encoding,
				Procedure: "hello",
				Body:      strings.NewReader("world"),
			})
			require.NoError(t, err)

			// Read exactly the payload, like a decoder that stops when the
			// message ends, and close the body.
			got := make([]byte, len(payload))
			_, err = io.ReadFull(res.Body, got)
			require.NoError(t, err)
			assert.Equal(t, payload, got)
			require.NoError(t, res.Body.Close())

			assert.False(t, <-canceled, "closing the response body canceled the request on the server")
		})
	}
}
