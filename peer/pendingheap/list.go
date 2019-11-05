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

package pendingheap

import (
	"math/rand"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/abstractlist"
	"go.uber.org/zap"
)

type listConfig struct {
	capacity  int
	shuffle   bool
	failFast  bool
	autoFlush bool
	seed      int64
	nextRand  func(int) int
	logger    *zap.Logger
}

var defaultListConfig = listConfig{
	capacity: 10,
	shuffle:  true,
	seed:     time.Now().UnixNano(),
}

// ListOption customizes the behavior of a pending requests peer heap.
type ListOption func(*listConfig)

// Capacity specifies the default capacity of the underlying
// data structures for this list.
//
// Defaults to 10.
func Capacity(capacity int) ListOption {
	return func(c *listConfig) {
		c.capacity = capacity
	}
}

// Seed specifies the random seed to use for shuffling peers.
//
// Defaults to time in nanoseconds.
func Seed(seed int64) ListOption {
	return func(c *listConfig) {
		c.seed = seed
	}
}

// Logger provides an optional logger for the list.
func Logger(logger *zap.Logger) ListOption {
	return func(c *listConfig) {
		c.logger = logger
	}
}

// FailFast indicates that the peer list should not wait for a peer to become
// available when choosing a peer.
//
// This option is preferrable when the better failure mode is to retry from the
// origin, since another proxy instance might already have a connection.
func FailFast() ListOption {
	return func(c *listConfig) {
		c.failFast = true
	}
}

// New creates a new pending heap.
func New(transport peer.Transport, opts ...ListOption) *List {
	cfg := defaultListConfig
	for _, o := range opts {
		o(&cfg)
	}

	plOpts := []abstractlist.Option{
		abstractlist.Capacity(cfg.capacity),
		abstractlist.Logger(cfg.logger),
	}
	if !cfg.shuffle {
		plOpts = append(plOpts, abstractlist.NoShuffle())
	}
	if cfg.failFast {
		plOpts = append(plOpts, abstractlist.FailFast())
	}
	if cfg.autoFlush {
		plOpts = append(plOpts, abstractlist.AutoFlush())
	}

	nextRandFn := nextRand(cfg.seed)
	if cfg.nextRand != nil {
		// only true in tests
		nextRandFn = cfg.nextRand
	}

	return &List{
		List: abstractlist.New(
			"fewest-pending-requests",
			transport,
			&pendingHeap{
				nextRand: nextRandFn,
			},
			plOpts...,
		),
	}
}

// List is a peer list which rotates which peers are to be selected in a circle
type List struct {
	*abstractlist.List
}

// nextRand is a convenience function for creating a new rand.Rand from a given
// seed.
func nextRand(seed int64) func(int) int {
	r := rand.New(rand.NewSource(seed))

	return func(numPeers int) int {
		if numPeers == 0 {
			return 0
		}
		return r.Intn(numPeers)
	}
}
