// Copyright (c) 2020 Uber Technologies, Inc.
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
	"context"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
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
		list: abstractlist.New(
			"round-robin",
			transport,
			newPeerRing(),
			plOpts...,
		),
	}
}

var _ peer.List = (*List)(nil)
var _ peer.Chooser = (*List)(nil)
var _ introspection.IntrospectableChooser = (*List)(nil)

// List is a PeerList which rotates which peers are to be selected in a circle
type List struct {
	list *abstractlist.List
}

// Start causes the peer list to start.
//
// Starting will retain all peers that have been added but not removed
// the first time it is called.
//
// Start may be called any number of times and in any order in relation to Stop
// but will only cause the list to start the first time, and only if it has not
// already been stopped.
func (l *List) Start() error {
	return l.list.Start()
}

// Stop causes the peer list to stop.
//
// Stopping will release all retained peers to the underlying transport.
//
// Stop may be called any number of times and in order in relation to Start but
// will only cause the list to stop the first time, and only if it has
// previously been started.
func (l *List) Stop() error {
	return l.list.Stop()
}

// IsRunning returns whether the list has started and not yet stopped.
func (l *List) IsRunning() bool {
	return l.list.IsRunning()
}

// Choose returns a peer, suitable for sending a request.
//
// The peer is not guaranteed to be connected and available, but the peer list
// makes every attempt to ensure this and minimize the probability that a
// chosen peer will fail to carry a request.
func (l *List) Choose(ctx context.Context, req *transport.Request) (peer peer.Peer, onFinish func(error), err error) {
	return l.list.Choose(ctx, req)
}

// Update may add and remove logical peers in the list.
//
// The peer list uses a transport to obtain a physical peer for each logical
// peer.
// The transport is responsible for informing the peer list whether the peer is
// available or unavailable, but cannot guarantee that the peer will still be
// available after it is chosen.
func (l *List) Update(updates peer.ListUpdates) error {
	return l.list.Update(updates)
}

// NotifyStatusChanged forwards a status change notification to an individual
// peer in the list.
//
// This satisfies the peer.Subscriber interface and should only be used to
// send notifications in tests.
// The list's RetainPeer and ReleasePeer methods deal with an individual
// peer.Subscriber instance for each peer in the list, avoiding a map lookup.
func (l *List) NotifyStatusChanged(pid peer.Identifier) {
	l.list.NotifyStatusChanged(pid)
}

// Introspect reveals information about the list to the internal YARPC
// introspection system.
func (l *List) Introspect() introspection.ChooserStatus {
	return l.list.Introspect()
}
