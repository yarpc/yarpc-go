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

package yarpcroundrobin

import (
	"time"

	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeerlist"
)

// options describes how to build a round-robin peer list.
type options struct {
	capacity int
	shuffle  bool
	seed     int64
}

var defaultListOptions = options{
	capacity: 10,
	shuffle:  true,
	seed:     time.Now().UnixNano(),
}

// ListOption customizes the behavior of a roundrobin list.
type ListOption interface {
	apply(*options)
}

type listOption func(*options)

func (o listOption) apply(options *options) {
	o(options)
}

// Capacity specifies the default initial capacity of the underlying data
// structures for this list.
//
// Defaults to 10.
func Capacity(capacity int) ListOption {
	return listOption(func(options *options) {
		options.capacity = capacity
	})
}

// New creates a new round robin peer list.
func New(name string, dialer yarpc.Dialer, opts ...ListOption) *List {
	options := defaultListOptions
	for _, option := range opts {
		option.apply(&options)
	}

	plOpts := []yarpcpeerlist.ListOption{
		yarpcpeerlist.Capacity(options.capacity),
		yarpcpeerlist.Seed(options.seed),
	}
	if !options.shuffle {
		plOpts = append(plOpts, yarpcpeerlist.NoShuffle())
	}

	return &List{
		List: yarpcpeerlist.New(
			name,
			dialer,
			newPeerRing(),
			plOpts...,
		),
	}
}

// List is a PeerList that rotates which peers are to be selected in a cycle.
type List struct {
	*yarpcpeerlist.List
}
