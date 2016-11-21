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
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/internal/errors"

	"go.uber.org/atomic"
)

// PeerIdentifier uniquely references a host:port combination using a common interface
type PeerIdentifier string

// Identifier generates a (should be) unique identifier for this PeerIdentifier (to use in maps, etc)
func (p PeerIdentifier) Identifier() string {
	return string(p)
}

// NewPeer creates a new hostport.Peer from a hostport.PeerIdentifier, transport.Agent, and transport.PeerSubscriber
func NewPeer(pid PeerIdentifier, agent transport.Agent) *Peer {
	return &Peer{
		PeerIdentifier:   pid,
		agent:            agent,
		subscribers:      make(map[transport.PeerSubscriber]struct{}),
		connectionStatus: transport.PeerUnavailable,
	}
}

// Peer keeps a subscriber to send status updates to it, and the PeerAgent that created it
type Peer struct {
	PeerIdentifier

	agent            transport.Agent
	subscribers      map[transport.PeerSubscriber]struct{}
	pending          atomic.Int32
	connectionStatus transport.PeerConnectionStatus
}

// HostPort surfaces the HostPort in this function, if you want to access the hostport directly (for a downstream call)
// You need to cast the transport.Peer to a *hostport.Peer and run this function
func (p *Peer) HostPort() string {
	return string(p.PeerIdentifier)
}

// Agent returns the Agent that is in charge of this hostport.Peer (and should be the one to handle requests)
func (p *Peer) Agent() transport.Agent {
	return p.agent
}

// AddSubscriber adds a subscriber to the peer's subscriber map
// This function isn't thread safe
func (p *Peer) AddSubscriber(sub transport.PeerSubscriber) {
	p.subscribers[sub] = struct{}{}
}

// RemoveSubscriber removes a subscriber from the peer's subscriber map
// This function isn't thread safe
func (p *Peer) RemoveSubscriber(sub transport.PeerSubscriber) error {
	if _, ok := p.subscribers[sub]; !ok {
		return errors.ErrPeerHasNoReferenceToSubscriber{
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
func (p *Peer) Status() transport.PeerStatus {
	return transport.PeerStatus{
		PendingRequestCount: int(p.pending.Load()),
		ConnectionStatus:    p.connectionStatus,
	}
}

// SetStatus sets the status of the Peer (to be used by the Agent)
func (p *Peer) SetStatus(status transport.PeerConnectionStatus) {
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
