// Copyright (c) 2026 Uber Technologies, Inc.
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
	"container/heap"
	"fmt"
)

type peerHeap struct {
	peers []*peerScore

	// next is an incrementing counter for every push, which is compared when
	// scores are equal. This ends up implementing round-robin when scores are
	// equal.
	next int
}

func (ph *peerHeap) Len() int {
	return len(ph.peers)
}

// Less returns whether the left peer has a lower score. If the scores are
// equal, it returns the older peer (where "last" is lower.)
func (ph *peerHeap) Less(i, j int) bool {
	p1 := ph.peers[i]
	p2 := ph.peers[j]
	if p1.score == p2.score {
		return p1.last < p2.last
	}
	return p1.score < p2.score
}

// Swap implements the heap.Interface. Do NOT use this method directly.
func (ph *peerHeap) Swap(i, j int) {
	p1 := ph.peers[i]
	p2 := ph.peers[j]

	ph.peers[i], ph.peers[j] = ph.peers[j], ph.peers[i]
	p1.idx = j
	p2.idx = i
}

// Push implements the heap.Interface. Do NOT use this method directly.
// Use pushPeer instead.
func (ph *peerHeap) Push(x interface{}) {
	ps := x.(*peerScore)
	ps.idx = len(ph.peers)
	ph.peers = append(ph.peers, ps)
}

// Pop implements the heap.Interface. Do NOT use this method directly.
// Use popPeer instead.
func (ph *peerHeap) Pop() interface{} {
	lastIdx := len(ph.peers) - 1
	last := ph.peers[lastIdx]
	ph.peers = ph.peers[:lastIdx]
	return last
}

func (ph *peerHeap) delete(idx int) {
	// Swap the element we want to delete with the last element, then pop it off.
	ph.Swap(idx, ph.Len()-1)
	ph.Pop()

	// If the original index still exists in the list, it contains a different
	// element so update the heap.
	if idx < ph.Len() {
		ph.update(idx)
	}
}

func (ph *peerHeap) validate(ps *peerScore) error {
	if ps.idx < 0 || ps.idx >= ph.Len() || ph.peers[ps.idx] != ps {
		return fmt.Errorf("peerHeap bug: %+v has bad index %v (len %v)", ps, ps.idx, ph.Len())
	}
	return nil
}

func (ph *peerHeap) pushPeer(ps *peerScore) {
	ph.next++
	ps.last = ph.next
	heap.Push(ph, ps)
}

func (ph *peerHeap) peekPeer() (*peerScore, bool) {
	if ph.Len() == 0 {
		return nil, false
	}
	return ph.peers[0], true
}

func (ph *peerHeap) popPeer() (*peerScore, bool) {
	if ph.Len() == 0 {
		return nil, false
	}

	peer := heap.Pop(ph).(*peerScore)
	return peer, true
}

func (ph *peerHeap) update(i int) {
	heap.Fix(ph, i)
}
