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
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// _responseDrainBufferSize is the size of the buffers that drainingBody.Close
// reads into.
const _responseDrainBufferSize = 4 << 10

var (
	// _responseDrainMaxBytes is the most that a response body's Close reads
	// while it waits for the end of the response. It matches the limit net/http
	// uses when it reads the rest of a request body after a handler returns.
	_responseDrainMaxBytes int64 = 256 << 10

	// _responseDrainTimeout is the longest that a response body's Close waits
	// for the end of the response. The end normally arrives within a few
	// hundred microseconds of the last bytes of the message. If it comes
	// later, Close closes the body anyway, as it would have without waiting.
	_responseDrainTimeout = time.Millisecond

	_responseDrainBufferPool = sync.Pool{
		New: func() interface{} {
			return new([_responseDrainBufferSize]byte)
		},
	}
)

// drainingBody is the body of a response returned by Outbound.Call. Its Close
// reads the rest of the response before it closes the body.
//
// A streaming decoder, such as the Thrift no-wire client's, stops reading as
// soon as the message ends. The end of the HTTP response can arrive after
// that: the END_STREAM flag on HTTP/2, or the last chunk on HTTP/1.1. Closing
// the body before it arrives resets the stream with RST_STREAM(CANCEL) on
// HTTP/2 and drops the connection on HTTP/1.1. Proxies such as Muttley and
// Envoy count either one as the caller hanging up (client-connection-failed),
// although the caller received the whole response.
//
// Normally no bytes are left to read, so Close only waits for the end of the
// response. It stops after _responseDrainMaxBytes or _responseDrainTimeout,
// whichever comes first, and then closes the body anyway, as it would have
// without waiting. Close never reports a failure to drain: by the time it runs,
// the caller already has what it needed from the body.
//
// When the caller's reads have already reached the end of the response, Close
// skips all of this and only closes the body. That is the case for encodings
// that read the whole body, and for HTTP/1.1 responses with a Content-Length,
// whose last bytes come with io.EOF.
type drainingBody struct {
	io.ReadCloser

	// ended is set once a Read returns an error, io.EOF included. The response
	// is then over, so Close has nothing to wait for.
	ended atomic.Bool

	closeOnce sync.Once
	closeErr  error
}

func (b *drainingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil {
		b.ended.Store(true)
	}
	return n, err
}

func (b *drainingBody) Close() error {
	if !b.ended.Load() {
		b.drain()
	}
	b.closeBody()
	return b.closeErr
}

// drain reads until the response ends, for at most _responseDrainMaxBytes and
// _responseDrainTimeout.
func (b *drainingBody) drain() {
	// Closing a net/http or x/net/http2 response body ends a Read that is
	// blocked on it, so the timer bounds how long one Read can wait. The
	// deadline bounds the loop for a body that keeps returning without ever
	// reaching the end.
	timer := time.AfterFunc(_responseDrainTimeout, b.closeBody)
	deadline := time.Now().Add(_responseDrainTimeout)

	buf := _responseDrainBufferPool.Get().(*[_responseDrainBufferSize]byte)
	for drained := int64(0); drained < _responseDrainMaxBytes && time.Now().Before(deadline); {
		n, err := b.ReadCloser.Read(buf[:min(int64(len(buf)), _responseDrainMaxBytes-drained)])
		// An empty read without an error makes no progress, so give up on the
		// body rather than spin on it.
		if err != nil || n == 0 {
			break
		}
		drained += int64(n)
	}
	_responseDrainBufferPool.Put(buf)

	timer.Stop()
}

// closeBody closes the underlying body once. If the drain timer and Close
// both call it, the second call waits for the first one to finish.
func (b *drainingBody) closeBody() {
	b.closeOnce.Do(func() { b.closeErr = b.ReadCloser.Close() })
}
