package peerlist

import (
	"sync"

	"go.uber.org/yarpc/api/peer"
)

// peerThunk captures a peer and its corresponding subscriber,
// and serves as a subscriber by proxy.
type peerThunk struct {
	lock          sync.RWMutex
	list          *List
	id            peer.Identifier
	peer          peer.Peer
	subscriber    peer.Subscriber
	boundOnFinish func(error)
}

func (t *peerThunk) onStart() {
	t.peer.StartRequest()
	t.Subscriber().NotifyStatusChanged(t.id)
}

func (t *peerThunk) onFinish(error) {
	t.peer.EndRequest()
	t.Subscriber().NotifyStatusChanged(t.id)
}

func (t *peerThunk) Identifier() string {
	return t.peer.Identifier()
}

func (t *peerThunk) Status() peer.Status {
	return t.peer.Status()
}

func (t *peerThunk) StartRequest() {
	t.peer.StartRequest()
}

func (t *peerThunk) EndRequest() {
	t.peer.EndRequest()
}

// NotifyStatusChanged forwards a status notification to the peer list and to
// the underlying identifier chooser list.
func (t *peerThunk) NotifyStatusChanged(pid peer.Identifier) {
	t.list.notifyStatusChanged(pid)

	s := t.Subscriber()
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

// Subscriber returns the subscriber.
func (t *peerThunk) Subscriber() peer.Subscriber {
	t.lock.RLock()
	s := t.subscriber
	t.lock.RUnlock()
	return s
}
