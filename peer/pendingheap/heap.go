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

package pendingheap

import (
	"container/heap"
	"context"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/peerlist/v2"
)

type pendingHeap struct {
	sync.Mutex

	peers []*peerScore

	// next is an incrementing counter for every push, which is compared when
	// scores are equal. This ends up implementing round-robin when scores are
	// equal.
	next int

	// nextRand is used for random insertions among equally scored peers when new
	// peers are added.
	//
	// nextRand MUST return a number in [0, numPeers)
	nextRand func(numPeers int) int
}

var _ peerlist.Implementation = (*pendingHeap)(nil)

func (ph *pendingHeap) Choose(ctx context.Context, req *transport.Request) peer.StatusPeer {
	ph.Lock()
	ps, ok := ph.popPeer()
	if !ok {
		ph.Unlock()
		return nil
	}

	// Note: We push the peer back to reset the "next" counter.
	// This gives us round-robin behavior.
	ph.pushPeer(ps)

	ph.Unlock()
	return ps.peer
}

func (ph *pendingHeap) Add(p peer.StatusPeer, _ peer.Identifier) peer.Subscriber {
	if p == nil {
		return nil
	}

	ps := &peerScore{peer: p, heap: ph}
	ps.score = scorePeer(p)

	ph.Lock()
	ph.pushPeerRandom(ps)
	ph.Unlock()
	return ps
}

func (ph *pendingHeap) Remove(p peer.StatusPeer, _ peer.Identifier, sub peer.Subscriber) {
	ps, ok := sub.(*peerScore)
	if !ok {
		return
	}

	ph.Lock()
	ph.delete(ps)
	ph.Unlock()
}

func (ph *pendingHeap) notifyStatusChanged(ps *peerScore) {
	status := ps.peer.Status()
	ph.Lock()
	// If the index is negative, the subscriber has already been deleted from the
	// heap. This may occur when calling a peer and simultaneously removing it
	// from the heap.
	if ps.index < 0 {
		ph.Unlock()
		return
	}

	ps.status = status
	ps.score = scorePeer(ps.peer)
	ph.update(ps.index)
	ph.Unlock()
}

// Len must be called in the context of a lock, as it is indirectly called
// through heap.Push and heap.Pop.
func (ph *pendingHeap) Len() int {
	return len(ph.peers)
}

// Less returns whether the left peer has a lower score. If the scores are
// equal, it returns the older peer (where "last" is lower.)
// Less must be called in the context of a lock, as it is indirectly called
// through heap.Push and heap.Pop.
func (ph *pendingHeap) Less(i, j int) bool {
	p1 := ph.peers[i]
	p2 := ph.peers[j]
	if p1.score == p2.score {
		return p1.last < p2.last
	}
	return p1.score < p2.score
}

// Swap implements the heap.Interface. Do NOT use this method directly.
// Swap must be called in the context of a lock, as it is indirectly called
// through heap.Push and heap.Pop.
func (ph *pendingHeap) Swap(i, j int) {
	p1 := ph.peers[i]
	p2 := ph.peers[j]

	ph.peers[i], ph.peers[j] = ph.peers[j], ph.peers[i]
	p1.index = j
	p2.index = i
}

// Push implements the heap.Interface. Do NOT use this method directly.
// Use pushPeer instead.
// Push must be called in the context of a lock, as it is indirectly called
// through heap.Push.
func (ph *pendingHeap) Push(x interface{}) {
	ps := x.(*peerScore)
	ps.index = len(ph.peers)
	ph.peers = append(ph.peers, ps)
}

// Pop implements the heap.Interface. Do NOT use this method directly.
// Use popPeer instead.
// Pop must be called in the context of a lock, as it is indirectly called
// through heap.Pop.
func (ph *pendingHeap) Pop() interface{} {
	lastIndex := len(ph.peers) - 1
	last := ph.peers[lastIndex]
	ph.peers = ph.peers[:lastIndex]
	return last
}

// delete removes the score at the given index.
// delete must be called in a lock.
func (ph *pendingHeap) delete(ps *peerScore) {
	if ps.index < 0 {
		return
	}
	index := ps.index

	// Swap the element we want to delete with the last element, then pop it off.
	ph.Swap(index, ph.Len()-1)
	ph.Pop()

	// If the original index still exists in the list, it contains a different
	// element so update the heap.
	if index < ph.Len() {
		ph.update(index)
	}

	// Set the index to negative so we do not try to delete it again.
	ps.index = -1
}

// pushPeer must be called in the context of a lock.
func (ph *pendingHeap) pushPeer(ps *peerScore) {
	ph.next++
	ps.last = ph.next
	heap.Push(ph, ps)
}

// pushPeerRandom inserts a peer randomly into the heap among equally scored
// peers. This is expected to be called only by Add.
//
// This ensures that batches of peer updates are inserted randomly throughout
// the heap, preventing hearding behavior that may be observed during batch
// deployments.
//
// This must be called in the context of a lock.
func (ph *pendingHeap) pushPeerRandom(ps *peerScore) {
	ph.next++
	ps.last = ph.next

	random := ph.nextRand(len(ph.peers) + 1)
	if random < len(ph.peers) {
		randPeer := ph.peers[random]
		ps.last, randPeer.last = randPeer.last, ps.last
		heap.Fix(ph, random)
	}

	heap.Push(ph, ps)
}

// popPeer must be called in the context of a lock.
func (ph *pendingHeap) popPeer() (*peerScore, bool) {
	if ph.Len() == 0 {
		return nil, false
	}

	peer := heap.Pop(ph).(*peerScore)
	return peer, true
}

// update must be called in the context of a lock.
func (ph *pendingHeap) update(i int) {
	heap.Fix(ph, i)
}

func (ph *pendingHeap) Start() error {
	return nil
}

func (ph *pendingHeap) Stop() error {
	return nil
}

func (ph *pendingHeap) IsRunning() bool {
	return true
}
