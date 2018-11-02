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

package yarpcpendingheap

import (
	"go.uber.org/yarpc/api/peer"
)

// peerScore is a book-keeping object for each retained peer
type peerScore struct {
	// immutable after creation
	peer peer.StatusPeer
	heap *pendingHeap
	// mutable
	status peer.Status
	score  int64
	index  int // index in the peer list.
	last   int // snapshot of the heap's incrementing counter.
}

func (ps *peerScore) NotifyStatusChanged(_ peer.Identifier) {
	ps.heap.notifyStatusChanged(ps)
}

func scorePeer(p peer.StatusPeer) int64 {
	status := p.Status()
	score := int64(status.PendingRequestCount)
	return score
}
