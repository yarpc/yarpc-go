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
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

// Implementation is a collection of available peers, with its own
// subscribers for pending request count change notifications.
// The abstract list uses the implementation to track peers that can be
// returned by Choose, as opposed to those that were added but reported
// unavailable by the underlying transport.
// Use "go.uber.org/yarpc/peer/abstractlist".List in conjunction with an
// Implementation to produce a "go.uber.org/yarpc/api/peer".ChooserList.
//
// The abstractlist.List calls Add, Remove, and Choose under a write lock so
// the implementation is free to perform mutations on its own data without
// locks.
//
// Choose must return nil immediately if the collection is empty.
// The abstractlist.List guarantees that peers will only be added if they're
// absent, and only removed they are present.
// Choose should not block.
type Implementation interface {
	Add(peer.StatusPeer, peer.Identifier) Subscriber
	Remove(peer.StatusPeer, peer.Identifier, Subscriber)
	Choose(*transport.Request) peer.StatusPeer
}

// Subscriber is a callback that implementations of peer list data structures
// must provide.
//
// The peer list uses the Subscriber to send notifications when a peer’s
// pending request count changes.
// A peer list implementation may have a single subscriber or a subscriber for
// each peer.
type Subscriber interface {
	UpdatePendingRequestCount(int)
}

