// Copyright (c) 2019 Uber Technologies, Inc.
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

package abstractlist

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
)

var (
	_noContextDeadlineError = "%q peer list can't wait for peer without a context deadline"
	_unavailableError       = "%q peer list timed out waiting for peer: %s"
)

// Implementation is a collection of available peers, with its own
// subscribers for peer status change notifications.
// The available peer list encapsulates the logic for selecting from among
// available peers, whereas a ChooserList is responsible for retaining,
// releasing, and monitoring peer availability.
// Use "go.uber.org/yarpc/peer/abstractlist".List in conjunction with an
// Implementation to produce a "go.uber.org/yarpc/api/peer".List.
//
// abstractlist.List and abstractlist.Implementation compose well with sharding
// schemes the degenerate to returning the only available peer.
//
// The abstractlist.List calls Add, Remove, and Choose under a write lock so
// the implementation is free to perform mutations on its own data without
// locks.
type Implementation interface {
	Add(peer.StatusPeer, peer.Identifier) Subscriber
	Remove(peer.StatusPeer, peer.Identifier, Subscriber)
	// Choose must return an available peer under a list read lock, so must
	// not block.
	Choose(context.Context, *transport.Request) peer.StatusPeer
}

// Subscriber is a callback that implementations of peer list data structures
// must provide.
//
// The peer list uses the Subscriber to send notifications when a peer’s
// pending request count changes.
// A peer list implementation may have a single subscriber or a subscriber for
// each peer.
//
// Peer list data structure implementations
type Subscriber interface {
	UpdatePendingRequestCount(peer.Identifier, int)
}

type options struct {
	capacity  int
	noShuffle bool
	failFast  bool
	seed      int64
}

var defaultOptions = options{
	capacity: 10,
	seed:     time.Now().UnixNano(),
}

// Option customizes the behavior of a list.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(options *options) { f(options) }

// Capacity specifies the default capacity of the underlying
// data structures for this list
//
// Defaults to 10.
func Capacity(capacity int) Option {
	return optionFunc(func(options *options) {
		options.capacity = capacity
	})
}

// NoShuffle disables the default behavior of shuffling peer list order.
func NoShuffle() Option {
	return optionFunc(func(options *options) {
		options.noShuffle = true
	})
}

// FailFast indicates that the peer list should not wait for peers to be added,
// when choosing a peer.
//
// This option is particularly useful for proxies.
func FailFast() Option {
	return optionFunc(func(options *options) {
		options.failFast = true
	})
}

// Seed specifies the random seed to use for shuffling peers
//
// Defaults to approximately the process start time in nanoseconds.
func Seed(seed int64) Option {
	return optionFunc(func(options *options) {
		options.seed = seed
	})
}

// New creates a new peer list with an identifier chooser for available peers.
func New(name string, transport peer.Transport, implementation Implementation, opts ...Option) *List {
	options := defaultOptions
	for _, o := range opts {
		o.apply(&options)
	}

	return &List{
		once:               lifecycle.NewOnce(),
		name:               name,
		peers:              make(map[string]*peerFacade, options.capacity),
		offlinePeers:       make(map[string]peer.Identifier, options.capacity),
		implementation:     implementation,
		transport:          transport,
		noShuffle:          options.noShuffle,
		failFast:           options.failFast,
		randSrc:            rand.NewSource(options.seed),
		peerAvailableEvent: make(chan struct{}, 1),
	}
}

// List is an abstract peer list, backed by an Implementation to
// determine which peer to choose among available peers.
// The abstract list manages available versus unavailable peers, intercepting
// these notifications from the transport's concrete implementation of
// peer.StatusPeer with the peer.Subscriber API.
// The peer list will not choose an unavailable peer, prefering to block until
// one becomes available.
//
// The list is a suitable basis for concrete implementations like round-robin.
type List struct {
	lock sync.RWMutex
	once *lifecycle.Once

	name string

	peers              map[string]*peerFacade
	offlinePeers       map[string]peer.Identifier
	implementation     Implementation
	peerAvailableEvent chan struct{}
	transport          peer.Transport

	noShuffle bool
	failFast  bool
	randSrc   rand.Source
}

// Name returns the name of the list.
func (pl *List) Name() string { return pl.name }

// Transport returns the underlying transport for retaining and releasing peers.
func (pl *List) Transport() peer.Transport { return pl.transport }

// Update applies the additions and removals of peer Identifiers to the list
// it returns a multi-error result of every failure that happened without
// circuit breaking due to failures.
func (pl *List) Update(updates peer.ListUpdates) error {
	if len(updates.Additions) == 0 && len(updates.Removals) == 0 {
		return nil
	}

	pl.lock.Lock()
	defer pl.lock.Unlock()

	if !pl.once.IsRunning() {
		return pl.updateOffline(updates)
	}
	return pl.updateOnline(updates)
}

