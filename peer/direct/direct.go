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

package direct

import (
	"context"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

var _ peer.Chooser = (*Chooser)(nil)

type listConfig struct{}

// ListOption customizes the behavior of the peer chooser.
type ListOption func(*listConfig)

// New creates a new direct peer chooser.
func New(cfg Configuration, transport peer.Transport, opts ...ListOption) (*Chooser, error) {
	if transport == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("%q chooser requires non-nil transport", name)
	}
	return &Chooser{transport: transport}, nil
}

// Chooser is a peer.Chooser that returns the peer identified by
// transport.Request#ShardKey, suitable for directly addressing a peer.
type Chooser struct {
	transport peer.Transport
}

// Start statisfies the peer.Chooser interface.
func (c *Chooser) Start() error {
	return nil // no-op
}

// Stop statisfies the peer.Chooser interface.
func (c *Chooser) Stop() error {
	return nil // no-op
}

// IsRunning statisfies the peer.Chooser interface.
func (c *Chooser) IsRunning() bool {
	return true // no-op
}

// Choose uses the peer identifier set as the transport.Request#ShardKey to
// return the peer.
func (c *Chooser) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	if req.ShardKey == "" {
		return nil, nil,
			yarpcerrors.InvalidArgumentErrorf("%q chooser requires ShardKey to be non-empty", name)
	}

	id := newPeerIdentifier(req.ShardKey)
	sub := newPeerSubscriber()

	transportPeer, err := c.transport.RetainPeer(id, sub)
	if err != nil {
		return nil, nil, err
	}

	onFinish := func(error) {
		_ = c.transport.ReleasePeer(transportPeer, sub)
	}
	return transportPeer, onFinish, nil
}