type options struct {
	capacity  int
	noShuffle bool
	failFast  bool
	seed      int64
	logger    *zap.Logger
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

// Logger specifies a logger.
func Logger(logger *zap.Logger) Option {
	return optionFunc(func(options *options) {
		options.logger = logger
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

	logger := options.logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &List{
		once:               lifecycle.NewOnce(),
		name:               name,
		logger:             logger,
		peers:              make(map[string]*peerFacade, options.capacity),
		plans:              make(map[string]peer.Identifier, options.capacity),
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
//
// This abstract list does not participate in the transport’s pending request
// count tracking.
// The list tracks pending request counts for the peers that it chooses, does
// not inform the transport of these choices, and ignores notifications from
// the transport about choices other peer lists that share the same peers have
// made.
type List struct {
	name   string
	logger *zap.Logger

	once               *lifecycle.Once
	lock               sync.RWMutex
	peers              map[string]*peerFacade
	implementation     Implementation
	peerAvailableEvent chan struct{}
	transport          peer.Transport

	updatesLock sync.RWMutex
	plans       map[string]peer.Identifier
	updates     []operation
	recycle     []operation
	started     bool
	stopped     bool

	noShuffle bool
	failFast  bool
	randSrc   rand.Source
}

type opCode byte

const (
	addCode opCode = iota + 1
	removeCode
	stopCode
)

type operation struct {
	Code opCode
	ID   peer.Identifier
}

// Name returns the name of the list.
func (pl *List) Name() string { return pl.name }

// Transport returns the underlying transport for retaining and releasing peers.
func (pl *List) Transport() peer.Transport { return pl.transport }

// Update applies the additions and removals of peer Identifiers to the list
// it returns a multi-error result of every failure that happened without
// circuit breaking due to failures.
//
// Updates must be serialized so no peer is removed if it is absent and no peer
// is added if it is present.
// Updates should not have overlapping additions and removals, but the list
// will tollerate this case, but may cause existing connections to close and be
// replaced.
//
// Update will return errors if its invariants are violated, regardless of
// whether updates are sent while the list is running.
// Updates may be interleaved with Start and Stop in any order any number of
// times.
func (pl *List) Update(updates peer.ListUpdates) error {
	pl.logger.Debug("peer list update",
		zap.Int("additions", len(updates.Additions)),
		zap.Int("removals", len(updates.Removals)))

	if len(updates.Additions) == 0 && len(updates.Removals) == 0 {
		return nil
	}

	if err := pl.update(updates); err != nil {
		return err
	}

	pl.notifyFlushNeeded()

	return nil
}

func (pl *List) update(updates peer.ListUpdates) error {
	// We can shuffle the additions before taking on the lock, to minimize the
	// duration we hold the lock.
	additions := updates.Additions
	if !pl.noShuffle {
		additions = shuffle(pl.randSrc, additions)
	}

	pl.updatesLock.Lock()
	defer pl.updatesLock.Unlock()

	var errs error
	for _, id := range updates.Removals {
		errs = multierr.Append(errs, pl.remove(id))
	}
	for _, id := range additions {
		errs = multierr.Append(errs, pl.add(id))
	}

	return errs
}

func (pl *List) remove(id peer.Identifier) error {
	addr := id.Identifier()

	_, ok := pl.plans[addr]
	if !ok {
		return peer.ErrPeerRemoveNotInList(addr)
	}

	delete(pl.plans, addr)

	pl.updates = append(pl.updates, operation{
		Code: removeCode,
		ID:   id,
	})

	pl.logger.Debug("enqueue peer to remove", zap.String("address", addr))

	return nil
}

// addOffline must be run under a list lock.
func (pl *List) add(id peer.Identifier) error {
	addr := id.Identifier()

	if _, ok := pl.plans[addr]; ok {
		return peer.ErrPeerAddAlreadyInList(addr)
	}

	pl.plans[addr] = id

	pl.updates = append(pl.updates, operation{
		Code: addCode,
		ID:   id,
	})

	pl.logger.Debug("enqueue peer to add", zap.String("address", addr))

	return nil
}

// notifyFlushNeeded must not be called while holding the update queue lock nor the
// list lock.
func (pl *List) notifyFlushNeeded() {
	// TODO or when AutoFlush is enabled for tests.
	if pl.once.IsRunning() {
		pl.Flush()
	}
}

// Flush effects all enqueued updates synchronously and should only be used to
// synchronize tests.
func (pl *List) Flush() {
	pl.updatesLock.Lock()
	defer pl.updatesLock.Unlock()

	pl.lock.Lock()
	defer pl.lock.Unlock()

	if pl.stopped {
		return
	}

	for _, op := range pl.updates {
		switch op.Code {
		case addCode:
			pl.doAdd(op.ID)
		case removeCode:
			pl.doRemove(op.ID)
		case stopCode:
			pl.doStop()
			pl.stopped = true
			return
		}
	}

	// Clear updates, retain all prior allocated capacity.
	pl.updates = pl.updates[0:0]
}

// doAdd retains a peer and sets up a facade (a thin proxy for a peer)
// to receive connection status notifications from the dialer and track pending
// request counts.
//
// doAdd does not add the peer to the list of peers available for
// choosing (the Implementation).
// The facade is responsible for adding and removing the peer from the
// collection of available peers based on connection status notifications.
//
// doAdd must be run inside a list lock.
func (pl *List) doAdd(id peer.Identifier) {
	addr := id.Identifier()

	if _, ok := pl.peers[addr]; ok {
		pl.logger.DPanic("assertion error: adding peer that is already present", zap.String("address", addr))
		return
	}

	pf := &peerFacade{list: pl, id: id}
	pf.onFinish = pl.onFinishFunc(pf)

	// The transport must not call back before returning.
	p, err := pl.transport.RetainPeer(id, pf)
	if err != nil {
		pl.logger.DPanic("error retaining peer for peer list", zap.Error(err))
		return
	}

	pf.peer = p
	pl.peers[addr] = pf
	pl.notifyStatusChanged(pf)

	pl.logger.Debug("added peer", zap.String("address", addr))
}

// doRemove releases and forgets a peer.
//
// doRemove must be run under a list lock.
func (pl *List) doRemove(id peer.Identifier) {
	addr := id.Identifier()

	pf, ok := pl.peers[addr]
	if !ok {
		pl.logger.DPanic("assertion error: removing peer that is currently absent", zap.String("address", addr))
		return
	}

	if pf.status.ConnectionStatus == peer.Available {
		pl.implementation.Remove(pf, pf.id, pf.subscriber)
		pf.subscriber = nil
	}
	pf.status.ConnectionStatus = peer.Unavailable

	delete(pl.peers, addr)

	pl.logger.Debug("removed peer", zap.String("address", addr))

	// The transport must not call back before returning.
	if err := pl.transport.ReleasePeer(id, pf); err != nil {
		pl.logger.DPanic("error releasing peer for peer list", zap.Error(err))
	}
}

func (pl *List) doStop() {
	for _, peer := range pl.peers {
		pl.doRemove(peer)
	}
}

// Start notifies the List that requests will start coming
func (pl *List) Start() error {
	return pl.once.Start(pl.start)
}

func (pl *List) start() error {
	pl.logger.Debug("starting peer list")

	// TODO flush in goroutine
	pl.Flush()

	pl.logger.Debug("started peer list")
	return nil
}

// Stop notifies the List that requests will stop coming
func (pl *List) Stop() error {
	return pl.once.Stop(pl.stop)
}

func (pl *List) stop() error {
	pl.logger.Debug("stopping peer list")
	pl.stopUpdates()

	// TODO flush conditionally
	pl.Flush()

	pl.logger.Debug("stopped peer list")
	return nil
}

func (pl *List) stopUpdates() {
	pl.updatesLock.Lock()
	defer pl.updatesLock.Unlock()

	pl.updates = append(pl.updates, operation{
		Code: stopCode,
	})
}

// IsRunning returns whether the peer list is running.
func (pl *List) IsRunning() bool {
	return pl.once.IsRunning()
}

// Choose selects the next available peer in the peer list.
func (pl *List) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, nil, pl.newNoContextDeadlineError()
	}
	// We wait for the chooser to start and produce an error if the list does
	// not start before the context deadline times out.
	// This ensures that the developer sees a meaningful error if they forget
	// to run the lifecycle methods.
	if err := pl.once.WaitUntilRunning(ctx); err != nil {
		return nil, nil, intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "%q peer list is not running", pl.name)
	}

	// Choose runs without a lock because it spends the bulk of its time in a
	// wait loop.
	for {
		p := pl.choose(req)
		// choose signals that there are no available peers by returning nil.
		// Thereafter, every Choose call will wait for a peer or peers to
		// become available again.
		// We reach for an available peer optimistically, resorting to waiting
		// for a notification only if the underlying list is empty.
		if p != nil {
			// We call notifyPeerAvailable because there is a chance that more
			// than one chooser is blocked in waitForPeerAddedEvent.
			// Once a peer becomes available, all of these goroutines should
			// resume, not just one, until no peers are available again.
			// The underlying channel has a limited capacity, so every success
			// must trigger the rest to resume.
			pl.notifyPeerAvailable()
			pf := p.(*peerFacade)
			pl.onStart(pf)
			return pf.peer, pf.onFinish, nil
		}
		if pl.failFast {
			return nil, nil, yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "%q peer list has no peer available", pl.name)
		}
		if err := pl.waitForPeerAddedEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
}

// choose guards the underlying implementation's consistency around a lock, and
// recovers the lock if the underlying list panics.
func (pl *List) choose(req *transport.Request) peer.StatusPeer {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	return pl.implementation.Choose(req)
}

func (pl *List) onStart(pf *peerFacade) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	pf.status.PendingRequestCount++
	if pf.subscriber != nil {
		pf.subscriber.UpdatePendingRequestCount(pf.status.PendingRequestCount)
	}
}

func (pl *List) onFinish(pf *peerFacade, err error) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	pf.status.PendingRequestCount--
	if pf.subscriber != nil {
		pf.subscriber.UpdatePendingRequestCount(pf.status.PendingRequestCount)
	}
}

