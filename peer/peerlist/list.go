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
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
)

var (
	_noContextDeadlineError = "can't wait for peer without a context deadline for a %s peer list"
)

type listOptions struct {
	capacity int
}

var defaultListOptions = listOptions{
	capacity: 10,
}

// ListOption customizes the behavior of a list.
type ListOption interface {
	apply(*listOptions)
}

type listOptionFunc func(*listOptions)

func (f listOptionFunc) apply(options *listOptions) { f(options) }

// Capacity specifies the default capacity of the underlying
// data structures for this list
//
// Defaults to 10.
func Capacity(capacity int) ListOption {
	return listOptionFunc(func(options *listOptions) {
		options.capacity = capacity
	})
}

// New creates a new peer list with an identifier chooser for available peers.
func New(name string, transport peer.Transport, availableChooser peer.ListImplementation, opts ...ListOption) *List {
	options := defaultListOptions
	for _, o := range opts {
		o.apply(&options)
	}

	return &List{
		once:               lifecycle.NewOnce(),
		name:               name,
		uninitializedPeers: make(map[string]peer.Identifier, options.capacity),
		unavailablePeers:   make(map[string]*peerThunk, options.capacity),
		availablePeers:     make(map[string]*peerThunk, options.capacity),
		availableChooser:   availableChooser,
		transport:          transport,
		peerAvailableEvent: make(chan struct{}, 1),
	}
}

// List is an abstract peer list, backed by a peer.ListImplementation to
// determine which peer to choose among available peers.
// The abstract list manages available versus unavailable peers, intercepting
// these notifications from the transport's concrete implementation of
// peer.Peer with the peer.Subscriber API.
// The peer list will not choose an unavailable peer, prefering to block until
// one becomes available.
//
// The list is a suitable basis for concrete implementations like round-robin.
type List struct {
	lock sync.RWMutex

	name string

	shouldRetainPeers  atomic.Bool
	uninitializedPeers map[string]peer.Identifier

	unavailablePeers   map[string]*peerThunk
	availablePeers     map[string]*peerThunk
	availableChooser   peer.ListImplementation
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
	if t := pl.getThunk(pid); t != nil {
		return peer.ErrPeerAddAlreadyInList(pid.Identifier())
	}

	t := &peerThunk{list: pl, id: pid}
	t.boundOnFinish = t.onFinish
	p, err := pl.transport.RetainPeer(pid, t)
	if err != nil {
		return err
	}
	t.peer = p
	return pl.addPeer(t)
}

// Must be run in a mutex.Lock()
func (pl *List) addPeer(t *peerThunk) error {
	if t.peer.Status().ConnectionStatus != peer.Available {
		return pl.addToUnavailablePeers(t)
	}

	return pl.addToAvailablePeers(t)
}

// Must be run in a mutex.Lock()
func (pl *List) addToUnavailablePeers(t *peerThunk) error {
	pl.unavailablePeers[t.peer.Identifier()] = t
	return nil
}

// Must be run in a mutex.Lock()
func (pl *List) addToAvailablePeers(t *peerThunk) error {
	if pl.availablePeers[t.peer.Identifier()] != nil {
		return peer.ErrPeerAddAlreadyInList(t.peer.Identifier())
	}
	sub := pl.availableChooser.Add(t)
	t.SetSubscriber(sub)
	pl.availablePeers[t.Identifier()] = t
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

	if err := pl.availableChooser.Start(); err != nil {
		return err
	}

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
	return pl.once.Stop(pl.stop)
}

// stop will release all the peers from the list
func (pl *List) stop() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	var errs error

	if err := pl.availableChooser.Stop(); err != nil {
		errs = multierr.Append(errs, err)
	}

	availablePeers := pl.removeAllAvailablePeers(pl.availablePeers)
	errs = pl.releaseAll(errs, availablePeers)
	pl.addToUninitialized(availablePeers)

	unavailablePeers := pl.removeAllUnavailablePeers(pl.unavailablePeers)
	errs = pl.releaseAll(errs, unavailablePeers)
	pl.addToUninitialized(unavailablePeers)

	pl.shouldRetainPeers.Store(false)

	return errs
}

func (pl *List) addToUninitialized(thunks []*peerThunk) {
	for _, t := range thunks {
		pl.uninitializedPeers[t.id.Identifier()] = t.id
	}
}

// removeAllAvailablePeers will clear the availablePeers list and return all
// the Peers in the list in a slice
// Must be run in a mutex.Lock()
func (pl *List) removeAllAvailablePeers(toRemove map[string]*peerThunk) []*peerThunk {
	thunks := make([]*peerThunk, 0, len(toRemove))
	for id, t := range toRemove {
		thunks = append(thunks, t)
		delete(pl.availablePeers, id)
		pl.availableChooser.Remove(t, t.Subscriber())
	}
	return thunks
}

// removeAllUnavailablePeers will clear the unavailablePeers list and
// return all the Peers in the list in a slice
// Must be run in a mutex.Lock()
func (pl *List) removeAllUnavailablePeers(toRemove map[string]*peerThunk) []*peerThunk {
	thunks := make([]*peerThunk, 0, len(toRemove))
	for id, t := range toRemove {
		thunks = append(thunks, t)
		delete(toRemove, id)
	}
	return thunks
}

