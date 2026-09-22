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
	"io"
	"sync"

	"go.uber.org/yarpc/internal/bufferpool"
)

// ErrRequestBodyReleased is the error a Read returns after Close. Close has
// then put the buffer back in the pool.
var ErrRequestBodyReleased = errors.New("pooled request body has been released")

// pooledBody is a request body. It holds one buffer from the pool.
//
// The body ends when a Read reaches io.EOF or when Close runs. The first event
// wins. Then the buffer goes back to the pool. If Close runs during a Read, the
// Read puts the buffer back.
//
// The end reason is the answer to every later Read. A drained body answers
// io.EOF. The transport reads once more after ContentLength bytes and expects
// io.EOF. A closed body answers ErrRequestBodyReleased. This error stops the
// transport from ending the stream cleanly on a truncated request.
//
// To send the request again, the transport calls GetBody. GetBody encodes the
// request into a new buffer.
type pooledBody struct {
	mu     sync.Mutex
	buf    *bufferpool.Buffer
	reader *bytes.Reader

	// encode writes the request into a new buffer from the pool. GetBody
	// calls it. It never changes, so GetBody can read it without the mutex.
	encode func() (*bufferpool.Buffer, error)

	// ended is nil while the body can still be read.
	ended error

	// reading is true while a goroutine is inside the copy. While it is true,
	// nobody releases the buffer.
	reading bool
}

var (
	_ io.ReadCloser = (*pooledBody)(nil)
	_ interface {
		GetBody() (io.ReadCloser, error)
	} = (*pooledBody)(nil)
)

func newPooledBody(buf *bufferpool.Buffer, encode func() (*bufferpool.Buffer, error)) *pooledBody {
	return &pooledBody{
		buf:    buf,
		reader: bytes.NewReader(buf.Bytes()),
		encode: encode,
	}
}

func (p *pooledBody) Read(dst []byte) (int, error) {
	p.mu.Lock()
	if p.ended != nil {
		p.mu.Unlock()
		return 0, p.ended
	}
	p.reading = true
	reader := p.reader
	p.mu.Unlock()

	n, err := reader.Read(dst)

	p.mu.Lock()
	p.reading = false
	if err == io.EOF {
		// The transport has the whole body. It does not read again.
		p.end(io.EOF)
	}
	if p.ended != nil {
		// Two cases lead here. This Read drained the body. Or Close ran
		// during the copy and left the release to this Read.
		p.release()
	}
	p.mu.Unlock()

	return n, err
}

func (p *pooledBody) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.end(ErrRequestBodyReleased)
	if !p.reading {
		p.release()
	}
	return nil
}

// GetBody encodes the request into a new buffer from the pool. It returns a
// new body for that buffer. The transport calls it to send the request again
// after an HTTP/2 GOAWAY or a retried connection error.
func (p *pooledBody) GetBody() (io.ReadCloser, error) {
	buf, err := p.encode()
	if err != nil {
		return nil, err
	}
	return newPooledBody(buf, p.encode), nil
}

// end records why the body stopped. The first reason wins. A later Close does
// not change the io.EOF of a drained body into an error.
func (p *pooledBody) end(reason error) {
	if p.ended == nil {
		p.ended = reason
	}
}

// release returns the buffer to the pool. The caller holds p.mu. The caller
// must be sure that no Read is inside the copy.
func (p *pooledBody) release() {
	if p.buf != nil {
		bufferpool.Put(p.buf)
		p.buf = nil
		p.reader = nil
	}
}
