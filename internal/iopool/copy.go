// Copyright (c) 2019 Uber Technologies, Inc.
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

package iopool

import (
	"io"
	"sync"
)

type buffer struct {
	b []byte
}

const _copyBufSize = 1024 * 32

var _pool = sync.Pool{
	New: func() interface{} {
		return &buffer{make([]byte, _copyBufSize)}
	},
}

// Copy copies bytes from the Reader to the Writer until the Reader is exhausted.
func Copy(dst io.Writer, src io.Reader) (int64, error) {
	// To avoid unnecessary memory allocations we maintain our own pool of
	// buffers.
	buf := _pool.Get().(*buffer)
	written, err := io.CopyBuffer(dst, src, buf.b)
	_pool.Put(buf)
	return written, err
}
