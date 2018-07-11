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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
)

func TestBenchIdentifier(t *testing.T) {
	peerID := NewPeerIdentifier(-1)
	assert.Equal(t, "-1", peerID.Identifier())
}

func TestBenchIdentifiers(t *testing.T) {
	peerIDs := NewPeerIdentifiers(-1)
	assert.Empty(t, len(peerIDs))
	peerIDs = NewPeerIdentifiers(0)
	assert.Empty(t, len(peerIDs))
	peerIDs = NewPeerIdentifiers(2)
	assert.Equal(t, "0", peerIDs[0].Identifier())
	assert.Equal(t, "1", peerIDs[1].Identifier())
}

type FakePeerSubscriber struct {
	peers    map[string]*Peer
	counters map[string]int
	t        *testing.T
}

func NewFakePeerSubscriber(t *testing.T) *FakePeerSubscriber {
	return &FakePeerSubscriber{
		peers:    make(map[string]*Peer),
		counters: make(map[string]int),
		t:        t,
	}
}

func (sub *FakePeerSubscriber) Register(pid peer.Identifier, p *Peer) {
	sub.peers[pid.Identifier()] = p
}

func (sub *FakePeerSubscriber) NotifyStatusChanged(pid peer.Identifier) {
	id := pid.Identifier()
	_, ok := sub.peers[id]
	require.True(sub.t, ok, fmt.Sprintf(`peer %q not registered`, id))
	sub.counters[id] = int(sub.peers[id].pendingRequests.Load())
}

func TestNewBenchPeer(t *testing.T) {
	p := NewPeer(0, &FakePeerSubscriber{})
	assert.Equal(t, "0", p.Identifier())
}

func TestStartEndRequest(t *testing.T) {
	sub := NewFakePeerSubscriber(t)
	p1 := NewPeer(0, sub)
	p2 := NewPeer(1, sub)
	sub.Register(p1.id, p1)
	sub.Register(p2.id, p2)

	p1.StartRequest()
	p2.StartRequest()
	p2.StartRequest()

	assert.Equal(t, 1, int(p1.pendingRequests.Load()))
	assert.Equal(t, 1, sub.counters[p1.id.Identifier()])

	p1.EndRequest()

	assert.Equal(t, 0, int(p1.pendingRequests.Load()))
	assert.Equal(t, 0, sub.counters[p1.id.Identifier()])
	assert.Equal(t, 2, int(p2.pendingRequests.Load()))
	assert.Equal(t, 2, sub.counters[p2.id.Identifier()])
}
