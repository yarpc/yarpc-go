// Copyright (c) 2017 Uber Technologies, Inc.
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
	"fmt"
	"sync"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	yerrors "go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/introspection"
	ysync "go.uber.org/yarpc/internal/sync"
)

type listConfig struct {
	startupWait time.Duration
	capacity    int
}

var defaultListConfig = listConfig{
	startupWait: 5 * time.Second,
	capacity:    10,
}

// ListOption customizes the behavior of a roundrobin list.
type ListOption func(*listConfig)

// StartupWait specifies how long updates to the list will wait
// before the list has been started
//
// Defaults to 5 seconds.
func StartupWait(t time.Duration) ListOption {
	return func(c *listConfig) {
		c.startupWait = t
	}
}

// Capacity specifies the default capacity of the underlying
// data structures for this list
//
// Defaults to 10.
func Capacity(capacity int) ListOption {
	return func(c *listConfig) {
		c.capacity = capacity
	}
}

// New creates a new round robin PeerList
func New(transport peer.Transport, opts ...ListOption) *List {
	cfg := defaultListConfig
	for _, o := range opts {
		o(&cfg)
	}

	return &List{
		once:               ysync.Once(),
		unavailablePeers:   make(map[string]peer.Peer, cfg.capacity),
		availablePeerRing:  NewPeerRing(cfg.capacity),
		transport:          transport,
		peerAvailableEvent: make(chan struct{}, 1),
		startupWait:        cfg.startupWait,
	}
}

// List is a PeerList which rotates which peers are to be selected in a circle
type List struct {
	lock sync.Mutex

	unavailablePeers   map[string]peer.Peer
	availablePeerRing  *PeerRing
	peerAvailableEvent chan struct{}
	transport          peer.Transport
	startupWait        time.Duration

	once ysync.LifecycleOnce
}

// Update applies the additions and removals of peer Identifiers to the list
// it returns a multi-error result of every failure that happened without
// circuit breaking due to failures
func (pl *List) Update(updates peer.ListUpdates) error {
	// Wait for the list to be running before we accept updates.
	ctx, cancel := context.WithTimeout(context.Background(), pl.startupWait)
	defer cancel()
	if err := pl.once.WhenRunning(ctx); err != nil {
		return err
	}

	additions := updates.Additions
	removals := updates.Removals

	if len(additions) == 0 && len(removals) == 0 {
		return nil
	}

	pl.lock.Lock()
	defer pl.lock.Unlock()

	var errs []error

	for _, peerID := range removals {
		if err := pl.removePeerIdentifier(peerID); err != nil {
			errs = append(errs, err)
		}
	}

	for _, peerID := range additions {
		if err := pl.addPeerIdentifier(peerID); err != nil {
			errs = append(errs, err)
		}
	}

	return yerrors.MultiError(errs)
}

// Must be run inside a mutex.Lock()
func (pl *List) addPeerIdentifier(pid peer.Identifier) error {
	p, err := pl.transport.RetainPeer(pid, pl)
	if err != nil {
		return err
	}

	return pl.addPeer(p)
}

// Must be run in a mutex.Lock()
func (pl *List) addPeer(p peer.Peer) error {
	if p.Status().ConnectionStatus != peer.Available {
		return pl.addToUnavailablePeers(p)
	}

	return pl.addToAvailablePeers(p)
}

// Must be run in a mutex.Lock()
func (pl *List) addToUnavailablePeers(p peer.Peer) error {
	pl.unavailablePeers[p.Identifier()] = p
	return nil
}

// Must be run in a mutex.Lock()
func (pl *List) addToAvailablePeers(p peer.Peer) error {
	if err := pl.availablePeerRing.Add(p); err != nil {
		return err
	}

	pl.notifyPeerAvailable()
	return nil
}

// Start notifies the List that requests will start coming
func (pl *List) Start() error {
	return pl.once.Start(nil)
}

// Stop notifies the List that requests will stop coming
func (pl *List) Stop() error {
	return pl.once.Stop(pl.clearPeers)
}

