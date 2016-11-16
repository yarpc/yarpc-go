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
	"sync"

	yerrors "go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"

	"go.uber.org/atomic"
)

// New creates a new round robin PeerList using
func New(peerIDs []transport.PeerIdentifier, agent transport.Agent) (*RoundRobin, error) {
	rr := &RoundRobin{
		pr:                 NewPeerRing(len(peerIDs)),
		agent:              agent,
		peerAvailableEvent: make(chan struct{}, 1),
	}

	err := rr.addAll(peerIDs)
	return rr, err
}

// RoundRobin is a PeerList which rotates which peers are to be selected in a circle
type RoundRobin struct {
	lock sync.Mutex

	pr             *PeerRing
	peerAvailableEvent chan struct{}
	agent          transport.Agent
	started        atomic.Bool
}

func (pl *RoundRobin) addAll(peerIDs []transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	var errs []error

	for _, peerID := range peerIDs {
		if err := pl.addPeer(peerID); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return yerrors.MultiError(errs)
}

// Add a peer identifier to the round robin
func (pl *RoundRobin) Add(pid transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()
	return pl.addPeer(pid)
}

// Must be run inside a mutex.Lock()
func (pl *RoundRobin) addPeer(pid transport.PeerIdentifier) error {
	p, err := pl.agent.RetainPeer(pid, pl)
	if err != nil {
		return err
	}

	if err = pl.pr.Add(p); err != nil {
		return err
	}

	pl.notifyPeerAvailable()
	return nil
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
	pl.lock.Lock()
	defer pl.lock.Unlock()

	var errs []error

	peers := pl.pr.RemoveAll()
	for _, p := range peers {
		if err := pl.agent.ReleasePeer(p, pl); err != nil {
			errs = append(errs, err)
		}
	}

	return yerrors.MultiError(errs)
}

// Remove a peer identifier from the round robin
func (pl *RoundRobin) Remove(pid transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if err := pl.pr.Remove(pid); err != nil {
		// The peer has already been removed
		return err
	}

	return pl.agent.ReleasePeer(pid, pl)
}

// ChoosePeer selects the next available peer in the round robin
func (pl *RoundRobin) ChoosePeer(ctx context.Context, req *transport.Request) (transport.Peer, error) {
	if !pl.started.Load() {
		return nil, errors.ErrPeerListNotStarted("RoundRobinList")
	}

	for {
		if nextPeer := pl.nextPeer(); nextPeer != nil {
			pl.notifyPeerAvailable()
			return nextPeer, nil
		}

		if err := pl.waitForPeerAddedEvent(ctx); err != nil {
			return nil, err
		}
	}
}

func (pl *RoundRobin) nextPeer() transport.Peer {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	return pl.pr.Next()
}

// notifyPeerAvailable writes to a channel indicating that a Peer is currently
// available for requests
func (pl *RoundRobin) notifyPeerAvailable() {
	select {
	case pl.peerAvailableEvent <- struct{}{}:
	default:
	}
}

// waitForPeerAddedEvent waits until a peer is added to the peer list or the
// given context finishes.
// Must NOT be run in a mutex.Lock()
func (pl *RoundRobin) waitForPeerAddedEvent(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.ErrChooseContextHasNoDeadline("RoundRobinList")
	}

	select {
	case <-pl.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NotifyStatusChanged when the peer's status changes
func (pl *RoundRobin) NotifyStatusChanged(transport.Peer) {}