func (pl *List) onFinishFunc(pf *peerFacade) func(error) {
	return func(err error) {
		pl.onFinish(pf, err)
	}
}

// NotifyStatusChanged receives status change notifications for peers in the
// list.
//
// This function exists only as is necessary for dispatching connection status
// changes from tests.
func (pl *List) NotifyStatusChanged(pid peer.Identifier) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	pf := pl.peers[pid.Identifier()]
	pl.notifyStatusChanged(pf)
}

func (pl *List) lockAndNotifyStatusChanged(pf *peerFacade) {
	pl.lock.Lock()
	defer pl.lock.Unlock()

	pl.notifyStatusChanged(pf)
}

func (pl *List) status(pf *peerFacade) peer.Status {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	return pf.status
}

// notifyStatusChanged must be run under a list lock.
func (pl *List) notifyStatusChanged(pf *peerFacade) {
	if pf == nil {
		return
	}

	status := pf.peer.Status().ConnectionStatus
	if pf.status.ConnectionStatus != status {
		pf.status.ConnectionStatus = status
		switch status {
		case peer.Available:
			sub := pf.list.implementation.Add(pf, pf.id)
			pf.subscriber = sub
			pf.list.notifyPeerAvailable()
		default:
			pf.list.implementation.Remove(pf, pf.id, pf.subscriber)
			pf.subscriber = nil
		}
	}
}

