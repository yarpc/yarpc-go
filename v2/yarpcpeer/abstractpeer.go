package yarpcpeer

import (
	"sync"

	"go.uber.org/atomic"
)

// NewAbstractPeer creates a new AbstractPeer from any identifier transport,
// and subscriber.
func NewAbstractPeer(pid Identifier, transport Transport) *AbstractPeer {
	p := &AbstractPeer{
		pid:         pid,
		transport:   transport,
		subscribers: make(map[Subscriber]struct{}),
	}
	p.connectionStatus.Store(int32(Unavailable))
	return p
}

// AbstractPeer keeps a subscriber to send status updates to it, and the Transport that created it
type AbstractPeer struct {
	lock sync.RWMutex

	pid              Identifier
	transport        Transport
	subscribers      map[Subscriber]struct{}
	pending          atomic.Int32
	connectionStatus atomic.Int32
}

// Identifier returns the corresponding peer identifier string.
func (p *AbstractPeer) Identifier() string {
	return p.pid.Identifier()
}

// Transport returns the Transport that is in charge of this Peer (and should be the one to handle requests)
func (p *AbstractPeer) Transport() Transport {
	return p.transport
}

// Subscribe adds a subscriber to the peer's subscriber map
func (p *AbstractPeer) Subscribe(sub Subscriber) {
	p.lock.Lock()
	p.subscribers[sub] = struct{}{}
	p.lock.Unlock()
}

// Unsubscribe removes a subscriber from the peer's subscriber map
func (p *AbstractPeer) Unsubscribe(sub Subscriber) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.subscribers[sub]; !ok {
		return ErrPeerHasNoReferenceToSubscriber{
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
func (p *AbstractPeer) Status() Status {
	return Status{
		PendingRequestCount: int(p.pending.Load()),
		ConnectionStatus:    ConnectionStatus(p.connectionStatus.Load()),
	}
}

// SetStatus sets the status of the Peer (to be used by the Transport)
func (p *AbstractPeer) SetStatus(status ConnectionStatus) {
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
	subs := make([]Subscriber, 0, len(p.subscribers))
	for sub := range p.subscribers {
		subs = append(subs, sub)
	}
	p.lock.RUnlock()

	for _, sub := range subs {
		sub.NotifyStatusChanged(p.pid)
	}
}
