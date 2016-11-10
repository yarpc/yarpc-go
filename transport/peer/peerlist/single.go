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

package peerlist

import (
	"context"
	"sync"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"
)

type single struct {
	lock sync.RWMutex

	initialPeerID transport.PeerIdentifier
	peer          transport.Peer
	agent         transport.Agent
	started       bool
}

// NewSingle creates a static PeerList with a single Peer
func NewSingle(pid transport.PeerIdentifier, agent transport.Agent) transport.PeerList {
	return &single{
		initialPeerID: pid,
		agent:         agent,
		started:       false,
	}
}

func (pl *single) Start() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()
	if pl.started {
		return errors.ErrPeerListAlreadyStarted("single")
	}
	pl.started = true

	peer, err := pl.agent.RetainPeer(pl.initialPeerID, pl)
	if err != nil {
		pl.started = false
		return err
	}
	pl.peer = peer
	return nil
}

func (pl *single) Stop() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if !pl.started {
		return errors.ErrPeerListNotStarted("single")
	}
	pl.started = false

	err := pl.agent.ReleasePeer(pl.initialPeerID, pl)
	if err != nil {
		return err
	}

	pl.peer = nil
	return nil
}

func (pl *single) ChoosePeer(context.Context, *transport.Request) (transport.Peer, error) {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	if !pl.started {
		return nil, errors.ErrPeerListNotStarted("single")
	}
	return pl.peer, nil
}

// NotifyStatusChanged when the Peer status changes
func (pl *single) NotifyStatusChanged(transport.Peer) {}
