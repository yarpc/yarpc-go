// Copyright (c) 2025 Uber Technologies, Inc.
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

package peerheap

import (
	"context"
	"math"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
)

var (
	_noContextDeadlineError = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "can't wait for peer without a context deadline for peerheap")
)

const unavailablePenalty = math.MaxInt32

type heapConfig struct {
	startupWait time.Duration
}

var defaultHeapConfig = heapConfig{
	startupWait: 5 * time.Second,
}

// HeapOption customizes the behavior of a peer heap.
type HeapOption func(*heapConfig)

// StartupWait specifies how long updates to the heap will block
// before the list heap been started
//
// Defaults to 5 seconds.
func StartupWait(t time.Duration) HeapOption {
	return func(c *heapConfig) {
		c.startupWait = t
	}
}

// List is a peer list and peer chooser that favors the peer with the least
// pending requests, and then favors the least recently used or most recently
// introduced peer.
type List struct {
	mu   sync.Mutex
	once *lifecycle.Once

	transport peer.Transport

	byScore      peerHeap
	byIdentifier map[string]*peerScore

	peerAvailableEvent chan struct{}

	startupWait time.Duration
}

// IsRunning returns whether the peer list is running.
func (pl *List) IsRunning() bool {
	return pl.once.IsRunning()
}

// Start starts the peer list.
func (pl *List) Start() error {
	return pl.once.Start(nil)
}

// Stop stops the peer list. This releases all retained peers.
func (pl *List) Stop() error {
	return pl.once.Stop(pl.clearPeers) // TODO clear peers
}

// New returns a new peer heap-chooser-list for the given transport.
func New(transport peer.Transport, opts ...HeapOption) *List {
	cfg := defaultHeapConfig
	for _, o := range opts {
		o(&cfg)
	}

	return &List{
		once:               lifecycle.NewOnce(),
		transport:          transport,
		byIdentifier:       make(map[string]*peerScore),
		peerAvailableEvent: make(chan struct{}, 1),
		startupWait:        cfg.startupWait,
	}
}

// Update satisfies the peer.List interface, so a peer list updater can manage
// the retained peers.
func (pl *List) Update(updates peer.ListUpdates) error {
	ctx, cancel := context.WithTimeout(context.Background(), pl.startupWait)
	defer cancel()
	if err := pl.once.WaitUntilRunning(ctx); err != nil {
		return intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "%s peer list is not running", "peer heap")
	}

	var errs error

	pl.mu.Lock()
	defer pl.mu.Unlock()

	for _, pid := range updates.Removals {
		errs = multierr.Append(errs, pl.releasePeer(pid))
	}

	for _, pid := range updates.Additions {
		errs = multierr.Append(errs, pl.retainPeer(pid))
	}

	return errs
}

// retainPeer must be called with the mutex locked.
func (pl *List) retainPeer(pid peer.Identifier) error {
	if _, ok := pl.byIdentifier[pid.Identifier()]; ok {
		return peer.ErrPeerAddAlreadyInList(pid.Identifier())
	}

	ps := &peerScore{id: pid, list: pl}
	p, err := pl.transport.RetainPeer(pid, ps)
	if err != nil {
		return err
	}

	ps.peer = p
	ps.score = scorePeer(p)
	ps.boundFinish = ps.finish
	pl.byIdentifier[pid.Identifier()] = ps
	pl.byScore.pushPeer(ps)
	pl.internalNotifyStatusChanged(ps)
	return nil
}

// releasePeer must be called with the mutex locked.
func (pl *List) releasePeer(pid peer.Identifier) error {
	ps, ok := pl.byIdentifier[pid.Identifier()]
	if !ok {
		return peer.ErrPeerRemoveNotInList(pid.Identifier())
	}

	if err := pl.byScore.validate(ps); err != nil {
		return err
	}

	err := pl.transport.ReleasePeer(pid, ps)
	delete(pl.byIdentifier, pid.Identifier())
	pl.byScore.delete(ps.idx)
	ps.list = nil
	return err
}

func (pl *List) clearPeers() error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	var errs error

	for {
		ps, ok := pl.byScore.peekPeer()
		if !ok {
			break
		}

		errs = multierr.Append(errs, pl.releasePeer(ps.id))
	}

	return errs
}

// Choose satisfies peer.Chooser, providing a single peer for a request, a
// callback for when the request is finished, or an error if it fails.
// The choose method takes a context that must have a deadline.
// Choose resepects this deadline, waiting for an available peer until the
// deadline.
// The peer heap does not use the given *transport.Request and can safely
// receive nil.
func (pl *List) Choose(ctx context.Context, _ *transport.Request) (peer.Peer, func(error), error) {
	if err := pl.once.WaitUntilRunning(ctx); err != nil {
		return nil, nil, intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "%s peer list is not running", "peer heap")
	}

	for {
		if ps, ok := pl.get(); ok {
			pl.notifyPeerAvailable()
			ps.peer.StartRequest()
			return ps.peer, ps.boundFinish, nil
		}

		if err := pl.waitForPeerAvailableEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
}

func (pl *List) get() (*peerScore, bool) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	ps, ok := pl.byScore.popPeer()
	if !ok {
		return nil, false
	}

	// Note: We push the peer back to reset the "next" counter.
	// This gives us round-robin behavior.
	pl.byScore.pushPeer(ps)

	return ps, ps.status.ConnectionStatus == peer.Available
}

// waitForPeerAvailableEvent waits until a peer is added to the peer list or the
// given context finishes.
// Must NOT be run in a mutex.Lock()
func (pl *List) waitForPeerAvailableEvent(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return _noContextDeadlineError
	}

	select {
	case <-pl.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return newUnavailableError(ctx.Err())
	}
}

func newUnavailableError(err error) error {
	return yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "peer heap timed out waiting for peer: %s", err.Error())
}

// notifyPeerAvailable writes to a channel indicating that a Peer is currently
// available for requests
func (pl *List) notifyPeerAvailable() {
	select {
	case pl.peerAvailableEvent <- struct{}{}:
	default:
	}
}

// NotifyStatusChanged receives notifications when a peer becomes available,
// connected, unavailable, or when its pending request count changes.
// This method satisfies peer.Subscriber and is only used for tests, since
// the peer heap has a subscriber for each invividual peer.
func (pl *List) NotifyStatusChanged(pid peer.Identifier) {
	pl.mu.Lock()
	ps := pl.byIdentifier[pid.Identifier()]
	pl.mu.Unlock()
	ps.NotifyStatusChanged(pid)
}

func (pl *List) peerScoreChanged(ps *peerScore) {
	pl.mu.Lock()
	pl.rescorePeer(ps)
	pl.mu.Unlock()

	if ps.peer.Status().ConnectionStatus == peer.Available {
		pl.notifyPeerAvailable()
	}
}

func (pl *List) internalNotifyStatusChanged(ps *peerScore) {
	pl.rescorePeer(ps)

	if ps.peer.Status().ConnectionStatus == peer.Available {
		pl.notifyPeerAvailable()
	}
}

func (pl *List) rescorePeer(ps *peerScore) {
	p := ps.peer
	ps.status = p.Status()
	ps.score = scorePeer(p)
	pl.byScore.update(ps.idx)
}

func scorePeer(p peer.Peer) int64 {
	status := p.Status()
	score := int64(status.PendingRequestCount)
	if status.ConnectionStatus != peer.Available {
		score += int64(unavailablePenalty)
	}
	return score
}
