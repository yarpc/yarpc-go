// Package yarpcpeer provides an abstract peer implementation as a utility for
// transport implementations.
package yarpcpeer

import (
	"sync"

	"go.uber.org/atomic"
	yarpc "go.uber.org/yarpc/v2"
)

// NewAbstractPeer creates a new AbstractPeer from any identifier.
func NewAbstractPeer(pid yarpc.Identifier) *AbstractPeer {
	p := &AbstractPeer{
		pid:         pid,
		subscribers: make(map[yarpc.Subscriber]struct{}),
	}
	p.connectionStatus.Store(int32(yarpc.Unavailable))
	return p
}

// AbstractPeer tracks peer subscribers.
type AbstractPeer struct {
	lock sync.RWMutex

	pid              yarpc.Identifier
	subscribers      map[yarpc.Subscriber]struct{}
	pending          atomic.Int32
	connectionStatus atomic.Int32
}

// Identifier returns the corresponding peer identifier string.
func (p *AbstractPeer) Identifier() string {
	return p.pid.Identifier()
}

// Subscribe adds a subscriber to the peer's subscriber map
func (p *AbstractPeer) Subscribe(sub yarpc.Subscriber) {
	p.lock.Lock()
	p.subscribers[sub] = struct{}{}
	p.lock.Unlock()
}

// Unsubscribe removes a subscriber from the peer's subscriber map
func (p *AbstractPeer) Unsubscribe(sub yarpc.Subscriber) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.subscribers[sub]; !ok {
		return yarpc.ErrPeerHasNoReferenceToSubscriber{
			PeerIdentifier: p.pid,
			PeerSubscriber: sub,
		}
	}

	delete(p.subscribers, sub)
	return nil
}

// NumSubscribers returns the number of subscriptions attached to the peer
func (p *AbstractPeer) NumSubscribers() int {
	p.lock.RLock()
	subs := len(p.subscribers)
	p.lock.RUnlock()
	return subs
}

// Status returns the current status of the Peer
func (p *AbstractPeer) Status() yarpc.Status {
	return yarpc.Status{
		PendingRequestCount: int(p.pending.Load()),
		ConnectionStatus:    yarpc.ConnectionStatus(p.connectionStatus.Load()),
	}
}

// SetStatus sets the status of the Peer (to be used by the transport)
func (p *AbstractPeer) SetStatus(status yarpc.ConnectionStatus) {
	p.connectionStatus.Store(int32(status))
	p.notifyStatusChanged()
}

// StartRequest runs at the beginning of a request and returns a callback for when the request finished
func (p *AbstractPeer) StartRequest() {
	p.pending.Inc()
	p.notifyStatusChanged()
}

// EndRequest should be run after a request has finished.
func (p *AbstractPeer) EndRequest() {
	p.pending.Dec()
	p.notifyStatusChanged()
}

func (p *AbstractPeer) notifyStatusChanged() {
	p.lock.RLock()
	subs := make([]yarpc.Subscriber, 0, len(p.subscribers))
	for sub := range p.subscribers {
		subs = append(subs, sub)
	}
	p.lock.RUnlock()

	for _, sub := range subs {
		sub.NotifyStatusChanged(p.pid)
	}
}
