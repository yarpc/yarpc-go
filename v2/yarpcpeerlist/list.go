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

package yarpcpeerlist

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/multierr"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerrors"
	"go.uber.org/yarpc/v2/yarpcpeer"
)

var (
	_noContextDeadlineError = "can't wait for peer without a context deadline for a %s peer list"
)

// Implementation is a collection of available peers, with its own
// subscribers for peer status change notifications.
// The available peer list encapsulates the logic for selecting from among
// available peers, whereas a ChooserList is responsible for retaining,
// releasing, and monitoring peer availability.
// Use "go.uber.org/yarpc/v2/yarpcpeerlist".List in conjunction with an
// Implementation to produce a "go.uber.org/yarpc/v2/yarpcpeer".List.
//
// yarpcpeerlist.List and yarpcpeerlist.Implementation compose well with
// sharding schemes the degenerate to returning the only available peer.
//
// The yarpcpeerlist.List calls Add, Remove, and Choose under a write lock so
// the implementation is free to perform mutations on its own data without
// locks.
type Implementation interface {
	Add(yarpcpeer.StatusPeer, yarpcpeer.Identifier) yarpcpeer.Subscriber
	Remove(yarpcpeer.StatusPeer, yarpcpeer.Identifier, yarpcpeer.Subscriber)
	// Choose must return an available peer under a list read lock, so must
	// not block.
	Choose(context.Context, *yarpc.Request) yarpcpeer.StatusPeer
}

type listOptions struct {
	capacity  int
	noShuffle bool
	seed      int64
}

