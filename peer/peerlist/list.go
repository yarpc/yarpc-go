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

package peerlist

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/atomic"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
)

var (
	_noContextDeadlineError = "can't wait for peer without a context deadline for a %s peer list"
)

type listConfig struct {
	capacity int
}

var defaultListConfig = listConfig{
	capacity: 10,
}

// ListOption customizes the behavior of a list.
type ListOption func(*listConfig)

// Capacity specifies the default capacity of the underlying
// data structures for this list
//
// Defaults to 10.
func Capacity(capacity int) ListOption {
	return func(c *listConfig) {
		c.capacity = capacity
	}
}

// New creates a new peer list with an identifier chooser for available peers.
func New(name string, transport peer.Transport, identifierChooser peer.IdentifierChooserList, opts ...ListOption) *List {
	cfg := defaultListConfig
	for _, o := range opts {
		o(&cfg)
	}

	return &List{
		once:               lifecycle.NewOnce(),
		name:               name,
		uninitializedPeers: make(map[string]peer.Identifier, cfg.capacity),
		unavailablePeers:   make(map[string]peer.Peer, cfg.capacity),
		availablePeers:     make(map[string]peer.Peer, cfg.capacity),
		identifierChooser:  identifierChooser,
		transport:          transport,
		peerAvailableEvent: make(chan struct{}, 1),
	}
}

// List is a peer list which rotates which peers are to be selected in a circle
type List struct {
	lock sync.Mutex

	name string

	shouldRetainPeers  atomic.Bool
	uninitializedPeers map[string]peer.Identifier

	unavailablePeers   map[string]peer.Peer
	availablePeers     map[string]peer.Peer
	identifierChooser  peer.IdentifierChooserList
	peerAvailableEvent chan struct{}
	transport          peer.Transport

	once *lifecycle.Once
}

// Update applies the additions and removals of peer Identifiers to the list
// it returns a multi-error result of every failure that happened without
// circuit breaking due to failures.
func (pl *List) Update(updates peer.ListUpdates) error {
	if len(updates.Additions) == 0 && len(updates.Removals) == 0 {
		return nil
	}

	pl.lock.Lock()
	defer pl.lock.Unlock()

	if pl.shouldRetainPeers.Load() {
		return pl.updateInitialized(updates)
	}
	return pl.updateUninitialized(updates)
}

// updateInitialized applies peer list updates when the peer list
// is able to retain peers, putting the updates into the available
// or unavailable containers.
//
// Must be run inside a mutex.Lock()
func (pl *List) updateInitialized(updates peer.ListUpdates) error {
	var errs error
	for _, peerID := range updates.Removals {
		errs = multierr.Append(errs, pl.removePeerIdentifier(peerID))
	}

	for _, peerID := range updates.Additions {
		errs = multierr.Append(errs, pl.addPeerIdentifier(peerID))
	}
	return errs
}

// updateUninitialized applies peer list updates when the peer list
// is **not** able to retain peers, putting the updates into a single
// uninitialized peer list.
//
// Must be run inside a mutex.Lock()
func (pl *List) updateUninitialized(updates peer.ListUpdates) error {
	var errs error
	for _, peerID := range updates.Removals {
		if _, ok := pl.uninitializedPeers[peerID.Identifier()]; ok {
			delete(pl.uninitializedPeers, peerID.Identifier())
		} else {
			errs = multierr.Append(errs, peer.ErrPeerRemoveNotInList(peerID.Identifier()))
		}
	}
	for _, peerID := range updates.Additions {
		pl.uninitializedPeers[peerID.Identifier()] = peerID
	}

	return errs
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
	if err := pl.identifierChooser.Update(peer.ListUpdates{
		Additions: []peer.Identifier{p},
	}); err != nil {
		return err
	}

	pl.availablePeers[p.Identifier()] = p
	pl.notifyPeerAvailable()
	return nil
}

// Start notifies the List that requests will start coming
func (pl *List) Start() error {
	return pl.once.Start(pl.start)
}

func (pl *List) start() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	var errs error
	for k, pid := range pl.uninitializedPeers {
		errs = multierr.Append(errs, pl.addPeerIdentifier(pid))
		delete(pl.uninitializedPeers, k)
	}

	pl.shouldRetainPeers.Store(true)

	return errs
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

	availablePeers := pl.removeAllAvailablePeers(pl.availablePeers)
	errs = append(errs, pl.releaseAll(availablePeers)...)
	pl.addToUninitialized(availablePeers)

	unvavailablePeers := pl.removeAllUnavailablePeers(pl.unavailablePeers)
	errs = append(errs, pl.releaseAll(unvavailablePeers)...)
	pl.addToUninitialized(unvavailablePeers)

	pl.shouldRetainPeers.Store(false)

	return multierr.Combine(errs...)
}

func (pl *List) addToUninitialized(peers []peer.Peer) {
	for _, p := range peers {
		pl.uninitializedPeers[p.Identifier()] = p
	}
}

