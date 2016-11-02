package errors

import "fmt"

// ErrPeerHasNoReferenceToSubscriber is called when a Peer is expected to operate on a PeerSubscriber it has no reference to
type ErrPeerHasNoReferenceToSubscriber struct {
	Peer           interface{}
	PeerSubscriber interface{}
}

func (e ErrPeerHasNoReferenceToSubscriber) Error() string {
	return fmt.Sprintf("peer (%v) has no reference to peer subscriber (%v)", e.Peer, e.PeerSubscriber)
}
