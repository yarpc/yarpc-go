// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcpendingheap

import (
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeerlist"
)

type listOptions struct {
	capacity int
	shuffle  bool
}

var defaultListOptions = listOptions{
	capacity: 10,
	shuffle:  true,
}

// ListOption customizes the behavior of a pending requests peer heap.
type ListOption interface {
	apply(*listOptions)
}

type listOptionFunc func(*listOptions)

func (f listOptionFunc) apply(options *listOptions) { f(options) }

// Capacity specifies the default capacity of the underlying
// data structures for this list.
//
// Defaults to 10.
func Capacity(capacity int) ListOption {
	return listOptionFunc(func(c *listOptions) {
		c.capacity = capacity
	})
}

// New creates a new pending heap.
func New(dialer yarpc.Dialer, opts ...ListOption) *List {
	cfg := defaultListOptions
	for _, o := range opts {
		o.apply(&cfg)
	}

	plOpts := []yarpcpeerlist.ListOption{
		yarpcpeerlist.Capacity(cfg.capacity),
	}
	if !cfg.shuffle {
		plOpts = append(plOpts, yarpcpeerlist.NoShuffle())
	}

	return &List{
		List: yarpcpeerlist.New(
			"fewest-pending-requests",
			dialer,
			&pendingHeap{},
			plOpts...,
		),
	}
}

// List is a PeerList which rotates which peers are to be selected in a circle
type List struct {
	*yarpcpeerlist.List
}