// clearPeers will release all the peers from the list
func (pl *List) clearPeers() error {
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
func (pl *List) removeAllUnavailable() []peer.Peer {
	peers := make([]peer.Peer, 0, len(pl.unavailablePeers))
	for id, p := range pl.unavailablePeers {
		peers = append(peers, p)
		delete(pl.unavailablePeers, id)
	}
	return peers
}

// releaseAll will iterate through a list of peers and call release
// on the transport
func (pl *List) releaseAll(peers []peer.Peer) []error {
	var errs []error
	for _, p := range peers {
		if err := pl.transport.ReleasePeer(p, pl); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// removePeerIdentifier will go remove references to the peer identifier and release
// it from the transport
// Must be run in a mutex.Lock()
func (pl *List) removePeerIdentifier(pid peer.Identifier) error {
	if err := pl.removePeerIdentifierReferences(pid); err != nil {
		// The peer has already been removed
		return err
	}

	return pl.transport.ReleasePeer(pid, pl)
}

// removePeerIdentifierReferences will search through the Available and Unavailable Peers
// for the PeerID and remove it
// Must be run in a mutex.Lock()
func (pl *List) removePeerIdentifierReferences(pid peer.Identifier) error {
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
func (pl *List) removeFromUnavailablePeers(p peer.Peer) {
	delete(pl.unavailablePeers, p.Identifier())
}

// Choose selects the next available peer in the round robin
func (pl *List) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	if err := pl.once.WhenRunning(ctx); err != nil {
		return nil, nil, err
	}

	for {
		if nextPeer := pl.nextPeer(); nextPeer != nil {
			pl.notifyPeerAvailable()
			nextPeer.StartRequest()
			return nextPeer, pl.getOnFinishFunc(nextPeer), nil
		}

		if err := pl.waitForPeerAddedEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
}

// IsRunning returns whether the peer list is running.
func (pl *List) IsRunning() bool {
	return pl.once.IsRunning()
}

// nextPeer grabs the next available peer from the PeerRing and returns it,
// if there are no available peers it returns nil
func (pl *List) nextPeer() peer.Peer {
	pl.lock.Lock()
	p := pl.availablePeerRing.Next()
	pl.lock.Unlock()
	return p
}

// notifyPeerAvailable writes to a channel indicating that a Peer is currently
// available for requests
func (pl *List) notifyPeerAvailable() {
	select {
	case pl.peerAvailableEvent <- struct{}{}:
	default:
	}
}

// getOnFinishFunc creates a closure that will be run at the end of the request
func (pl *List) getOnFinishFunc(p peer.Peer) func(error) {
	return func(_ error) {
		p.EndRequest()
	}
}

// waitForPeerAddedEvent waits until a peer is added to the peer list or the
// given context finishes.
// Must NOT be run in a mutex.Lock()
func (pl *List) waitForPeerAddedEvent(ctx context.Context) error {
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
func (pl *List) NotifyStatusChanged(pid peer.Identifier) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if p := pl.availablePeerRing.GetPeer(pid); p != nil {
		// TODO: log error
		_ = pl.handleAvailablePeerStatusChange(p)
		return
	}

	if p, ok := pl.unavailablePeers[pid.Identifier()]; ok && p != nil {
		// TODO: log error
		_ = pl.handleUnavailablePeerStatusChange(p)
	}
	// No action required
}

// handleAvailablePeerStatusChange checks the connection status of a connected peer to potentially
// move that Peer from the PeerRing to the unavailable peer map
// Must be run in a mutex.Lock()
func (pl *List) handleAvailablePeerStatusChange(p peer.Peer) error {
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
func (pl *List) handleUnavailablePeerStatusChange(p peer.Peer) error {
	if p.Status().ConnectionStatus != peer.Available {
		// Peer is in the proper pool, ignore
		return nil
	}

	pl.removeFromUnavailablePeers(p)

	return pl.addToAvailablePeers(p)
}

// Introspect returns a ChooserStatus with a summary of the Peers.
func (pl *List) Introspect() introspection.ChooserStatus {
	state := "Stopped"
	if pl.IsRunning() {
		state = "Running"
	}

	pl.lock.Lock()
	availables := pl.availablePeerRing.All()
	unavailables := make([]peer.Peer, 0, len(pl.unavailablePeers))
	for _, peer := range pl.unavailablePeers {
		unavailables = append(unavailables, peer)
	}
	pl.lock.Unlock()

	peersStatus := make([]introspection.PeerStatus, 0,
		len(availables)+len(unavailables))

	buildPeerStatus := func(peer peer.Peer) introspection.PeerStatus {
		ps := peer.Status()
		return introspection.PeerStatus{
			Identifier: peer.Identifier(),
			State: fmt.Sprintf("%s, %d pending request(s)",
				ps.ConnectionStatus.String(),
				ps.PendingRequestCount),
		}
	}

	for _, peer := range availables {
		peersStatus = append(peersStatus, buildPeerStatus(peer))
	}

	for _, peer := range unavailables {
		peersStatus = append(peersStatus, buildPeerStatus(peer))
	}

	return introspection.ChooserStatus{
		Name: "Single",
		State: fmt.Sprintf("%s (%d/%d available)", state, len(availables),
			len(availables)+len(unavailables)),
		Peers: peersStatus,
	}
}
