// Copyright (c) 2021 Uber Technologies, Inc.
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
	"fmt"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

var _ peer.Chooser = (*Chooser)(nil)

type chooserOptions struct {
	logger *zap.Logger
}

// ChooserOption customizes the behavior of the peer chooser.
type ChooserOption func(*chooserOptions)

// Logger sets the logger for the chooser.
func Logger(logger *zap.Logger) ChooserOption {
	return func(c *chooserOptions) {
		c.logger = logger
	}
}

// New creates a new direct peer chooser.
func New(cfg Configuration, transport peer.Transport, opts ...ChooserOption) (*Chooser, error) {
	options := &chooserOptions{
		logger: zap.NewNop(),
	}
	for _, opt := range opts {
		opt(options)
	}

	if transport == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("%q chooser requires non-nil transport", name)
	}
	return &Chooser{
		transport: transport,
		logger:    options.logger,
	}, nil
}

// Chooser is a peer.Chooser that returns the peer identified by
// transport.Request#ShardKey, suitable for directly addressing a peer.
type Chooser struct {
	transport peer.Transport
	logger    *zap.Logger
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

	id := hostport.Identify(req.ShardKey)
	sub := &peerSubscriber{
		peerIdentifier: id,
	}

	transportPeer, err := c.transport.RetainPeer(id, sub)
	if err != nil {
		return nil, nil, err
	}

	onFinish := func(error) {
		if err := c.transport.ReleasePeer(transportPeer, sub); err != nil {
			c.logger.Error(
				fmt.Sprintf("error releasing peer from transport in %q chooser", name),
				zap.Error(err))
		}
	}
	return transportPeer, onFinish, nil
}

type peerSubscriber struct {
	peerIdentifier peer.Identifier
}

func (d *peerSubscriber) NotifyStatusChanged(_ peer.Identifier) {}
