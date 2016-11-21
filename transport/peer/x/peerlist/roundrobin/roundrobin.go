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
		nonAvailablePeers:  make(map[string]transport.Peer, len(peerIDs)),
		availablePeerRing:  NewPeerRing(len(peerIDs)),
		agent:              agent,
		peerAvailableEvent: make(chan struct{}, 1),
	}

	err := rr.addAll(peerIDs)
	return rr, err
}

// RoundRobin is a PeerList which rotates which peers are to be selected in a circle
type RoundRobin struct {
	lock sync.Mutex

	nonAvailablePeers  map[string]transport.Peer
	availablePeerRing  *PeerRing
	peerAvailableEvent chan struct{}
	agent              transport.Agent
	started            atomic.Bool
}

func (pl *RoundRobin) addAll(peerIDs []transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	var errs []error

	for _, peerID := range peerIDs {
		if err := pl.addPeerIdentifier(peerID); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return yerrors.MultiError(errs)
}

// Add a peer identifier to the round robin
func (pl *RoundRobin) Add(pid transport.PeerIdentifier) error {
	pl.lock.Lock()
	err := pl.addPeerIdentifier(pid)
	pl.lock.Unlock()
	return err
}

// Must be run inside a mutex.Lock()
func (pl *RoundRobin) addPeerIdentifier(pid transport.PeerIdentifier) error {
	p, err := pl.agent.RetainPeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.addPeer(p)
}

// Must be run in a mutex.Lock()
func (pl *RoundRobin) addPeer(p transport.Peer) error {
	if p.Status().ConnectionStatus != transport.PeerAvailable {
		return pl.addToUnavailablePeers(p)
	}

	return pl.addToAvailablePeers(p)
}

// Must be run in a mutex.Lock()
func (pl *RoundRobin) addToUnavailablePeers(p transport.Peer) error {
	pl.nonAvailablePeers[p.Identifier()] = p
	return nil
}

// Must be run in a mutex.Lock()
func (pl *RoundRobin) addToAvailablePeers(p transport.Peer) error {
	if err := pl.availablePeerRing.Add(p); err != nil {
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

	availablePeers := pl.availablePeerRing.RemoveAll()
	errs = append(errs, pl.releaseAll(availablePeers)...)

	unvavailablePeers := pl.removeAllUnavailable()
	errs = append(errs, pl.releaseAll(unvavailablePeers)...)

	return yerrors.MultiError(errs)
}

// removeAllUnavailable will clear the nonAvailablePeers list and
// return all the Peers in the list in a slice
// Must be run in a mutex.Lock()
func (pl *RoundRobin) removeAllUnavailable() []transport.Peer {
	peers := make([]transport.Peer, 0, len(pl.nonAvailablePeers))
	for id, peer := range pl.nonAvailablePeers {
		peers = append(peers, peer)
		delete(pl.nonAvailablePeers, id)
	}
	return peers
}

// releaseAll will iterate through a list of peers and call release
// on the agent
func (pl *RoundRobin) releaseAll(peers []transport.Peer) []error {
	var errs []error
	for _, p := range peers {
		if err := pl.agent.ReleasePeer(p, pl); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// Remove a peer identifier from the round robin
func (pl *RoundRobin) Remove(pid transport.PeerIdentifier) error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if err := pl.removeByPeerIdentifier(pid); err != nil {
		// The peer has already been removed
		return err
	}

	return pl.agent.ReleasePeer(pid, pl)
}

// removeByPeerIdentifier will search through the Available and Unavailable Peers
// for the PeerID and remove it
// Must be run in a mutex.Lock()
func (pl *RoundRobin) removeByPeerIdentifier(pid transport.PeerIdentifier) error {
	if peer := pl.availablePeerRing.GetPeer(pid); peer != nil {
		return pl.availablePeerRing.Remove(peer)
	}

	if peer, ok := pl.nonAvailablePeers[pid.Identifier()]; ok && peer != nil {
		pl.removeFromUnavailablePeers(peer)
		return nil
	}

	return errors.ErrPeerRemoveNotInList(pid.Identifier())
}

// removeFromUnavailablePeers remove a peer from the Unavailable Peers list
// the Peer should already be validated as non-nil and in the Unavailable list
// Must be run in a mutex.Lock()
func (pl *RoundRobin) removeFromUnavailablePeers(p transport.Peer) {
	delete(pl.nonAvailablePeers, p.Identifier())
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

// nextPeer grabs the next available peer from the PeerRing and returns it,
// if there are no available peers it returns nil
func (pl *RoundRobin) nextPeer() transport.Peer {
	pl.lock.Lock()
	peer := pl.availablePeerRing.Next()
	pl.lock.Unlock()
	return peer
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
func (pl *RoundRobin) NotifyStatusChanged(pid transport.PeerIdentifier) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if peer := pl.availablePeerRing.GetPeer(pid); peer != nil {
		pl.handleAvailablePeerStatusChange(peer)
		return
	}

	if peer, ok := pl.nonAvailablePeers[pid.Identifier()]; ok && peer != nil {
		pl.handleUnavailablePeerStatusChange(peer)
		return
	}
	// No action required
}

// handleAvailablePeerStatusChange checks the connection status of a connected peer to potentially
// move that Peer from the PeerRing to the nonAvailable peer map
// Must be run in a mutex.Lock()
func (pl *RoundRobin) handleAvailablePeerStatusChange(p transport.Peer) error {
	if p.Status().ConnectionStatus == transport.PeerAvailable {
		// Peer is in the proper pool, ignore
		return nil
	}

	if err := pl.availablePeerRing.Remove(p); err != nil {
		// Peer was not in list
		return err
	}

	return pl.addToUnavailablePeers(p)

}

// handleUnavailablePeerStatusChange checks the connection status of an unavailable peer to potentially
// move that Peer from the nonAvailablePeerMap into the available Peer Ring
// Must be run in a mutex.Lock()
func (pl *RoundRobin) handleUnavailablePeerStatusChange(p transport.Peer) error {
	if p.Status().ConnectionStatus != transport.PeerAvailable {
		// Peer is in the proper pool, ignore
		return nil
	}

	pl.removeFromUnavailablePeers(p)

	return pl.addToAvailablePeers(p)
}
