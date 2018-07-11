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

package chooserbenchmark

import (
	"strconv"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
)

var _ peer.Identifier = (*PeerIdentifier)(nil)

// PeerIdentifier uses an integer to uniquely identify a peer
type PeerIdentifier struct {
	id int
}

// NewPeerIdentifier create a peer identifier with given id
func NewPeerIdentifier(id int) peer.Identifier {
	return PeerIdentifier{id: id}
}

// NewPeerIdentifiers create n peer identifiers, with IDs 0 to n-1
func NewPeerIdentifiers(n int) []peer.Identifier {
	if n <= 0 {
		return nil
	}
	ids := make([]peer.Identifier, n)
	for i := 0; i < n; i++ {
		ids[i] = NewPeerIdentifier(i)
	}
	return ids
}

// Identifier return unique string that identify the peer
func (p PeerIdentifier) Identifier() string {
	return strconv.Itoa(p.id)
}

var _ peer.Peer = (*Peer)(nil)

// Peer is a minimum implementation to mock YARPC peers
type Peer struct {
	id              PeerIdentifier
	pendingRequests atomic.Int64
	sub             peer.Subscriber
}

// Identifier use BenchIdentifier to generate a unique string
func (p *Peer) Identifier() string {
	return p.id.Identifier()
}

// NewPeer creates a BenchPeer from an integer and peer.Subscriber
func NewPeer(id int, ps peer.Subscriber) *Peer {
	p := &Peer{
		id:  PeerIdentifier{id: id},
		sub: ps,
	}
	return p
}

// Status returns the current status of the BenchPeer.
// pending request count is for policies like fewest pending request
// connection status is always available, no adding peers, peer fails for now
func (p *Peer) Status() peer.Status {
	return peer.Status{
		PendingRequestCount: int(p.pendingRequests.Load()),
		ConnectionStatus:    peer.Available,
	}
}

// StartRequest indicates a request has started
func (p *Peer) StartRequest() {
	p.pendingRequests.Inc()
	p.sub.NotifyStatusChanged(p.id)
}

// EndRequest indicates a request has ended
func (p *Peer) EndRequest() {
	p.pendingRequests.Dec()
	p.sub.NotifyStatusChanged(p.id)
}
