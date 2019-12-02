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

package roundrobin

import (
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/abstractlist"
	"go.uber.org/zap"
)

type listConfig struct {
	capacity int
	shuffle  bool
	failFast bool
	seed     int64
	logger   *zap.Logger
}

var defaultListConfig = listConfig{
	capacity: 10,
	shuffle:  true,
	seed:     time.Now().UnixNano(),
}

// ListOption customizes the behavior of a roundrobin list.
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

// Logger specifies a logger.
func Logger(logger *zap.Logger) ListOption {
	return func(c *listConfig) {
		c.logger = logger
	}
}

// New creates a new round robin peer list.
func New(transport peer.Transport, opts ...ListOption) *List {
	cfg := defaultListConfig
	for _, o := range opts {
		o(&cfg)
	}

	plOpts := []abstractlist.Option{
		abstractlist.Capacity(cfg.capacity),
		abstractlist.Seed(cfg.seed),
	}
	if cfg.logger != nil {
		plOpts = append(plOpts, abstractlist.Logger(cfg.logger))
	}
	if !cfg.shuffle {
		plOpts = append(plOpts, abstractlist.NoShuffle())
	}
	if cfg.failFast {
		plOpts = append(plOpts, abstractlist.FailFast())
	}

	return &List{
		List: abstractlist.New(
			"round-robin",
			transport,
			newPeerRing(),
			plOpts...,
		),
	}
}

// List is a PeerList which rotates which peers are to be selected in a circle
type List struct {
	*abstractlist.List
}
