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

// Package yarpcpeer provides an abstract peer implementation as a utility for
// transport implementations.
package yarpcpeer

import (
	"sync"

	"go.uber.org/atomic"
	yarpc "go.uber.org/yarpc/v2"
)

// NewAbstractPeer creates a new AbstractPeer from any identifier.
func NewAbstractPeer(id yarpc.Identifier) *AbstractPeer {
	p := &AbstractPeer{
		id:          id,
		subscribers: make(map[yarpc.Subscriber]int),
	}
	p.connectionStatus.Store(int32(yarpc.Unavailable))
	return p
}

// AbstractPeer tracks peer subscribers.
type AbstractPeer struct {
	lock sync.RWMutex

	id               yarpc.Identifier
	subscribers      map[yarpc.Subscriber]int
	numSubscribers   atomic.Int32
	pending          atomic.Int32
	connectionStatus atomic.Int32
}

// Identifier returns the corresponding peer identifier string.
func (p *AbstractPeer) Identifier() string {
	return p.id.Identifier()
}

// Subscribe adds a subscriber to the peer's subscriber map
func (p *AbstractPeer) Subscribe(sub yarpc.Subscriber) {
	p.lock.Lock()
	p.subscribers[sub]++
	p.numSubscribers.Inc()
	p.lock.Unlock()
}

// Unsubscribe removes a subscriber from the peer's subscriber map
func (p *AbstractPeer) Unsubscribe(sub yarpc.Subscriber) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.subscribers[sub]; !ok {
		return ErrPeerHasNoReferenceToSubscriber{
			PeerIdentifier: p.id,
			PeerSubscriber: sub,
		}
	}

	p.subscribers[sub]--
	if p.subscribers[sub] == 0 {
		delete(p.subscribers, sub)
	}
	p.numSubscribers.Dec()
	return nil
}

// NumSubscribers returns the number of subscriptions attached to the peer
func (p *AbstractPeer) NumSubscribers() int {
	return int(p.numSubscribers.Load())
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
		sub.NotifyStatusChanged(p.id)
	}
}
