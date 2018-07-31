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

package peer

import (
	"context"
	"fmt"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
	"go.uber.org/yarpc/pkg/lifecycle"
)

// Single implements the Chooser interface for a single peer
type Single struct {
	once          *lifecycle.Once
	t             peer.Transport
	pid           peer.Identifier
	p             peer.Peer
	err           error
	boundOnFinish func(error)
}

// NewSingle creates a static Chooser with a single Peer
func NewSingle(pid peer.Identifier, transport peer.Transport) *Single {
	s := &Single{
		once: lifecycle.NewOnce(),
		pid:  pid,
		t:    transport,
	}
	s.boundOnFinish = s.onFinish
	return s
}

// Transport exposes the transport for tests.
func (s *Single) Transport() peer.Transport {
	return s.t
}

// Choose returns the single peer
func (s *Single) Choose(ctx context.Context, _ *transport.Request) (peer.Peer, func(error), error) {
	if err := s.once.WaitUntilRunning(ctx); err != nil {
		return nil, nil, err
	}
	s.p.StartRequest()
	return s.p, s.boundOnFinish, s.err
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
	return s.once.Start(s.start)
}

func (s *Single) start() error {
	p, err := s.t.RetainPeer(s.pid, s)
	s.p = p
	s.err = err
	return err
}

// Stop is a noop
func (s *Single) Stop() error {
	return s.once.Stop(s.stop)
}

func (s *Single) stop() error {
	return s.t.ReleasePeer(s.pid, s)
}

// IsRunning is a noop
func (s *Single) IsRunning() bool {
	return true
}

// Introspect returns a ChooserStatus with a single PeerStatus.
func (s *Single) Introspect() introspection.ChooserStatus {
	if !s.once.IsRunning() {
		return introspection.ChooserStatus{
			Name: "Single",
			Peers: []introspection.PeerStatus{
				{
					Identifier: s.pid.Identifier(),
					State:      "uninitialized",
				},
			},
		}
	}

	peerStatus := s.p.Status()
	peer := introspection.PeerStatus{
		Identifier: s.p.Identifier(),
		State: fmt.Sprintf("%s, %d pending request(s)",
			peerStatus.ConnectionStatus.String(),
			peerStatus.PendingRequestCount),
	}

	return introspection.ChooserStatus{
		Name:  "Single",
		Peers: []introspection.PeerStatus{peer},
	}
}