func (pl *List) updateOnline(updates peer.ListUpdates) error {
	var errs error
	for _, id := range updates.Removals {
		errs = multierr.Append(errs, pl.remove(id))
	}

	add := updates.Additions
	if !pl.noShuffle {
		add = shuffle(pl.randSrc, add)
	}

	for _, id := range add {
		errs = multierr.Append(errs, pl.add(id))
	}
	return errs
}

func (pl *List) updateOffline(updates peer.ListUpdates) error {
	var errs error
	for _, id := range updates.Removals {
		errs = multierr.Append(errs, pl.removeOffline(id))
	}
	for _, id := range updates.Additions {
		errs = multierr.Append(errs, pl.addOffline(id))
	}
	return errs
}

// Start notifies the List that requests will start coming
func (pl *List) Start() error {
	return pl.once.Start(pl.start)
}

func (pl *List) start() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	all := pl.offlinePeerIdentifiers()

	var err error
	err = multierr.Append(err, pl.updateOffline(peer.ListUpdates{
		Removals: all,
	}))
	err = multierr.Append(err, pl.updateOnline(peer.ListUpdates{
		Additions: all,
	}))
	return err
}

// Stop notifies the List that requests will stop coming
func (pl *List) Stop() error {
	return pl.once.Stop(pl.stop)
}

func (pl *List) stop() error {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	all := pl.onlinePeerIdentifiers()

	var err error
	err = multierr.Append(err, pl.updateOnline(peer.ListUpdates{
		Removals: all,
	}))
	err = multierr.Append(err, pl.updateOffline(peer.ListUpdates{
		Additions: all,
	}))
	return err
}

// IsRunning returns whether the peer list is running.
func (pl *List) IsRunning() bool {
	return pl.once.IsRunning()
}

// add retains a peer and sets up a thunk (a thin proxy for a peer) to receive
// connection status notifications from the dialer and track pending request
// counts.
//
// add does not add the peer to the list of peers available for choosing (the
// Implementation).
// The thunk is responsible for adding and removing the peer from the
// collection of available peers based on connection status notifications.
// Must be run inside a mutex.Lock()
func (pl *List) add(id peer.Identifier) error {
	addr := id.Identifier()

	if _, ok := pl.peers[addr]; ok {
		return peer.ErrPeerAddAlreadyInList(addr)
	}

	t := newPeerFacade(pl, id)
	t.boundOnFinish = t.onFinish

	p, err := pl.transport.RetainPeer(id, t)
	if err != nil {
		return err
	}

	t.peer = p
	pl.peers[addr] = t
	t.notifyStatusChanged(p)

	return nil
}

func (pl *List) addOffline(id peer.Identifier) error {
	addr := id.Identifier()

	if _, ok := pl.offlinePeers[addr]; ok {
		return peer.ErrPeerAddAlreadyInList(addr)
	}

	pl.offlinePeers[addr] = id
	return nil
}

// remove releases and forgets a peer.
// Must be run in a mutex.Lock()
func (pl *List) remove(id peer.Identifier) error {
	addr := id.Identifier()

	t, ok := pl.peers[addr]
	if !ok {
		return peer.ErrPeerRemoveNotInList(addr)
	}

	t.remove()
	delete(pl.peers, addr)

	return pl.transport.ReleasePeer(id, t)
}

func (pl *List) removeOffline(id peer.Identifier) error {
	addr := id.Identifier()

	_, ok := pl.offlinePeers[addr]
	if !ok {
		return peer.ErrPeerRemoveNotInList(addr)
	}

	delete(pl.offlinePeers, addr)

	return nil
}

// Choose selects the next available peer in the peer list
func (pl *List) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	if err := pl.once.WaitUntilRunning(ctx); err != nil {
		return nil, nil, intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "%q peer list is not running", pl.name)
	}
	for {
		p := pl.choose(ctx, req)
		// choose signals that there are no available peers by returning nil.
		// Thereafter, every Choose call will wait for a peer or peers to
		// become available again.
		if p != nil {
			// We call notifyPeerAvailable because there is a chance that more
			// than one chooser is blocked in waitForPeerAddedEvent.
			// Once a peer becomes available, all of these goroutines should
			// resume, not just one, until no peers are available again.
			// The underlying channel has a limited capacity, so every success
			// must trigger the rest to resume.
			pl.notifyPeerAvailable()
			t := p.(*peerFacade)
			t.onStart()
			return t.peer, t.boundOnFinish, nil
		} else if pl.failFast {
			return nil, nil, yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "%q peer list has no peer available", pl.name)
		}
		if err := pl.waitForPeerAddedEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
}

