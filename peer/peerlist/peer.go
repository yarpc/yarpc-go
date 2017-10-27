package peerlist

import (
	"sync"

	"go.uber.org/yarpc/api/peer"
)

// peerThunk captures a peer and its corresponding subscriber,
// and serves as a subscriber by proxy.
type peerThunk struct {
	lock       sync.RWMutex
	list       *List
	id         peer.Identifier
	peer       peer.Peer
	subscriber peer.Subscriber
}

// NotifyStatusChanged forwards a status notification to the peer list and to
// the underlying identifier chooser list.
func (t *peerThunk) NotifyStatusChanged(pid peer.Identifier) {
	t.lock.RLock()
	pl := t.list
	s := t.subscriber
	t.lock.RUnlock()

	pl.NotifyStatusChanged(pid)

	if s != nil {
		s.NotifyStatusChanged(pid)
	}
}

// SetSubscriber assigns a subscriber to the subscriber thunk.
func (t *peerThunk) SetSubscriber(s peer.Subscriber) {
	t.lock.Lock()
	t.subscriber = s
	t.lock.Unlock()
}

// Peer returns the peer.
func (t *peerThunk) Peer() peer.Peer {
	t.lock.RLock()
	p := t.peer
	t.lock.RUnlock()
	return p
}

// SetPeer assigns the peer.
func (t *peerThunk) SetPeer(p peer.Peer) {
	t.lock.Lock()
	t.peer = p
	t.subscriber = nil
	t.lock.Unlock()
}

// Get captures a snapshot of id, peer, and subscriber.
func (t *peerThunk) Get() (peer.Identifier, peer.Peer, peer.Subscriber) {
	t.lock.RLock()
	i := t.id
	p := t.peer
	s := t.subscriber
	t.lock.RUnlock()
	return i, p, s
}
