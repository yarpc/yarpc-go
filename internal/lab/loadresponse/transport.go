// Copyright (c) 2020 Uber Technologies, Inc.
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

package main

import (
	"strconv"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
)

var _ peer.Identifier = Identifier(0)

// Identifier is a peer identifier for a member of the server cluster for
// purposes of simulation.
type Identifier int

// Identifier returns the decimal representation of the server number for a
// peer.
func (i Identifier) Identifier() string {
	return strconv.Itoa(int(i))
}

var _ peer.Peer = (*Peer)(nil)

// Peer is a simulated transport peer for a server, capturing the per client
// state for a member of the server cluster.
type Peer struct {
	ID                  int
	Server              *Server
	PendingRequestCount atomic.Int32
}

// Identifier returns the peer identifier of a server.
func (p *Peer) Identifier() string {
	return strconv.Itoa(p.ID)
}

// Status consistently returns the status and pending request count for the
// server.
//
// Peer lists no longer use the peer's pending request count tracking.
// This is internal to the abstract peer list.
func (p *Peer) Status() peer.Status {
	return peer.Status{
		ConnectionStatus:    peer.Available,
		PendingRequestCount: int(p.PendingRequestCount.Load()),
	}
}

// StartRequest marks the beginning of a request.
func (p *Peer) StartRequest() {
	p.PendingRequestCount.Inc()
}

// EndRequest marks the end of a request.
func (p *Peer) EndRequest() {
	p.PendingRequestCount.Dec()
}

var _ peer.Transport = (*Transport)(nil)

// Transport is a simulated transport.
type Transport struct {
	Peers []Peer
}

// NewTransport creates a transport for the given simulated servers.
func NewTransport(servers []Server) *Transport {
	count := len(servers)
	t := &Transport{
		Peers: make([]Peer, count),
	}
	for i := 0; i < count; i++ {
		t.Peers[i].ID = i
		t.Peers[i].Server = &servers[i]
	}
	return t
}

// RetainPeer returns the peer for the identified server.
func (t *Transport) RetainPeer(id peer.Identifier, _ peer.Subscriber) (peer.Peer, error) {
	i, err := strconv.Atoi(id.Identifier())
	if err != nil {
		return nil, err
	}
	return &t.Peers[i], nil
}

// ReleasePeer does nothing.
//
// The simulated peers are static.
func (t *Transport) ReleasePeer(peer.Identifier, peer.Subscriber) error {
	return nil
}
