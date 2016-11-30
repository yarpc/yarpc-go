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

package single

import (
	"context"

	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/transport"
)

// Single implements the peer.Chooser interface for a single peer
type Single struct {
	p   peer.Peer
	err error
}

// New creates a static peer.Chooser with a single Peer
func New(pid peer.Identifier, agent peer.Agent) *Single {
	s := &Single{}
	p, err := agent.RetainPeer(pid, s)
	s.p = p
	s.err = err
	return s
}

// ChoosePeer returns the single peer
func (s *Single) ChoosePeer(context.Context, *transport.Request) (peer.Peer, error) {
	return s.p, s.err
}

// NotifyStatusChanged when the Peer status changes
func (s *Single) NotifyStatusChanged(peer.Identifier) {}

// Start is a noop
func (s *Single) Start() error {
	// TODO deprecated
	return nil
}

// Stop is a noop
func (s *Single) Stop() error {
	// TODO deprecated
	return nil
}