var defaultListOptions = listOptions{
	capacity: 10,
	seed:     time.Now().UnixNano(),
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

// NoShuffle disables the default behavior of shuffling peerlist order.
func NoShuffle() ListOption {
	return listOptionFunc(func(options *listOptions) {
		options.noShuffle = true
	})
}

// Seed specifies the random seed to use for shuffling peers
//
// Defaults to approximately the process start time in nanoseconds.
func Seed(seed int64) ListOption {
	return listOptionFunc(func(options *listOptions) {
		options.seed = seed
	})
}

// New creates a new peer list with an identifier chooser for available peers.
func New(name string, transport yarpcpeer.Transport, implementation Implementation, opts ...ListOption) *List {
	options := defaultListOptions
	for _, o := range opts {
		o.apply(&options)
	}

	return &List{
		name:               name,
		unavailablePeers:   make(map[string]*peerThunk, options.capacity),
		availablePeers:     make(map[string]*peerThunk, options.capacity),
		implementation:     implementation,
		transport:          transport,
		noShuffle:          options.noShuffle,
		randSrc:            rand.NewSource(options.seed),
		peerAvailableEvent: make(chan struct{}, 1),
	}
}

// List is an abstract peer list, backed by an Implementation to
// determine which peer to choose among available peers.
// The abstract list manages available versus unavailable peers, intercepting
// these notifications from the transport's concrete implementation of
// yarpcpeer.Peer with the yarpcpeer.Subscriber API.
// The peer list will not choose an unavailable peer, prefering to block until
// one becomes available.
//
// The list is a suitable basis for concrete implementations like round-robin.
type List struct {
	lock sync.RWMutex

	name string

	unavailablePeers   map[string]*peerThunk
	availablePeers     map[string]*peerThunk
	implementation     Implementation
	peerAvailableEvent chan struct{}
	transport          yarpcpeer.Transport

	noShuffle bool
	randSrc   rand.Source
}

// Update applies the additions and removals of peer Identifiers to the list
// it returns a multi-error result of every failure that happened without
// circuit breaking due to failures.
func (pl *List) Update(updates yarpcpeer.ListUpdates) error {
	if len(updates.Additions) == 0 && len(updates.Removals) == 0 {
		return nil
	}

	pl.lock.Lock()
	defer pl.lock.Unlock()

	var errs error
	for _, pid := range updates.Removals {
		errs = multierr.Append(errs, pl.removePeerIdentifier(pid))
	}

	add := updates.Additions
	if !pl.noShuffle {
		add = shuffle(pl.randSrc, add)
	}

	for _, pid := range add {
		errs = multierr.Append(errs, pl.addPeerIdentifier(pid))
	}
	return errs
}

// Must be run inside a mutex.Lock()
func (pl *List) addPeerIdentifier(pid yarpcpeer.Identifier) error {
	if t := pl.getThunk(pid); t != nil {
		return yarpcpeer.ErrPeerAddAlreadyInList(pid.Identifier())
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
	if t.peer.Status().ConnectionStatus != yarpcpeer.Available {
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
		return yarpcpeer.ErrPeerAddAlreadyInList(t.peer.Identifier())
	}
	sub := pl.implementation.Add(t, t.id)
	t.SetSubscriber(sub)
	pl.availablePeers[t.Identifier()] = t
	pl.notifyPeerAvailable()
	return nil
}

// removeAllAvailablePeers will clear the availablePeers list and return all
// the Peers in the list in a slice
// Must be run in a mutex.Lock()
func (pl *List) removeAllAvailablePeers(toRemove map[string]*peerThunk) []*peerThunk {
	thunks := make([]*peerThunk, 0, len(toRemove))
	for id, t := range toRemove {
		thunks = append(thunks, t)
		delete(pl.availablePeers, id)
		pl.implementation.Remove(t, t.id, t.Subscriber())
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
func (pl *List) removePeerIdentifier(pid yarpcpeer.Identifier) error {
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
func (pl *List) removePeerIdentifierReferences(pid yarpcpeer.Identifier) (*peerThunk, error) {
	if t := pl.availablePeers[pid.Identifier()]; t != nil {
		return t, pl.removeFromAvailablePeers(t)
	}

	if t, ok := pl.unavailablePeers[pid.Identifier()]; ok && t != nil {
		pl.removeFromUnavailablePeers(t)
		return t, nil
	}

	return nil, yarpcpeer.ErrPeerRemoveNotInList(pid.Identifier())
}

// removeFromAvailablePeers remove a peer from the Available Peers list the
// Peer should already be validated as non-nil and in the Available list.
// Must be run in a mutex.Lock()
func (pl *List) removeFromAvailablePeers(t *peerThunk) error {
	delete(pl.availablePeers, t.peer.Identifier())
	pl.implementation.Remove(t, t.id, t.Subscriber())
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
func (pl *List) Choose(ctx context.Context, req *yarpc.Request) (yarpcpeer.Peer, func(error), error) {
	for {
		pl.lock.RLock()
		p := pl.implementation.Choose(ctx, req)
		pl.lock.RUnlock()

		if p != nil {
			t := p.(*peerThunk)
			pl.notifyPeerAvailable()
			t.StartRequest()
			return t.peer, t.boundOnFinish, nil
		}
		if err := pl.waitForPeerAddedEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
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
func (pl *List) NotifyStatusChanged(pid yarpcpeer.Identifier) {
	pl.lock.RLock()
	t := pl.getThunk(pid)
	pl.lock.RUnlock()

	if t != nil {
		t.NotifyStatusChanged(t.id)
	}
}

// getThunk returns either the available or unavailable peer thunk.
// Must be called under a lock.
func (pl *List) getThunk(pid yarpcpeer.Identifier) *peerThunk {
	if t := pl.availablePeers[pid.Identifier()]; t != nil {
		return t
	}
	return pl.unavailablePeers[pid.Identifier()]
}

// notifyStatusChanged gets called by peer thunks
func (pl *List) notifyStatusChanged(pid yarpcpeer.Identifier) {
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

// handleAvailablePeerStatusChange checks the connection status of a connected
// peer to potentially move that Peer from the implementation data structure to
// the unavailable peer map
// Must be run in a mutex.Lock()
func (pl *List) handleAvailablePeerStatusChange(t *peerThunk) error {
	if t.peer.Status().ConnectionStatus == yarpcpeer.Available {
		// Peer is in the proper pool, ignore
		return nil
	}

	pl.implementation.Remove(t, t.id, t.Subscriber())
	t.SetSubscriber(nil)
	delete(pl.availablePeers, t.peer.Identifier())

	return pl.addToUnavailablePeers(t)

}

// handleUnavailablePeerStatusChange checks the connection status of an unavailable peer to potentially
// move that Peer from the unavailablePeerMap into the available Peer Ring
// Must be run in a mutex.Lock()
func (pl *List) handleUnavailablePeerStatusChange(t *peerThunk) error {
	if t.peer.Status().ConnectionStatus != yarpcpeer.Available {
		// Peer is in the proper pool, ignore
		return nil
	}

	pl.removeFromUnavailablePeers(t)
	return pl.addToAvailablePeers(t)
}

// Available returns whether the identifier peer is available for traffic.
func (pl *List) Available(p yarpcpeer.Identifier) bool {
	_, ok := pl.availablePeers[p.Identifier()]
	return ok
}

// Peers returns a snapshot of all retained (available and
// unavailable) peers.
func (pl *List) Peers() []yarpcpeer.Peer {
	pl.lock.RLock()
	defer pl.lock.RUnlock()
	peers := make([]yarpcpeer.Peer, 0)
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

// shuffle randomizes the order of a slice of peers.
// see: https://en.wikipedia.org/wiki/Fisher-Yates_shuffle
func shuffle(src rand.Source, in []yarpcpeer.Identifier) []yarpcpeer.Identifier {
	shuffled := make([]yarpcpeer.Identifier, len(in))
	r := rand.New(src)
	copy(shuffled, in)
	for i := len(in) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

// values returns a slice of the values contained in a map of peers.
func values(m map[string]yarpcpeer.Identifier) []yarpcpeer.Identifier {
	vs := make([]yarpcpeer.Identifier, 0, len(m))
	for _, v := range m {
		vs = append(vs, v)
	}
	return vs
}
