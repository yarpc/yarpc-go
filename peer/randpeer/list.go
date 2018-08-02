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

package randpeer

import (
	"math/rand"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/peerlist/v2"
)

type listOptions struct {
	capacity int
	source   rand.Source
}

var defaultListOptions = listOptions{
	capacity: 10,
}

// ListOption customizes the behavior of a random list.
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
	return listOptionFunc(func(options *listOptions) {
		options.capacity = capacity
	})
}

// Seed specifies the seed for generating random choices.
func Seed(seed int64) ListOption {
	return listOptionFunc(func(options *listOptions) {
		options.source = rand.NewSource(seed)
	})
}

// Source is a source of randomness for the peer list.
func Source(source rand.Source) ListOption {
	return listOptionFunc(func(options *listOptions) {
		options.source = source
	})
}

// New creates a new random peer list.
func New(transport peer.Transport, opts ...ListOption) *List {
	options := defaultListOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	if options.source == nil {
		options.source = rand.NewSource(time.Now().UnixNano())
	}

	plOpts := []peerlist.ListOption{
		peerlist.Capacity(options.capacity),
		peerlist.NoShuffle(),
	}

	return &List{
		List: peerlist.New(
			"random",
			transport,
			newRandomList(options.capacity, options.source),
			plOpts...,
		),
	}
}

// List is a PeerList that rotates which peers are to be selected randomly
type List struct {
	*peerlist.List
}
