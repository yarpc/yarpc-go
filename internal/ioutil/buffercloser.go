// Copyright (c) 2017 Uber Technologies, Inc.
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

package ioutil

import (
	"bytes"
	"io"
	"sync"
)

var _pool = sync.Pool{
	New: func() interface{} {
		return &BufferCloser{
			b: &bytes.Buffer{},
		}
	},
}

// BufferCloser is a light wrapping around the bytes.Buffer that implements
// a "Close" method to return the buffer to a sync.Pool of buffers.
type BufferCloser struct {
	b *bytes.Buffer
}

// NewBufferCloser returns a new BufferCloser from the Buffer pool that has been
// reset.
func NewBufferCloser() *BufferCloser {
	buf := _pool.Get().(*BufferCloser)
	buf.b.Reset()
	return buf
}

// ReadFrom implements io.ReaderFrom
func (b *BufferCloser) ReadFrom(r io.Reader) (n int64, err error) {
	return b.b.ReadFrom(r)
}

// WriteTo implements io.WriterTo
func (b *BufferCloser) WriteTo(w io.Writer) (n int64, err error) {
	return b.b.WriteTo(w)
}

// Read implements io.Reader
func (b *BufferCloser) Read(p []byte) (n int, err error) {
	return b.b.Read(p)
}

// Close implements io.Closer.  This will return the buffer to the buffer pool.
func (b *BufferCloser) Close() error {
	_pool.Put(b)
	return nil
}
