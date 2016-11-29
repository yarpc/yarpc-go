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
	"sync"

	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/transport"
)

type single struct {
	lock sync.RWMutex

	initialPeerID peer.Identifier
	p             peer.Peer
	agent         peer.Agent
	started       bool
}

// New creates a static peer.Chooser with a single Peer
func New(pid peer.Identifier, agent peer.Agent) peer.Chooser {
	return &single{
		initialPeerID: pid,
		agent:         agent,
		started:       false,
	}
}

func (s *single) Start() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.started {
		return peer.ErrPeerListAlreadyStarted("single")
	}
	s.started = true

	p, err := s.agent.RetainPeer(s.initialPeerID, s)
	if err != nil {
		s.started = false
		return err
	}
	s.p = p
	return nil
}

func (s *single) Stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.started {
		return peer.ErrPeerListNotStarted("single")
	}
	s.started = false

	err := s.agent.ReleasePeer(s.initialPeerID, s)
	if err != nil {
		return err
	}

	s.p = nil
	return nil
}

func (s *single) ChoosePeer(context.Context, *transport.Request) (peer.Peer, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if !s.started {
		return nil, peer.ErrPeerListNotStarted("single")
	}
	return s.p, nil
}

// NotifyStatusChanged when the Peer status changes
func (s *single) NotifyStatusChanged(peer.Identifier) {}
