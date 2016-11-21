// Copyright (c) 2016 Uber Technologies, Inc.
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

package hostport

import (
	"go.uber.org/yarpc/peer"

	"go.uber.org/atomic"
)

// PeerIdentifier uniquely references a host:port combination using a common interface
type PeerIdentifier string

// Identifier generates a (should be) unique identifier for this PeerIdentifier (to use in maps, etc)
func (p PeerIdentifier) Identifier() string {
	return string(p)
}

// NewPeer creates a new hostport.Peer from a hostport.PeerIdentifier, peer.Agent, and peer.Subscriber
func NewPeer(pid PeerIdentifier, agent peer.Agent) *Peer {
	return &Peer{
		PeerIdentifier:   pid,
		agent:            agent,
		subscribers:      make(map[peer.Subscriber]struct{}),
		connectionStatus: peer.Unavailable,
	}
}

// Peer keeps a subscriber to send status updates to it, and the peer.Agent that created it
type Peer struct {
	PeerIdentifier

	agent            peer.Agent
	subscribers      map[peer.Subscriber]struct{}
	pending          atomic.Int32
	connectionStatus peer.ConnectionStatus
}

// HostPort surfaces the HostPort in this function, if you want to access the hostport directly (for a downstream call)
// You need to cast the Peer to a *hostport.Peer and run this function
func (p *Peer) HostPort() string {
	return string(p.PeerIdentifier)
}

// Agent returns the peer.Agent that is in charge of this hostport.Peer (and should be the one to handle requests)
func (p *Peer) Agent() peer.Agent {
	return p.agent
}

// AddSubscriber adds a subscriber to the peer's subscriber map
// This function isn't thread safe
func (p *Peer) AddSubscriber(sub peer.Subscriber) {
	p.subscribers[sub] = struct{}{}
}

// RemoveSubscriber removes a subscriber from the peer's subscriber map
// This function isn't thread safe
func (p *Peer) RemoveSubscriber(sub peer.Subscriber) error {
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
// This function isn't thread safe
func (p *Peer) NumSubscribers() int {
	return len(p.subscribers)
}

// Status returns the current status of the hostport.Peer
func (p *Peer) Status() peer.Status {
	return peer.Status{
		PendingRequestCount: int(p.pending.Load()),
		ConnectionStatus:    p.connectionStatus,
	}
}

// SetStatus sets the status of the Peer (to be used by the peer.Agent)
func (p *Peer) SetStatus(status peer.ConnectionStatus) {
	p.connectionStatus = status
	p.notifyStatusChanged()
}

// StartRequest runs at the beginning of a request and returns a callback for when the request finished
func (p *Peer) StartRequest() func() {
	p.pending.Inc()
	p.notifyStatusChanged()
	return p.endRequest
}

// endRequest should be run after a request has finished
func (p *Peer) endRequest() {
	p.pending.Dec()
	p.notifyStatusChanged()
}

func (p *Peer) notifyStatusChanged() {
	for sub := range p.subscribers {
		sub.NotifyStatusChanged(p)
	}
}
