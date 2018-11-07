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

package yarpcpeerlist

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctest"
)

const (
	id1 = yarpc.Address("1.2.3.4:1234")
	id2 = yarpc.Address("4.3.2.1:4321")
	id3 = yarpc.Address("1.1.1.1:1111")
)

func TestValues(t *testing.T) {
	vs := values(map[string]yarpc.Identifier{})
	assert.Equal(t, []yarpc.Identifier{}, vs)

	vs = values(map[string]yarpc.Identifier{"_": id1, "__": id2})
	assert.Equal(t, 2, len(vs))
	assert.Contains(t, vs, id1)
	assert.Contains(t, vs, id2)
}

func TestShuffle(t *testing.T) {
	for _, test := range []struct {
		msg  string
		seed int64
		in   []yarpc.Identifier
		want []yarpc.Identifier
	}{
		{
			"empty",
			0,
			[]yarpc.Identifier{},
			[]yarpc.Identifier{},
		},
		{
			"some",
			0,
			[]yarpc.Identifier{id1, id2, id3},
			[]yarpc.Identifier{id2, id3, id1},
		},
		{
			"different seed",
			7,
			[]yarpc.Identifier{id1, id2, id3},
			[]yarpc.Identifier{id2, id1, id3},
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
	mra yarpc.StatusPeer
	mrr yarpc.StatusPeer
}

var _ Implementation = (*mraList)(nil)

func (l *mraList) Add(peer yarpc.StatusPeer, pid yarpc.Identifier) yarpc.Subscriber {
	l.mra = peer
	return &mraSub{}
}

func (l *mraList) Remove(peer yarpc.StatusPeer, pid yarpc.Identifier, ps yarpc.Subscriber) {
	l.mrr = peer
}

func (l *mraList) Choose(ctx context.Context, req *yarpc.Request) yarpc.StatusPeer {
	return l.mra
}

type mraSub struct {
}

func (s *mraSub) NotifyStatusChanged(pid yarpc.Identifier) {
}

func TestPeerList(t *testing.T) {
	fake := yarpctest.NewFakeTransport("fake", yarpctest.InitialConnectionStatus(yarpc.Unavailable))
	impl := &mraList{}
	list := New("mra", fake, impl, Capacity(1), NoShuffle(), Seed(0))

	peers := list.Peers()
	assert.Len(t, peers, 0)

	assert.NoError(t, list.Update(yarpc.ListUpdates{
		Additions: []yarpc.Identifier{
			yarpc.Address("1.1.1.1:4040"),
			yarpc.Address("2.2.2.2:4040"),
		},
		Removals: []yarpc.Identifier{},
	}))

	// Invalid updates before start
	assert.Error(t, list.Update(yarpc.ListUpdates{
		Additions: []yarpc.Identifier{
			yarpc.Address("1.1.1.1:4040"),
		},
		Removals: []yarpc.Identifier{
			yarpc.Address("3.3.3.3:4040"),
		},
	}))

	// Connect to the peer and simulate a request.
	fake.SimulateConnect(yarpc.Address("2.2.2.2:4040"))
	assert.Equal(t, 1, list.NumAvailable())
	assert.Equal(t, 1, list.NumUnavailable())
	assert.True(t, list.Available(yarpc.Address("2.2.2.2:4040")))
	peers = list.Peers()
	assert.Len(t, peers, 2)
	p, onFinish, err := list.Choose(context.Background(), &yarpc.Request{})
	assert.Equal(t, "2.2.2.2:4040", p.Identifier())
	require.NoError(t, err)
	onFinish(nil)

	// Simulate a second connection and request.
	fake.SimulateConnect(yarpc.Address("1.1.1.1:4040"))
	assert.Equal(t, 2, list.NumAvailable())
	assert.Equal(t, 0, list.NumUnavailable())
	peers = list.Peers()
	assert.Len(t, peers, 2)
	p, onFinish, err = list.Choose(context.Background(), &yarpc.Request{})
	assert.Equal(t, "1.1.1.1:4040", p.Identifier())
	require.NoError(t, err)
	onFinish(nil)

	fake.SimulateDisconnect(yarpc.Address("2.2.2.2:4040"))
	assert.Equal(t, "2.2.2.2:4040", impl.mrr.Identifier())

	assert.NoError(t, list.Update(yarpc.ListUpdates{
		Additions: []yarpc.Identifier{
			yarpc.Address("3.3.3.3:4040"),
		},
		Removals: []yarpc.Identifier{
			yarpc.Address("2.2.2.2:4040"),
		},
	}))

	// Invalid updates
	assert.Error(t, list.Update(yarpc.ListUpdates{
		Additions: []yarpc.Identifier{
			yarpc.Address("3.3.3.3:4040"),
		},
		Removals: []yarpc.Identifier{
			yarpc.Address("4.4.4.4:4040"),
		},
	}))

	assert.NoError(t, list.Update(yarpc.ListUpdates{
		Additions: []yarpc.Identifier{},
		Removals: []yarpc.Identifier{
			yarpc.Address("3.3.3.3:4040"),
		},
	}))
}
