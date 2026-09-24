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

// ErrRequestBodyReleased means the caller tried to read a request body after
// Close ended it and its buffer went back to the pool.
var ErrRequestBodyReleased = errors.New("pooled request body has been released")

// errRequestBodyNotRepeatable means GetBody was asked for a replay of a body
// that carries no encode function. Only a test can build one.
var errRequestBodyNotRepeatable = errors.New("pooled request body cannot be replayed")

// pooledBody is a request body backed by one buffer from the pool. It is live
// until the first of two events ends it:
//
//	live ──Read hit io.EOF──▶ ended = io.EOF
//	     ──Close────────────▶ ended = ErrRequestBodyReleased
//
// The buffer goes back to the pool as soon as the body has ended and no
// goroutine is inside the copy, so nobody can read bytes that are in the pool.
// A Close that lands during a Read ends the body and leaves the release to
// that Read.
//
// The end reason is the answer every later Read gets, and which one it is
// matters. io.EOF says the body really ended: the transport probes one Read
// past the declared ContentLength and expects it. ErrRequestBodyReleased says
// it did not, so the transport does not end the stream on a truncated request.
//
// A body that has ended is finished for good. To send the request again, the
// transport asks GetBody, which encodes the request into a new buffer. A
// replay therefore costs one more encode, and no buffer is held for longer
// than the write it serves.
type pooledBody struct {
	mu     sync.Mutex
	buf    *bufferpool.Buffer
	reader *bytes.Reader

	// encode writes the request into a new buffer from the pool. GetBody
	// calls it. It never changes, so GetBody reads it without the mutex.
	encode func() (*bufferpool.Buffer, error)

	// ended is nil while the body can still be read.
	ended error

	// reading is true while a goroutine is inside the copy. The buffer is
	// never released while it is set. Only the transport's writer goroutine
	// reads the body, so a flag is enough; a count is not needed.
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
		// The transport has the whole body and will not read again.
		p.end(io.EOF)
	}
	if p.ended != nil {
		// Either this Read drained the body, or a Close ran during the copy
		// and left the release here.
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

// GetBody encodes the request into a new buffer from the pool and returns a
// new body over it. The transport calls it to send the request again after an
// HTTP/2 GOAWAY or a retried connection error.
//
// It does not touch this body, so it works whether this body is live, drained
// or released. The transport closes a body before it asks for a replay, so a
// design that shared one buffer would have to hold that buffer for the whole
// call.
func (p *pooledBody) GetBody() (io.ReadCloser, error) {
	if p.encode == nil {
		return nil, errRequestBodyNotRepeatable
	}

	buf, err := p.encode()
	if err != nil {
		return nil, err
	}
	return newPooledBody(buf, p.encode), nil
}

// end records why the body stopped. The first reason wins, so a later Close
// cannot turn the io.EOF of a drained body into an error.
func (p *pooledBody) end(reason error) {
	if p.ended == nil {
		p.ended = reason
	}
}

// release returns the buffer to the pool. The caller holds p.mu and has
// established that no Read is inside the copy.
func (p *pooledBody) release() {
	if p.buf != nil {
		bufferpool.Put(p.buf)
		p.buf = nil
		p.reader = nil
	}
}
