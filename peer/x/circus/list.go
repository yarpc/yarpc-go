// Copyright (c) 2020 Uber Technologies, Inc.
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

package circus

import (
	"context"
	"math/rand"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

var _ peer.List = (*List)(nil)
var _ peer.Chooser = (*List)(nil)

// List is a circus peer list.
type List struct {
	// We limit the width of the peers array to 256 to track offsets compactly.
	// By convention, the first entry is the head of the free list.
	transport peer.Transport
	options   options
	randSrc   rand.Source

	// columns
	// nodes is a block of 256 nodes shared by four circular doubly-linked
	// lists and their head nodes: unavailable, low load, high load, and
	// unallocated (free).
	nodes nodes
	// peers are the respective peers for each node of the list with an
	// associated peer.
	peers [size]peer.Peer
	// statuses are the last known connection statuses of each peer
	statuses [size]peer.ConnectionStatus
	// subscribers are thin structs the circus reuse for receiving messages from
	// clients and transports, when calls end, and when peer connection
	// statuses change.
	subscribers [size]subscriber
	// index tracks the position of all known peers.
	// An entry in this index that is present but zero means that the peer was
	// added, but did not receive a node due to the capacity limit.
	index map[string]uint8

	// hi is the index of the head node of the ring of
	// available peers with likely to have more load.
	hi uint8
	// lo is the index of the head node of the ring of
	// available peers likely to have less load.
	lo uint8

	// sync
	mx                 sync.Mutex
	peerAvailableEvent chan struct{}
}

// New creates a new circus peer list.
func New(transport peer.Transport, opts ...Option) *List {
	options := defaultOptions
	for _, o := range opts {
		o.apply(&options)
	}

	l := &List{
		transport:          transport,
		options:            options,
		nodes:              _zero,
		hi:                 _hi,
		lo:                 _lo,
		index:              make(map[string]uint8),
		peerAvailableEvent: make(chan struct{}, 1),
		randSrc:            rand.NewSource(options.seed),
	}

	// Each of the subscribers must know its corresponding list, index, and capture
	// its onFinish method as a closure.
	// The subscribers are reusable so we can do this initialization up front.
	for index := 0; index < size; index++ {
		subscriber := &l.subscribers[index]
		subscriber.index = uint8(index)
		subscriber.list = l
		subscriber.boundOnFinish = subscriber.onFinish
	}

	return l
}

// Start does nothing.
func (l *List) Start() error {
	return nil
}

// Stop does nothing.
func (l *List) Stop() error {
	return nil
}

// IsRunning returns true, the list is always running.
func (l *List) IsRunning() bool {
	return true
}

// Choose returns a peer.
//
// The caller must call the returned function when they are finished with the
// peer.
//
// This will return an error if the deadline elapses while waiting for the
// peer.
//
// If the circus has no peers and was constructed with the FailFast option, the
// choose method will return an error immediately.
func (l *List) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	// The outer loop is necessary to retry as peers become available.
	// Multiple choose loops can race to get the first peer when it comes online.
	// The retry loop runs outside of a lock, so one choose does not block another.
	// They are synchronized by the peerAvailable event channel.
	for {
		p, onFinish := l.choose()
		if p != nil {
			return p, onFinish, nil
		} else if l.options.failFast {
			return nil, nil, yarpcerrors.Newf(yarpcerrors.CodeUnavailable, _noPeerError, _name)
		}
		if err := l.waitForPeer(ctx); err != nil {
			return nil, nil, err
		}
	}
}

// choose makes a single attempt to get a peer.
// Multiple callers may race to win the lock after a peer becomes available.
// The choose call may return nil, if the ring of available peers is empty.
func (l *List) choose() (peer.Peer, func(error)) {
	l.mx.Lock()
	defer l.mx.Unlock()

	// Bail early if there are no peers in the low load ring.
	// Invariant: if the low concurrency ring is empty, the high concurrency
	// ring must also be empty.
	// We enforce this variant below.
	// If by plucking a peer from the low concurrency ring we have depleted it
	// entirely, we swap the rings.
	if l.nodes.empty(l.lo) {
		return nil, nil
	}

	// Shift the next node in the low concurrency ring to the end of the higher
	// concurrency ring.
	index := l.nodes[l.lo].next
	l.nodes.shift(index, l.hi)

	// If we deplete the entire low concurrency ring, we swap the high
	// concurrency ring.
	// Assuming relatively homogeneous latency for all requests,
	// this would usually occur only when concurrency for the whole list is
	// climbing.
	// When latency is not homogeneous, well, perfect load distribution is the
	// enemy of fast load distribution.
	if l.nodes.empty(l.lo) {
		l.hi, l.lo = l.lo, l.hi
	}

	peer := l.peers[index]
	onFinish := l.subscribers[index].boundOnFinish
	return peer, onFinish
}

