// Copyright (c) 2024 Uber Technologies, Inc.
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

// Package bufferpool maintains a pool of bytes.Buffers for use in
// encoding and transport implementations.
package bufferpool

import (
	"flag"
	"sync"
)

var _pool = NewPool()

// Option configures a buffer pool.
type Option func(*Pool)

// Pool represents a buffer pool with a set of options.
type Pool struct {
	testDetectUseAfterFree bool
	pool                   sync.Pool
}

func init() {
	// This is a hacky way to determine whether we are running in unit tests where
	// we want to enable use-after-free detection.
	// https://stackoverflow.com/a/36666114
	if flag.Lookup("test.v") != nil {
		_pool = NewPool(DetectUseAfterFreeForTests())
	}
}

// NewPool returns a pool that we can allocate buffers from.
func NewPool(opts ...Option) *Pool {
	pool := &Pool{}
	for _, opt := range opts {
		opt(pool)
	}
	return pool
}

// DetectUseAfterFreeForTests is an option that allows unit tests to detect
// bad use of a pooled buffer after it has been released to the pool.
func DetectUseAfterFreeForTests() Option {
	return func(p *Pool) {
		p.testDetectUseAfterFree = true
	}
}

// Get returns a buffer from the pool.
func (p *Pool) Get() *Buffer {
	buf, ok := p.pool.Get().(*Buffer)
	if !ok {
		buf = newBuffer(p)
	} else {
		buf.reuse()
	}
	return buf
}

func (p *Pool) release(buf *Buffer) {
	p.pool.Put(buf)
}

// Get returns a new Buffer from the Buffer pool that has been reset.
func Get() *Buffer {
	return _pool.Get()
}

// Put returns a Buffer to the Buffer pool.
func Put(buf *Buffer) {
	buf.Release()
}
