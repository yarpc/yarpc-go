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

package hashring32

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/peerlist"
)

type listConfig struct {
	capacity    int
	numReplicas int
}

var defaultListConfig = listConfig{
	capacity:    10,
	numReplicas: 100,
}

// ListOption customizes the behavior of a roundrobin list.
type ListOption func(*listConfig)

// Capacity specifies the default capacity of the underlying
// data structures for this list
//
// Defaults to 10.
func Capacity(capacity int) ListOption {
	return func(c *listConfig) {
		c.capacity = capacity
	}
}

// NumReplicas specifies the number of replicas for each peer in the ring.
//
// Defaults to 100.
func NumReplicas(n int) ListOption {
	return func(c *listConfig) {
		c.numReplicas = n
	}
}

// New creates a new hash ring peer list.
func New(transport peer.Transport, name string, hashFunc HashFunc, opts ...ListOption) *List {
	cfg := defaultListConfig
	for _, o := range opts {
		o(&cfg)
	}

	ring := newPeerRing(name, hashFunc, cfg.numReplicas)

	return &List{
		List: peerlist.New(
			"hashring32",
			transport,
			ring,
			peerlist.Capacity(cfg.capacity),
		),
		ring: ring,
	}
}

// Len returns the number of members of the ring.
func (l *List) Len() int {
	return l.ring.Len()
}

// List is a PeerList which rotates which peers are to be selected in a circle
type List struct {
	*peerlist.List

	ring *ring
}