func (l *List) onFinish(subscriber *subscriber) {
	l.mx.Lock()
	defer l.mx.Unlock()

	// There is some chance that the onFinish hook will be called long after
	// the underlying node has been moved to another list.
	// This would cause a problem if the peer has been freed, so we abort
	// early in that case.
	// We also abort early if the peer is no longer connected so we don't
	// accidentally put it back into rotation.
	index := subscriber.index
	if p := l.peers[index]; p == nil || p.Status().ConnectionStatus == peer.Unavailable {
		return
	}
	// Event accounting for the above cases, there remains a chance
	// that the peer was released and a new peer was retained, reusing this subscriber.
	// However, the only consequence of proceeding here is being selected later
	// and this should be rare enough as not to cause any material trouble.
	// Certainly less trouble that allocating a unique subscriber for each retained
	// peer and keeping it in a map for the later release.
	l.nodes.shift(index, l.lo)
}

// Update receives changes to the peer list's membership from a source like
// DNS.
func (l *List) Update(updates peer.ListUpdates) error {
	l.mx.Lock()
	defer l.mx.Unlock()

	var errs error

	for _, pid := range updates.Removals {
		addr := pid.Identifier()
		index := l.index[addr]
		// index, exists := l.index[addr]
		// if !exists {
		// 	panic(fmt.Sprintf("remove never added: %s", addr))
		// }
		delete(l.index, addr)
		if index > 0 {
			// if l.peers[index].Identifier() != addr {
			// 	panic(fmt.Sprintf("expected peer identifiers to match: %d %s %s", index, l.peers[index].Identifier(), addr))
			// }
			// remove from index of known peers.
			// release the peer for garbage collection to avoid heap
			// bloat.
			l.peers[index] = nil
			l.statuses[index] = peer.Unavailable
			// move the node to the end of the free list for later
			// reuse.
			l.nodes.shift(uint8(index), _free)
			err := l.transport.ReleasePeer(pid, &l.subscribers[index])
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	additions := updates.Additions
	if !l.options.noShuffle {
		additions = shuffle(l.randSrc, additions)
	}

	for _, pid := range additions {
		addr := pid.Identifier()
		// if _, exists := l.index[addr]; exists {
		// 	panic(fmt.Sprintf("add already added: %s", addr))
		// }
		l.index[addr] = 0
		// The maximum capacity of this peer list is 252 peers.
		// Beyond that, this list ignores all further peers.
		// TODO consider forcibly thinning out the low ring or randomly
		// sub-setting the additions and borrow a few tricks from reservoir
		// sampling.
		if l.nodes.empty(_free) {
			continue
		}
		index := l.nodes[_free].next
		peer, err := l.transport.RetainPeer(pid, &l.subscribers[index])
		if err != nil {
			errs = multierr.Append(errs, err)
		} else {
			l.index[addr] = index
			l.peers[index] = peer
			// Rotate the peer out of the free list.
			l.nodes.shift(index, _no)
			// Move to available list immediately if possible.
			l.updateStatus(index, pid)
		}
	}

	return errs
}

// notifyStatusChanged calls come from the transport when a peer connects or
// disconnects, via a subscriber that keeps track of the index of the peer in our
// list's internal tables.
// These calls move a peer between the low concurrency and unavailable lists
// depending on their connection status.
func (l *List) notifyStatusChanged(subscriber *subscriber, pid peer.Identifier) {
	l.mx.Lock()
	defer l.mx.Unlock()

	index := subscriber.index
	p := l.peers[index]
	if p == nil || p.Identifier() != pid.Identifier() {
		return
	}
	l.updateStatus(index, pid)
}

// updateStatus moves an individual peer between rings of the circus depending
// on the peer's connection status.
func (l *List) updateStatus(index uint8, pid peer.Identifier) {
	status := l.peers[index].Status().ConnectionStatus
	// Exit early if the peer is not changing status.
	if l.statuses[index] == status {
		return
	}
	l.statuses[index] = status

	if status == peer.Available {
		// Becoming available moves a peer to the end of the low concurrency ring.
		// The peer should not have any pending requests when it becomes
		// available but it is not tragic update loses a race.
		l.nodes.shift(index, l.lo)
		l.notifyPeerAvailable()
	} else {
		// Becoming unavailable unconditionally moves the peer from either the
		// high or low concurrency ring to the unavailable ring.
		l.nodes.shift(index, _no)
	}
}

// notifyPeerAvailable writes to a channel indicating that a Peer is currently
// available for requests.
// This allows all choosers who tried to obtain a peer when there were none
// available to race to grab the new one.
func (l *List) notifyPeerAvailable() {
	select {
	case l.peerAvailableEvent <- struct{}{}:
	default:
	}
}

// waitForPeer waits until a peer is added to the peer list or the
// given context finishes.
// MUST NOT be run under a lock.
func (l *List) waitForPeer(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return l.newNoContextDeadlineError()
	}

	select {
	case <-l.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return l.newUnavailableError(ctx.Err())
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

func (l *List) newNoContextDeadlineError() error {
	return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, _noContextDeadlineError, _name)
}

func (l *List) newUnavailableError(err error) error {
	return yarpcerrors.Newf(yarpcerrors.CodeUnavailable, _unavailableError, _name, err.Error())
}
