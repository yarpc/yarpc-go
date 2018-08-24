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
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/yarpc/v2/yarpctest"
	"go.uber.org/yarpc/v2/yarpctransport"
)

const (
	id1 = yarpcpeer.Address("1.2.3.4:1234")
	id2 = yarpcpeer.Address("4.3.2.1:4321")
	id3 = yarpcpeer.Address("1.1.1.1:1111")
)

func TestValues(t *testing.T) {
	vs := values(map[string]yarpcpeer.Identifier{})
	assert.Equal(t, []yarpcpeer.Identifier{}, vs)

	vs = values(map[string]yarpcpeer.Identifier{"_": id1, "__": id2})
	assert.Equal(t, 2, len(vs))
	assert.Contains(t, vs, id1)
	assert.Contains(t, vs, id2)
}

func TestShuffle(t *testing.T) {
	for _, test := range []struct {
		msg  string
		seed int64
		in   []yarpcpeer.Identifier
		want []yarpcpeer.Identifier
	}{
		{
			"empty",
			0,
			[]yarpcpeer.Identifier{},
			[]yarpcpeer.Identifier{},
		},
		{
			"some",
			0,
			[]yarpcpeer.Identifier{id1, id2, id3},
			[]yarpcpeer.Identifier{id2, id3, id1},
		},
		{
			"different seed",
			7,
			[]yarpcpeer.Identifier{id1, id2, id3},
			[]yarpcpeer.Identifier{id2, id1, id3},
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
	mra yarpcpeer.StatusPeer
	mrr yarpcpeer.StatusPeer
}

var _ Implementation = (*mraList)(nil)

func (l *mraList) Add(peer yarpcpeer.StatusPeer, pid yarpcpeer.Identifier) yarpcpeer.Subscriber {
	l.mra = peer
	return &mraSub{}
}

func (l *mraList) Remove(peer yarpcpeer.StatusPeer, pid yarpcpeer.Identifier, ps yarpcpeer.Subscriber) {
	l.mrr = peer
}

func (l *mraList) Choose(ctx context.Context, req *yarpctransport.Request) yarpcpeer.StatusPeer {
	return l.mra
}

type mraSub struct {
}

func (s *mraSub) NotifyStatusChanged(pid yarpcpeer.Identifier) {
}

func TestPeerList(t *testing.T) {
	fake := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(yarpcpeer.Unavailable))
	impl := &mraList{}
	list := New("mra", fake, impl, Capacity(1), NoShuffle(), Seed(0))

	peers := list.Peers()
	assert.Len(t, peers, 0)

	assert.NoError(t, list.Update(yarpcpeer.ListUpdates{
		Additions: []yarpcpeer.Identifier{
			yarpcpeer.Address("1.1.1.1:4040"),
			yarpcpeer.Address("2.2.2.2:4040"),
		},
		Removals: []yarpcpeer.Identifier{},
	}))

	// Invalid updates before start
	assert.Error(t, list.Update(yarpcpeer.ListUpdates{
		Additions: []yarpcpeer.Identifier{
			yarpcpeer.Address("1.1.1.1:4040"),
		},
		Removals: []yarpcpeer.Identifier{
			yarpcpeer.Address("3.3.3.3:4040"),
		},
	}))

	// Connect to the peer and simulate a request.
	fake.SimulateConnect(yarpcpeer.Address("2.2.2.2:4040"))
	assert.Equal(t, 1, list.NumAvailable())
	assert.Equal(t, 1, list.NumUnavailable())
	assert.True(t, list.Available(yarpcpeer.Address("2.2.2.2:4040")))
	peers = list.Peers()
	assert.Len(t, peers, 2)
	p, onFinish, err := list.Choose(context.Background(), &yarpctransport.Request{})
	assert.Equal(t, "2.2.2.2:4040", p.Identifier())
	require.NoError(t, err)
	onFinish(nil)

	// Simulate a second connection and request.
	fake.SimulateConnect(yarpcpeer.Address("1.1.1.1:4040"))
	assert.Equal(t, 2, list.NumAvailable())
	assert.Equal(t, 0, list.NumUnavailable())
	peers = list.Peers()
	assert.Len(t, peers, 2)
	p, onFinish, err = list.Choose(context.Background(), &yarpctransport.Request{})
	assert.Equal(t, "1.1.1.1:4040", p.Identifier())
	require.NoError(t, err)
	onFinish(nil)

	fake.SimulateDisconnect(yarpcpeer.Address("2.2.2.2:4040"))
	assert.Equal(t, "2.2.2.2:4040", impl.mrr.Identifier())

	assert.NoError(t, list.Update(yarpcpeer.ListUpdates{
		Additions: []yarpcpeer.Identifier{
			yarpcpeer.Address("3.3.3.3:4040"),
		},
		Removals: []yarpcpeer.Identifier{
			yarpcpeer.Address("2.2.2.2:4040"),
		},
	}))

	// Invalid updates
	assert.Error(t, list.Update(yarpcpeer.ListUpdates{
		Additions: []yarpcpeer.Identifier{
			yarpcpeer.Address("3.3.3.3:4040"),
		},
		Removals: []yarpcpeer.Identifier{
			yarpcpeer.Address("4.4.4.4:4040"),
		},
	}))

	assert.NoError(t, list.Update(yarpcpeer.ListUpdates{
		Additions: []yarpcpeer.Identifier{},
		Removals: []yarpcpeer.Identifier{
			yarpcpeer.Address("3.3.3.3:4040"),
		},
	}))
}
