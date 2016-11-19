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

package roundrobin

import (
	"context"

	yerrors "go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"

	"go.uber.org/atomic"
)

// New creates a new round robin PeerList using
func New(peerIDs []transport.PeerIdentifier, agent transport.Agent) (*RoundRobin, error) {
	rr := &RoundRobin{
		pr:    NewPeerRing(len(peerIDs)),
		agent: agent,
	}

	err := rr.addAll(peerIDs)
	return rr, err
}

// RoundRobin is a PeerList which rotates which peers are to be selected in a circle
type RoundRobin struct {
	pr      *PeerRing
	agent   transport.Agent
	started atomic.Bool
}

func (pl *RoundRobin) addAll(peerIDs []transport.PeerIdentifier) error {
	var errs []error

	for _, peerID := range peerIDs {
		if err := pl.addPeer(peerID); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return yerrors.MultiError(errs)
}

// Start notifies the RoundRobin that requests will start coming
func (pl *RoundRobin) Start() error {
	if pl.started.Swap(true) {
		return errors.ErrPeerListAlreadyStarted("RoundRobinList")
	}
	return nil
}

// Stop notifies the RoundRobin that requests will stop coming
func (pl *RoundRobin) Stop() error {
	if !pl.started.Swap(false) {
		return errors.ErrPeerListNotStarted("RoundRobinList")
	}
	return pl.clearPeers()
}

func (pl *RoundRobin) clearPeers() error {
	var errs []error

	peers := pl.pr.RemoveAll()
	for _, p := range peers {
		if err := pl.agent.ReleasePeer(p, pl); err != nil {
			errs = append(errs, err)
		}
	}

	return yerrors.MultiError(errs)
}

// ChoosePeer selects the next available peer in the round robin
func (pl *RoundRobin) ChoosePeer(context.Context, *transport.Request) (transport.Peer, error) {
	if !pl.started.Load() {
		return nil, errors.ErrPeerListNotStarted("RoundRobinList")
	}

	nextPeer := pl.pr.Next()
	if nextPeer == nil {
		return nil, errors.ErrNoPeerToSelect("RoundRobinList")
	}
	return nextPeer, nil
}

// Add a peer identifier to the round robin
func (pl *RoundRobin) Add(pid transport.PeerIdentifier) error {
	return pl.addPeer(pid)
}

func (pl *RoundRobin) addPeer(pid transport.PeerIdentifier) error {
	p, err := pl.agent.RetainPeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.pr.Add(p)
}

// Remove a peer identifier from the round robin
func (pl *RoundRobin) Remove(pid transport.PeerIdentifier) error {
	if err := pl.pr.Remove(pid); err != nil {
		// The peer has already been removed
		return err
	}

	return pl.agent.ReleasePeer(pid, pl)
}

// NotifyStatusChanged when the peer's status changes
func (pl *RoundRobin) NotifyStatusChanged(transport.Peer) {}
