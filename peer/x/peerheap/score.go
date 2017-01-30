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
	ps.list.notifyStatusChanged(ps)
}

func (ps *peerScore) finish(error) {
	ps.peer.EndRequest()
}
