// Copyright (c) 2016 Uber Technologies, Inc.
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

package sync

import (
	"sync"

	"github.com/uber-go/atomic"
)

// Once is a wrapper around sync.Once in order to simplify returning the
// same error multiple times from the same function.
type Once struct {
	done atomic.Bool
	once sync.Once
	err  error
}

// Do is a wrapper around the sync.Once `Do` method. This version takes a function that
// returns an error, and every subsequent call to the `Do` function will be returned the
// `err` of the `f` func.
// If f is nil we will replace it with a noop function.
func (o *Once) Do(f func() error) error {
	if f == nil {
		f = func() error { return nil }
	}

	o.once.Do(func() {
		o.err = f()
		o.done.Store(true)
	})

	return o.err
}

// Done returns whether the finished flag has been set and thus sync.Once has been run.
func (o *Once) Done() bool {
	return o.done.Load()
}
