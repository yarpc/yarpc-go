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

import "go.uber.org/yarpc/api/peer"

// peerScore is a book-keeping object for each retained peer and
// gets
type peerScore struct {
	// immutable after creation
	peer        peer.Peer
	id          peer.Identifier
	list        *List
	boundFinish func(error)

	status peer.Status
	score  int64
	idx    int // index in the peer list.
	last   int // snapshot of the heap's incrementing counter.
}

func (ps *peerScore) NotifyStatusChanged(_ peer.Identifier) {
	if ps == nil || ps.list == nil {
		// The subscriber is no longer retained by the peer list, or caller
		// obtained a nil subscriber (happens in tests).
		return
	}
	status := ps.peer.Status()
	if ps.status == status {
		return
	}
	ps.status = status
	ps.list.peerScoreChanged(ps)
}

func (ps *peerScore) finish(error) {
	ps.peer.EndRequest()
}