// choose in a function so panics in the implementation still unlock.
func (pl *List) choose(ctx context.Context, req *transport.Request) peer.StatusPeer {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	return pl.implementation.Choose(ctx, req)
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
	return yarpcerrors.Newf(yarpcerrors.CodeUnavailable, _unavailableError, pl.name, err.Error())
}

// NotifyStatusChanged receives status change notifications for peers in the
// list.
func (pl *List) NotifyStatusChanged(pid peer.Identifier) {
	pl.lock.RLock()
	p := pl.peers[pid.Identifier()]
	pl.lock.RUnlock()

	if p != nil {
		p.NotifyStatusChanged(p.id)
	}
}

// NumAvailable returns how many peers are available.
func (pl *List) NumAvailable() int {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	num := 0
	for _, p := range pl.peers {
		if p.status.ConnectionStatus == peer.Available {
			num++
		}
	}
	return num
}

// NumUnavailable returns how many peers are unavailable while the list is
// running.
func (pl *List) NumUnavailable() int {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	num := 0
	for _, p := range pl.peers {
		if p.status.ConnectionStatus == peer.Unavailable {
			num++
		}
	}
	return num
}

// NumUninitialized returns how many peers are unavailable because the peer
// list was stopped or has not yet started.
func (pl *List) NumUninitialized() int {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	return len(pl.offlinePeers)
}

// String returns a representation of the peer list state for debugging.
func (pl *List) String() string {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	peers := ""
	for _, peer := range pl.peers {
		peers += " " + peer.String()
	}

	return fmt.Sprintf("[%s%s]", pl.name, peers)
}

// Available returns whether the identifier peer is available for traffic.
func (pl *List) Available(pid peer.Identifier) bool {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	if p, ok := pl.peers[pid.Identifier()]; ok {
		return p.Status().ConnectionStatus == peer.Available
	}
	return false
}

// Uninitialized returns whether the identifier peer is present but uninitialized.
func (pl *List) Uninitialized(pid peer.Identifier) bool {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	_, exists := pl.offlinePeers[pid.Identifier()]
	return exists
}

// Peers returns a snapshot of all retained (available and
// unavailable) peers.
func (pl *List) Peers() []peer.StatusPeer {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	peers := make([]peer.StatusPeer, 0)
	for _, p := range pl.peers {
		peers = append(peers, p.peer)
	}
	return peers
}

func (pl *List) onlinePeerIdentifiers() []peer.Identifier {
	addrs := make([]string, 0, len(pl.peers))
	for addr := range pl.peers {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)

	ids := make([]peer.Identifier, len(addrs))
	for i, addr := range addrs {
		ids[i] = pl.peers[addr].peer
	}
	return ids
}

func (pl *List) offlinePeerIdentifiers() []peer.Identifier {
	addrs := make([]string, 0, len(pl.offlinePeers))
	for addr := range pl.offlinePeers {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)

	ids := make([]peer.Identifier, len(addrs))
	for i, addr := range addrs {
		id := pl.offlinePeers[addr]
		ids[i] = id
	}
	return ids
}

// Introspect returns a ChooserStatus with a summary of the Peers.
func (pl *List) Introspect() introspection.ChooserStatus {
	peers := pl.Peers()

	available := 0
	unavailable := 0
	for _, p := range peers {
		if p.Status().ConnectionStatus == peer.Available {
			available++
		} else {
			unavailable++
		}
	}

	peerStatuses := make([]introspection.PeerStatus, 0,
		len(pl.peers))

	buildPeerStatus := func(peer peer.StatusPeer) introspection.PeerStatus {
		ps := peer.Status()
		return introspection.PeerStatus{
			Identifier: peer.Identifier(),
			State: fmt.Sprintf("%s, %d pending request(s)",
				ps.ConnectionStatus.String(),
				ps.PendingRequestCount),
		}
	}

	for _, peer := range peers {
		peerStatuses = append(peerStatuses, buildPeerStatus(peer))
	}

	return introspection.ChooserStatus{
		Name: pl.name,
		State: fmt.Sprintf("%s (%d/%d available)", pl.once.State(), available,
			available+unavailable),
		Peers: peerStatuses,
	}
}

// shuffle randomizes the order of a slice of peers.
// see: https://en.wikipedia.org/wiki/Fisher-Yates_shuffle
func shuffle(src rand.Source, in []peer.Identifier) []peer.Identifier {
	shuffled := make([]peer.Identifier, len(in))
	r := rand.New(src)
	copy(shuffled, in)
	for i := len(in) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}