// notifyPeerAvailable writes to a channel indicating that a Peer is currently
// available for requests.
//
// notifyPeerAvailable may be called without a list lock.
func (pl *List) notifyPeerAvailable() {
	select {
	case pl.peerAvailableEvent <- struct{}{}:
	default:
	}
}

// waitForPeerAddedEvent waits until a peer is added to the peer list or the
// given context finishes.
//
// waitForPeerAddedEvent must not be run under a lock.
func (pl *List) waitForPeerAddedEvent(ctx context.Context) error {
	select {
	case <-pl.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return pl.newUnavailableError(ctx.Err())
	}
}

func (pl *List) newNoContextDeadlineError() error {
	return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "%q peer list can't wait for peer without a context deadline", pl.name)
}

func (pl *List) newUnavailableError(err error) error {
	return yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "%q peer list timed out waiting for peer: %s", pl.name, err.Error())
}

func (pl *List) countPeersWithStatus(status peer.ConnectionStatus) int {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	num := 0
	for _, pf := range pl.peers {
		if pf.status.ConnectionStatus == status {
			num++
		}
	}
	return num
}

// NumAvailable returns how many peers are available.
func (pl *List) NumAvailable() int {
	return pl.countPeersWithStatus(peer.Available)
}

// NumUnavailable returns how many peers are unavailable while the list is
// running.
func (pl *List) NumUnavailable() int {
	return pl.countPeersWithStatus(peer.Unavailable)
}

// Available returns whether the identifier peer is available for traffic.
func (pl *List) Available(pid peer.Identifier) bool {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	if pf, ok := pl.peers[pid.Identifier()]; ok {
		return pf.status.ConnectionStatus == peer.Available
	}
	return false
}

// Peers returns a snapshot of all retained (available and unavailable) peers.
func (pl *List) Peers() []peer.StatusPeer {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	peers := make([]peer.StatusPeer, 0, len(pl.peers))
	for _, pf := range pl.peers {
		peers = append(peers, pf.peer)
	}
	return peers
}

// Introspect returns a ChooserStatus with a summary of the Peers.
func (pl *List) Introspect() introspection.ChooserStatus {
	pl.lock.RLock()
	defer pl.lock.RUnlock()

	available := 0
	unavailable := 0
	for _, pf := range pl.peers {
		if pf.status.ConnectionStatus == peer.Available {
			available++
		} else {
			unavailable++
		}
	}

	peerStatuses := make([]introspection.PeerStatus, 0,
		len(pl.peers))

	buildPeerStatus := func(pf *peerFacade) introspection.PeerStatus {
		ps := pf.status
		return introspection.PeerStatus{
			Identifier: pf.peer.Identifier(),
			State: fmt.Sprintf("%s, %d pending request(s)",
				ps.ConnectionStatus.String(),
				ps.PendingRequestCount),
		}
	}

	for _, pf := range pl.peers {
		peerStatuses = append(peerStatuses, buildPeerStatus(pf))
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
