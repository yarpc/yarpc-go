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

package list

import (
	"context"
	"sync"

	yerrors "go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/transport"

	"go.uber.org/atomic"
)

// NewRoundRobin creates a new round robin PeerList using
func NewRoundRobin(peerIDs []peer.Identifier, agent peer.Agent) (*RoundRobin, error) {
	rr := &RoundRobin{
		unavailablePeers:   make(map[string]peer.Peer, len(peerIDs)),
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

	unavailablePeers   map[string]peer.Peer
	availablePeerRing  *PeerRing
	peerAvailableEvent chan struct{}
	agent              peer.Agent
	started            atomic.Bool
}

func (pl *RoundRobin) addAll(peerIDs []peer.Identifier) error {
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
func (pl *RoundRobin) Add(pid peer.Identifier) error {
	pl.lock.Lock()
	err := pl.addPeerIdentifier(pid)
	pl.lock.Unlock()
	return err
}

// Must be run inside a mutex.Lock()
func (pl *RoundRobin) addPeerIdentifier(pid peer.Identifier) error {
	p, err := pl.agent.RetainPeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.addPeer(p)
}

// Must be run in a mutex.Lock()
func (pl *RoundRobin) addPeer(p peer.Peer) error {
	if p.Status().ConnectionStatus != peer.Available {
		return pl.addToUnavailablePeers(p)
	}

	return pl.addToAvailablePeers(p)
}

// Must be run in a mutex.Lock()
func (pl *RoundRobin) addToUnavailablePeers(p peer.Peer) error {
	pl.unavailablePeers[p.Identifier()] = p
	return nil
}

// Must be run in a mutex.Lock()
func (pl *RoundRobin) addToAvailablePeers(p peer.Peer) error {
	if err := pl.availablePeerRing.Add(p); err != nil {
		return err
	}

	pl.notifyPeerAvailable()
	return nil
}

// Start notifies the RoundRobin that requests will start coming
func (pl *RoundRobin) Start() error {
	if pl.started.Swap(true) {
		return peer.ErrPeerListAlreadyStarted("RoundRobinList")
	}
	return nil
}

// Stop notifies the RoundRobin that requests will stop coming
func (pl *RoundRobin) Stop() error {
	if !pl.started.Swap(false) {
		return peer.ErrPeerListNotStarted("RoundRobinList")
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

// removeAllUnavailable will clear the unavailablePeers list and
// return all the Peers in the list in a slice
// Must be run in a mutex.Lock()
func (pl *RoundRobin) removeAllUnavailable() []peer.Peer {
	peers := make([]peer.Peer, 0, len(pl.unavailablePeers))
	for id, p := range pl.unavailablePeers {
		peers = append(peers, p)
		delete(pl.unavailablePeers, id)
	}
	return peers
}

// releaseAll will iterate through a list of peers and call release
// on the agent
func (pl *RoundRobin) releaseAll(peers []peer.Peer) []error {
	var errs []error
	for _, p := range peers {
		if err := pl.agent.ReleasePeer(p, pl); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// Remove a peer identifier from the round robin
func (pl *RoundRobin) Remove(pid peer.Identifier) error {
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
func (pl *RoundRobin) removeByPeerIdentifier(pid peer.Identifier) error {
	if p := pl.availablePeerRing.GetPeer(pid); p != nil {
		return pl.availablePeerRing.Remove(p)
	}

	if p, ok := pl.unavailablePeers[pid.Identifier()]; ok && p != nil {
		pl.removeFromUnavailablePeers(p)
		return nil
	}

	return peer.ErrPeerRemoveNotInList(pid.Identifier())
}

// removeFromUnavailablePeers remove a peer from the Unavailable Peers list
// the Peer should already be validated as non-nil and in the Unavailable list
// Must be run in a mutex.Lock()
func (pl *RoundRobin) removeFromUnavailablePeers(p peer.Peer) {
	delete(pl.unavailablePeers, p.Identifier())
}

// ChoosePeer selects the next available peer in the round robin
func (pl *RoundRobin) ChoosePeer(ctx context.Context, req *transport.Request) (peer.Peer, error) {
	if !pl.started.Load() {
		return nil, peer.ErrPeerListNotStarted("RoundRobinList")
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
func (pl *RoundRobin) nextPeer() peer.Peer {
	pl.lock.Lock()
	p := pl.availablePeerRing.Next()
	pl.lock.Unlock()
	return p
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
		return peer.ErrChooseContextHasNoDeadline("RoundRobinList")
	}

	select {
	case <-pl.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NotifyStatusChanged when the peer's status changes
func (pl *RoundRobin) NotifyStatusChanged(pid peer.Identifier) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if p := pl.availablePeerRing.GetPeer(pid); p != nil {
		pl.handleAvailablePeerStatusChange(p)
		return
	}

	if p, ok := pl.unavailablePeers[pid.Identifier()]; ok && p != nil {
		pl.handleUnavailablePeerStatusChange(p)
		return
	}
	// No action required
}

// handleAvailablePeerStatusChange checks the connection status of a connected peer to potentially
// move that Peer from the PeerRing to the unavailable peer map
// Must be run in a mutex.Lock()
func (pl *RoundRobin) handleAvailablePeerStatusChange(p peer.Peer) error {
	if p.Status().ConnectionStatus == peer.Available {
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
// move that Peer from the unavailablePeerMap into the available Peer Ring
// Must be run in a mutex.Lock()
func (pl *RoundRobin) handleUnavailablePeerStatusChange(p peer.Peer) error {
	if p.Status().ConnectionStatus != peer.Available {
		// Peer is in the proper pool, ignore
		return nil
	}

	pl.removeFromUnavailablePeers(p)

	return pl.addToAvailablePeers(p)
}
