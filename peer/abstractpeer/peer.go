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

package abstractpeer

import (
	"sync"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
)

// PeerIdentifier uniquely references a host:port combination using a common interface
type PeerIdentifier string

// Identifier generates a (should be) unique identifier for this PeerIdentifier (to use in maps, etc)
func (p PeerIdentifier) Identifier() string {
	return string(p)
}

// Identify coerces a string to a PeerIdentifier
func Identify(peer string) peer.Identifier {
	return PeerIdentifier(peer)
}

// NewPeer creates a new abstractpeer.Peer from a abstractpeer.PeerIdentifier, peer.Transport, and peer.Subscriber
func NewPeer(pid PeerIdentifier, transport peer.Transport) *Peer {
	p := &Peer{
		PeerIdentifier: pid,
		transport:      transport,
		subscribers:    make(map[peer.Subscriber]struct{}),
	}
	p.connectionStatus.Store(int32(peer.Unavailable))
	return p
}

// Peer keeps a subscriber to send status updates to it, and the peer.Transport that created it
type Peer struct {
	PeerIdentifier

	lock sync.RWMutex

	transport        peer.Transport
	subscribers      map[peer.Subscriber]struct{}
	pending          atomic.Int32
	connectionStatus atomic.Int32
}

// HostPort surfaces the HostPort in this function, if you want to access the hostport directly (for a downstream call)
// You need to cast the Peer to a *abstractpeer.Peer and run this function
func (p *Peer) HostPort() string {
	return string(p.PeerIdentifier)
}

// Transport returns the peer.Transport that is in charge of this abstractpeer.Peer (and should be the one to handle requests)
func (p *Peer) Transport() peer.Transport {
	return p.transport
}

// Subscribe adds a subscriber to the peer's subscriber map
func (p *Peer) Subscribe(sub peer.Subscriber) {
	p.lock.Lock()
	p.subscribers[sub] = struct{}{}
	p.lock.Unlock()
}

// Unsubscribe removes a subscriber from the peer's subscriber map
func (p *Peer) Unsubscribe(sub peer.Subscriber) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.subscribers[sub]; !ok {
		return peer.ErrPeerHasNoReferenceToSubscriber{
			PeerIdentifier: p.PeerIdentifier,
			PeerSubscriber: sub,
		}
	}

	delete(p.subscribers, sub)
	return nil
}

// NumSubscribers returns the number of subscriptions attached to the peer
func (p *Peer) NumSubscribers() int {
	p.lock.RLock()
	subs := len(p.subscribers)
	p.lock.RUnlock()
	return subs
}

// Status returns the current status of the abstractpeer.Peer
func (p *Peer) Status() peer.Status {
	return peer.Status{
		PendingRequestCount: int(p.pending.Load()),
		ConnectionStatus:    peer.ConnectionStatus(p.connectionStatus.Load()),
	}
}

// SetStatus sets the status of the Peer (to be used by the peer.Transport)
func (p *Peer) SetStatus(status peer.ConnectionStatus) {
	p.connectionStatus.Store(int32(status))
}

// StartRequest runs at the beginning of a request.
func (p *Peer) StartRequest() {
	p.pending.Inc()
}

// EndRequest should be run after a request has finished.
func (p *Peer) EndRequest() {
	p.pending.Dec()
}

// NotifyStatusChanged broadcasts a status change notification to all
// subscribers.
func (p *Peer) NotifyStatusChanged() {
	p.lock.RLock()
	subs := make([]peer.Subscriber, 0, len(p.subscribers))
	for sub := range p.subscribers {
		subs = append(subs, sub)
	}
	p.lock.RUnlock()

	for _, sub := range subs {
		sub.NotifyStatusChanged(p)
	}
}
