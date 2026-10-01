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
	"errors"
	"sync"

	"go.uber.org/yarpc/internal/bufferpool"
)

// ErrRequestBodyClosed means the caller tried to read a request body after
// Close, or after its buffer went back to the pool.
var ErrRequestBodyClosed = errors.New("pooled request body is closed")

// pooledBody is a rewindable request body backed by a pooled buffer. Read and
// Seek hold the mutex across the operation, so Close and release cannot run in
// the middle of one; after release, both refuse.
//
// Close does not release the buffer. The transport closes the body before it
// rewinds for a retry, so a Seek clears the closed state. release cannot
// assume the writer is done either: the transport writes the body on a
// goroutine of its own, and a cancelled call can return before that goroutine
// does.
type pooledBody struct {
	mu     sync.Mutex
	reader *bytes.Reader

	closed   bool
	released bool
}

// newPooledBody creates a pooledBody over the encoded bytes in buf. The
// returned release function returns the buffer to the pool; the caller must
// defer it, since Close does not release the buffer.
func newPooledBody(buf *bufferpool.Buffer) (*pooledBody, func()) {
	body := &pooledBody{reader: bytes.NewReader(buf.Bytes())}

	release := func() {
		body.mu.Lock()
		defer body.mu.Unlock()
		body.released = true
		bufferpool.Put(buf)
	}
	return body, release
}

func (p *pooledBody) Read(dst []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed || p.released {
		return 0, ErrRequestBodyClosed
	}
	return p.reader.Read(dst)
}

func (p *pooledBody) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	return nil
}

// Seek moves the read offset. The transport uses it to replay the request
// after an HTTP/2 GOAWAY or a retried connection error, so it also clears the
// closed state left by the close that precedes the replay.
func (p *pooledBody) Seek(offset int64, whence int) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.released {
		return 0, ErrRequestBodyClosed
	}
	p.closed = false
	return p.reader.Seek(offset, whence)
}
