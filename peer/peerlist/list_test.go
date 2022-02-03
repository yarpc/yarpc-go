// Copyright (c) 2022 Uber Technologies, Inc.
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

package peerlist

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpctest"
)

const (
	id1 = hostport.PeerIdentifier("1.2.3.4:1234")
	id2 = hostport.PeerIdentifier("4.3.2.1:4321")
	id3 = hostport.PeerIdentifier("1.1.1.1:1111")
)

func TestValues(t *testing.T) {
	vs := values(map[string]peer.Identifier{})
	assert.Equal(t, []peer.Identifier{}, vs)

	vs = values(map[string]peer.Identifier{"_": id1, "__": id2})
	assert.Equal(t, 2, len(vs))
	assert.Contains(t, vs, id1)
	assert.Contains(t, vs, id2)
}

func TestShuffle(t *testing.T) {
	for _, test := range []struct {
		msg  string
		seed int64
		in   []peer.Identifier
		want []peer.Identifier
	}{
		{
			"empty",
			0,
			[]peer.Identifier{},
			[]peer.Identifier{},
		},
		{
			"some",
			0,
			[]peer.Identifier{id1, id2, id3},
			[]peer.Identifier{id2, id3, id1},
		},
		{
			"different seed",
			7,
			[]peer.Identifier{id1, id2, id3},
			[]peer.Identifier{id2, id1, id3},
		},
	} {
		t.Run(test.msg, func(t *testing.T) {
			randSrc := rand.NewSource(test.seed)
			assert.Equal(t, test.want, shuffle(randSrc, test.in))
		})
	}
}

// most recently added peer list implementation for the test.
type mraList struct {
	mra peer.StatusPeer
	mrr peer.StatusPeer
}

var _ peer.ListImplementation = (*mraList)(nil)

func (l *mraList) Add(peer peer.StatusPeer) peer.Subscriber {
	l.mra = peer
	return &mraSub{}
}

func (l *mraList) Remove(peer peer.StatusPeer, ps peer.Subscriber) {
	l.mrr = peer
}

func (l *mraList) Choose(ctx context.Context, req *transport.Request) peer.StatusPeer {
	return l.mra
}

func (l *mraList) Start() error {
	return nil
}

func (l *mraList) Stop() error {
	return nil
}

func (l *mraList) IsRunning() bool {
	return true
}

type mraSub struct {
}

func (s *mraSub) NotifyStatusChanged(pid peer.Identifier) {
}

func TestPeerList(t *testing.T) {
	fake := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(peer.Unavailable))
	impl := &mraList{}
	list := New("mra", fake, impl, Capacity(1), NoShuffle(), Seed(0))

	peers := list.Peers()
	assert.Len(t, peers, 0)

	assert.NoError(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			hostport.Identify("1.1.1.1:4040"),
			hostport.Identify("2.2.2.2:4040"),
		},
		Removals: []peer.Identifier{},
	}))

	// Invalid updates before start
	assert.Error(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			hostport.Identify("1.1.1.1:4040"),
		},
		Removals: []peer.Identifier{
			hostport.Identify("3.3.3.3:4040"),
		},
	}))

	assert.Equal(t, 0, list.NumAvailable())
	assert.Equal(t, 0, list.NumUnavailable())
	assert.Equal(t, 2, list.NumUninitialized())
	assert.False(t, list.Available(hostport.Identify("2.2.2.2:4040")))
	assert.True(t, list.Uninitialized(hostport.Identify("2.2.2.2:4040")))

	require.NoError(t, list.Start())

	// Connect to the peer and simulate a request.
	fake.SimulateConnect(hostport.Identify("2.2.2.2:4040"))
	assert.Equal(t, 1, list.NumAvailable())
	assert.Equal(t, 1, list.NumUnavailable())
	assert.Equal(t, 0, list.NumUninitialized())
	assert.True(t, list.Available(hostport.Identify("2.2.2.2:4040")))
	assert.False(t, list.Uninitialized(hostport.Identify("2.2.2.2:4040")))
	peers = list.Peers()
	assert.Len(t, peers, 2)
	p, onFinish, err := list.Choose(context.Background(), &transport.Request{})
	assert.Equal(t, "2.2.2.2:4040", p.Identifier())
	require.NoError(t, err)
	onFinish(nil)

	// Simulate a second connection and request.
	fake.SimulateConnect(hostport.Identify("1.1.1.1:4040"))
	assert.Equal(t, 2, list.NumAvailable())
	assert.Equal(t, 0, list.NumUnavailable())
	assert.Equal(t, 0, list.NumUninitialized())
	peers = list.Peers()
	assert.Len(t, peers, 2)
	p, onFinish, err = list.Choose(context.Background(), &transport.Request{})
	assert.Equal(t, "1.1.1.1:4040", p.Identifier())
	require.NoError(t, err)
	onFinish(nil)

	fake.SimulateDisconnect(hostport.Identify("2.2.2.2:4040"))
	assert.Equal(t, "2.2.2.2:4040", impl.mrr.Identifier())

	assert.NoError(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			hostport.Identify("3.3.3.3:4040"),
		},
		Removals: []peer.Identifier{
			hostport.Identify("2.2.2.2:4040"),
		},
	}))

	// Invalid updates
	assert.Error(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			hostport.Identify("3.3.3.3:4040"),
		},
		Removals: []peer.Identifier{
			hostport.Identify("4.4.4.4:4040"),
		},
	}))

	require.NoError(t, list.Stop())

	// Invalid updates, after stop
	assert.Error(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			hostport.Identify("3.3.3.3:4040"),
		},
		Removals: []peer.Identifier{
			hostport.Identify("4.4.4.4:4040"),
		},
	}))

	assert.NoError(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{},
		Removals: []peer.Identifier{
			hostport.Identify("3.3.3.3:4040"),
		},
	}))
}