// releaseAll will iterate through a list of peers and call release
// on the transport
func (pl *List) releaseAll(errs error, peers []*peerThunk) error {
	for _, t := range peers {
		if err := pl.transport.ReleasePeer(t.peer, t); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}

// removePeerIdentifier will go remove references to the peer identifier and release
// it from the transport
// Must be run in a mutex.Lock()
func (pl *List) removePeerIdentifier(pid peer.Identifier) error {
	t, err := pl.removePeerIdentifierReferences(pid)
	if err != nil {
		// The peer has already been removed
		return err
	}

	return pl.transport.ReleasePeer(pid, t)
}

// removePeerIdentifierReferences will search through the Available and Unavailable Peers
// for the PeerID and remove it
// Must be run in a mutex.Lock()
func (pl *List) removePeerIdentifierReferences(pid peer.Identifier) (*peerThunk, error) {
	if t := pl.availablePeers[pid.Identifier()]; t != nil {
		return t, pl.removeFromAvailablePeers(t)
	}

	if t, ok := pl.unavailablePeers[pid.Identifier()]; ok && t != nil {
		pl.removeFromUnavailablePeers(t)
		return t, nil
	}

	return nil, peer.ErrPeerRemoveNotInList(pid.Identifier())
}

// removeFromAvailablePeers remove a peer from the Available Peers list the
// Peer should already be validated as non-nil and in the Available list.
// Must be run in a mutex.Lock()
func (pl *List) removeFromAvailablePeers(t *peerThunk) error {
	delete(pl.availablePeers, t.peer.Identifier())
	pl.availableChooser.Remove(t, t.Subscriber())
	t.SetSubscriber(nil)
	return nil
}

// removeFromUnavailablePeers remove a peer from the Unavailable Peers list the
// Peer should already be validated as non-nil and in the Unavailable list.
// Must be run in a mutex.Lock()
func (pl *List) removeFromUnavailablePeers(t *peerThunk) {
	delete(pl.unavailablePeers, t.peer.Identifier())
}

// Choose selects the next available peer in the peer list
func (pl *List) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	if err := pl.once.WaitUntilRunning(ctx); err != nil {
		return nil, nil, intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "%s peer list is not running", pl.name)
	}

	for {
		pl.lock.RLock()
		p := pl.availableChooser.Choose(ctx, req)
		pl.lock.RUnlock()

		if p != nil {
			t := p.(*peerThunk)
			pl.notifyPeerAvailable()
			t.onStart()
			return t.peer, t.boundOnFinish, nil
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

// notifyPeerAvailable writes to a channel indicating that a Peer is currently
// available for requests
func (pl *List) notifyPeerAvailable() {
	select {
	case pl.peerAvailableEvent <- struct{}{}:
	default:
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
	return yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "%s peer list timed out waiting for peer: %s", pl.name, err.Error())
}

// NotifyStatusChanged receives status change notifications for peers in the
// list.
func (pl *List) NotifyStatusChanged(pid peer.Identifier) {
	pl.lock.RLock()
	t := pl.getThunk(pid)
	pl.lock.RUnlock()

	if t != nil {
		t.NotifyStatusChanged(pid)
	}
}

// getThunk returns either the available or unavailable peer thunk.
// Must be called under a lock.
func (pl *List) getThunk(pid peer.Identifier) *peerThunk {
	if t := pl.availablePeers[pid.Identifier()]; t != nil {
		return t
	}
	return pl.unavailablePeers[pid.Identifier()]
}

// notifyStatusChanged gets called by peer thunks
func (pl *List) notifyStatusChanged(pid peer.Identifier) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	if t := pl.availablePeers[pid.Identifier()]; t != nil {
		// TODO: log error
		_ = pl.handleAvailablePeerStatusChange(t)
		return
	}

	if t := pl.unavailablePeers[pid.Identifier()]; t != nil {
		// TODO: log error
		_ = pl.handleUnavailablePeerStatusChange(t)
	}
	// No action required
}

// handleAvailablePeerStatusChange checks the connection status of a connected peer to potentially
// move that Peer from the PeerRing to the unavailable peer map
// Must be run in a mutex.Lock()
func (pl *List) handleAvailablePeerStatusChange(t *peerThunk) error {
	if t.peer.Status().ConnectionStatus == peer.Available {
		// Peer is in the proper pool, ignore
		return nil
	}

	pl.availableChooser.Remove(t, t.Subscriber())
	t.SetSubscriber(nil)
	delete(pl.availablePeers, t.peer.Identifier())

	return pl.addToUnavailablePeers(t)

}

// handleUnavailablePeerStatusChange checks the connection status of an unavailable peer to potentially
// move that Peer from the unavailablePeerMap into the available Peer Ring
// Must be run in a mutex.Lock()
func (pl *List) handleUnavailablePeerStatusChange(t *peerThunk) error {
	if t.peer.Status().ConnectionStatus != peer.Available {
		// Peer is in the proper pool, ignore
		return nil
	}

	pl.removeFromUnavailablePeers(t)
	return pl.addToAvailablePeers(t)
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

// Peers returns a snapshot of all retained (available and
// unavailable) peers.
func (pl *List) Peers() []peer.Peer {
	pl.lock.RLock()
	defer pl.lock.RUnlock()
	peers := make([]peer.Peer, 0)
	for _, t := range pl.availablePeers {
		peers = append(peers, t.peer)
	}
	for _, t := range pl.unavailablePeers {
		peers = append(peers, t.peer)
	}
	return peers
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
	for _, t := range pl.availablePeers {
		availables = append(availables, t.peer)
	}
	unavailables := make([]peer.Peer, 0, len(pl.unavailablePeers))
	for _, t := range pl.unavailablePeers {
		unavailables = append(unavailables, t.peer)
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
