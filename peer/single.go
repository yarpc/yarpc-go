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

package peer

import (
	"context"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
)

// Single implements the Chooser interface for a single peer
type Single struct {
	p   peer.Peer
	err error
}

// NewSingle creates a static Chooser with a single Peer
func NewSingle(pid peer.Identifier, transport peer.Transport) *Single {
	s := &Single{}
	p, err := transport.RetainPeer(pid, s)
	s.p = p
	s.err = err
	return s
}

// Choose returns the single peer
func (s *Single) Choose(context.Context, *transport.Request) (peer.Peer, func(error), error) {
	s.p.StartRequest()
	return s.p, s.onFinish, s.err
}

func (s *Single) onFinish(_ error) {
	s.p.EndRequest()
}

// NotifyStatusChanged receives notifications from the transport when the peer
// connects, disconnects, accepts a request, and so on.
func (s *Single) NotifyStatusChanged(_ peer.Identifier) {
}

// Start is a noop
func (s *Single) Start() error {
	return nil
}

// Stop is a noop
func (s *Single) Stop() error {
	return nil
}

// IsRunning is a noop
func (s *Single) IsRunning() bool {
	return true
}