// removeAllAvailablePeers will clear the availablePeers list and return all
// the Peers in the list in a slice
// Must be run in a mutex.Lock()
func (pl *List) removeAllAvailablePeers(toRemove map[string]peer.Peer) []peer.Peer {
	peers := make([]peer.Peer, 0, len(toRemove))
	for id, p := range toRemove {
		peers = append(peers, p)
		delete(pl.availablePeers, id)
		_ = pl.identifierChooser.Update(peer.ListUpdates{
			Removals: []peer.Identifier{p},
		})
	}
	return peers
}

// removeAllUnavailablePeers will clear the unavailablePeers list and
// return all the Peers in the list in a slice
// Must be run in a mutex.Lock()
func (pl *List) removeAllUnavailablePeers(toRemove map[string]peer.Peer) []peer.Peer {
	peers := make([]peer.Peer, 0, len(toRemove))
	for id, p := range toRemove {
		peers = append(peers, p)
		delete(toRemove, id)
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
	if p := pl.availablePeers[pid.Identifier()]; p != nil {
		return pl.removeFromAvailablePeers(p)
	}

	if p, ok := pl.unavailablePeers[pid.Identifier()]; ok && p != nil {
		pl.removeFromUnavailablePeers(p)
		return nil
	}

	return peer.ErrPeerRemoveNotInList(pid.Identifier())
}

// removeFromAvailablePeers remove a peer from the Available Peers list the
// Peer should already be validated as non-nil and in the Available list.
// Must be run in a mutex.Lock()
func (pl *List) removeFromAvailablePeers(p peer.Peer) error {
	delete(pl.availablePeers, p.Identifier())
	return pl.identifierChooser.Update(peer.ListUpdates{
		Removals: []peer.Identifier{p},
	})
}

// removeFromUnavailablePeers remove a peer from the Unavailable Peers list the
// Peer should already be validated as non-nil and in the Unavailable list.
// Must be run in a mutex.Lock()
func (pl *List) removeFromUnavailablePeers(p peer.Peer) {
	delete(pl.unavailablePeers, p.Identifier())
}

// Choose selects the next available peer in the peer list
func (pl *List) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	if err := pl.once.WaitUntilRunning(ctx); err != nil {
		return nil, nil, pl.newNotRunningError(err)
	}

	for {
		if nextPID := pl.choose(ctx, req); nextPID != nil {
			nextPeer := pl.availablePeers[nextPID.Identifier()]
			pl.notifyPeerAvailable()
			nextPeer.StartRequest()
			return nextPeer, pl.getOnFinishFunc(nextPeer), nil
		}

		if err := pl.waitForPeerAddedEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
}

func (pl *List) newNotRunningError(err error) error {
	return yarpcerrors.FailedPreconditionErrorf("%s peer list is not running: %s", pl.name, err.Error())
}

// IsRunning returns whether the peer list is running.
func (pl *List) IsRunning() bool {
	return pl.once.IsRunning()
}

// choose grabs the next available peer from the PeerRing and returns it,
// if there are no available peers it returns nil
func (pl *List) choose(ctx context.Context, req *transport.Request) peer.Identifier {
	pl.lock.Lock()
	pid := pl.identifierChooser.Choose(ctx, req)
	pl.lock.Unlock()
	return pid
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
		return pl.newNoContextDeadlineError()
	}

	select {
	case <-pl.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return pl.newUnavailableError(ctx.Err())
	}
}

func (pl *List) newNoContextDeadlineError() error {
	return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, _noContextDeadlineError, pl.name)
}

func (pl *List) newUnavailableError(err error) error {
	return yarpcerrors.UnavailableErrorf("%s peer list timed out waiting for peer: %s", pl.name, err.Error())
}

// NotifyStatusChanged when the peer's status changes
func (pl *List) NotifyStatusChanged(pid peer.Identifier) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if p := pl.availablePeers[pid.Identifier()]; p != nil {
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

	if err := pl.identifierChooser.Update(peer.ListUpdates{
		Removals: []peer.Identifier{p},
	}); err != nil {
		// Peer was not in list
		return err
	}
	delete(pl.availablePeers, p.Identifier())

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

// Available returns whether the identifier peer is available for traffic.
func (pl *List) Available(p peer.Identifier) bool {
	_, ok := pl.availablePeers[p.Identifier()]
	return ok
}

// Uninitialized returns whether a peer is waiting for the peer list to start.
func (pl *List) Uninitialized(p peer.Identifier) bool {
	_, ok := pl.uninitializedPeers[p.Identifier()]
	return ok
}

// NumAvailable returns how many peers are available.
func (pl *List) NumAvailable() int {
	return len(pl.availablePeers)
}

// NumUnavailable returns how many peers are unavailable.
func (pl *List) NumUnavailable() int {
	return len(pl.unavailablePeers)
}

// NumUninitialized returns how many peers are unavailable.
func (pl *List) NumUninitialized() int {
	return len(pl.uninitializedPeers)
}

// Introspect returns a ChooserStatus with a summary of the Peers.
func (pl *List) Introspect() introspection.ChooserStatus {
	state := "Stopped"
	if pl.IsRunning() {
		state = "Running"
	}

	pl.lock.Lock()
	availables := make([]peer.Peer, 0, len(pl.availablePeers))
	for _, peer := range pl.availablePeers {
		availables = append(availables, peer)
	}
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
