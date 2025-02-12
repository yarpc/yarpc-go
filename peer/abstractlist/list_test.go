// Copyright (c) 2025 Uber Technologies, Inc.
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

package abstractlist

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/x/introspection"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpctest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const (
	id1 = abstractpeer.PeerIdentifier("1.2.3.4:1234")
	id2 = abstractpeer.PeerIdentifier("4.3.2.1:4321")
	id3 = abstractpeer.PeerIdentifier("1.1.1.1:1111")
)

// values returns a slice of the values contained in a map of peers.
func values(m map[string]peer.Identifier) []peer.Identifier {
	vs := make([]peer.Identifier, 0, len(m))
	for _, v := range m {
		vs = append(vs, v)
	}
	return vs
}
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

var _ Implementation = (*mraList)(nil)

func (l *mraList) Add(peer peer.StatusPeer, pid peer.Identifier) Subscriber {
	l.mra = peer
	return &mraSub{}
}

func (l *mraList) Remove(peer peer.StatusPeer, pid peer.Identifier, ps Subscriber) {
	l.mra = nil
	l.mrr = peer
}

func (l *mraList) Choose(req *transport.Request) peer.StatusPeer {
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

type mraSub struct{}

func (s *mraSub) UpdatePendingRequestCount(int) {}

func TestPeerList(t *testing.T) {
	fake := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(peer.Unavailable))
	impl := &mraList{}
	core, log := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	list := New("mra", fake, impl, Capacity(1), NoShuffle(), Seed(0), Logger(logger))

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	peers := list.Peers()
	assert.Len(t, peers, 0)

	assert.NoError(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			abstractpeer.Identify("1.1.1.1:4040"),
			abstractpeer.Identify("2.2.2.2:4040"),
		},
		Removals: []peer.Identifier{},
	}))

	{
		entries := log.FilterMessage("peer list update").AllUntimed()
		require.Len(t, entries, 1)
		assert.Equal(t, map[string]interface{}{
			"additions": int64(2),
			"removals":  int64(0),
		}, entries[0].ContextMap())
	}

	// Invalid updates before start
	assert.Error(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			abstractpeer.Identify("1.1.1.1:4040"),
		},
		Removals: []peer.Identifier{
			abstractpeer.Identify("3.3.3.3:4040"),
		},
	}))

	assert.Equal(t, 0, list.NumAvailable())
	assert.Equal(t, 0, list.NumUnavailable())
	assert.Equal(t, 2, list.NumUninitialized())
	assert.False(t, list.Available(abstractpeer.Identify("2.2.2.2:4040")))
	assert.True(t, list.Uninitialized(abstractpeer.Identify("2.2.2.2:4040")))

	require.NoError(t, list.Start())

	// Connect to the peer and simulate a request.
	fake.SimulateConnect(abstractpeer.Identify("2.2.2.2:4040"))
	assert.Equal(t, 1, list.NumAvailable())
	assert.Equal(t, 1, list.NumUnavailable())
	assert.Equal(t, 0, list.NumUninitialized())
	assert.True(t, list.Available(abstractpeer.Identify("2.2.2.2:4040")))
	assert.False(t, list.Uninitialized(abstractpeer.Identify("2.2.2.2:4040")))
	peers = list.Peers()
	assert.Len(t, peers, 2)
	p, onFinish, err := list.Choose(ctx, &transport.Request{})
	require.NoError(t, err)
	assert.Equal(t, "2.2.2.2:4040", p.Identifier())
	require.NoError(t, err)
	onFinish(nil)

	// Simulate a second connection and request.
	fake.SimulateConnect(abstractpeer.Identify("1.1.1.1:4040"))
	assert.Equal(t, 2, list.NumAvailable())
	assert.Equal(t, 0, list.NumUnavailable())
	assert.Equal(t, 0, list.NumUninitialized())
	peers = list.Peers()
	assert.Len(t, peers, 2)
	p, onFinish, err = list.Choose(ctx, &transport.Request{})
	assert.Equal(t, "1.1.1.1:4040", p.Identifier())
	require.NoError(t, err)
	onFinish(nil)

	fake.SimulateDisconnect(abstractpeer.Identify("2.2.2.2:4040"))
	assert.Equal(t, "2.2.2.2:4040", impl.mrr.Identifier())

	assert.NoError(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			abstractpeer.Identify("3.3.3.3:4040"),
		},
		Removals: []peer.Identifier{
			abstractpeer.Identify("2.2.2.2:4040"),
		},
	}))

	// Invalid updates
	assert.Error(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			abstractpeer.Identify("3.3.3.3:4040"),
		},
		Removals: []peer.Identifier{
			abstractpeer.Identify("4.4.4.4:4040"),
		},
	}))

	require.NoError(t, list.Stop())

	// Invalid updates, after stop
	assert.Error(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			abstractpeer.Identify("3.3.3.3:4040"),
		},
		Removals: []peer.Identifier{
			abstractpeer.Identify("4.4.4.4:4040"),
		},
	}))

	assert.NoError(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{},
		Removals: []peer.Identifier{
			abstractpeer.Identify("3.3.3.3:4040"),
		},
	}))
}

func TestFailWait(t *testing.T) {
	fake := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(peer.Available))
	impl := &mraList{}
	list := New("mra", fake, impl)

	require.NoError(t, list.Start())

	// This case induces the list to enter the wait loop until a peer is added.

	go func() {
		time.Sleep(10 * testtime.Millisecond)
		if err := list.Update(peer.ListUpdates{
			Additions: []peer.Identifier{
				abstractpeer.Identify("0"),
			},
		}); err != nil {
			t.Log(err.Error())
			t.Fail()
		}
	}()

	{
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		p, onFinish, err := list.Choose(ctx, &transport.Request{})
		require.NoError(t, err)
		onFinish(nil)

		assert.Equal(t, "0", p.Identifier())
	}

	// The following case induces the Choose method to enter the wait loop and
	// exit with a timeout error.

	fake.SimulateDisconnect(abstractpeer.Identify("0"))

	{
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		_, _, err := list.Choose(ctx, &transport.Request{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has 1 peer but it is not responsive")
	}
}

func TestFailFast(t *testing.T) {
	fake := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(peer.Unavailable))
	impl := &mraList{}
	list := New("mra", fake, impl, FailFast())

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	require.NoError(t, list.Start())

	_, _, err := list.Choose(ctx, &transport.Request{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no peers")
}

func TestIntrospect(t *testing.T) {
	fake := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(peer.Unavailable))
	impl := &mraList{}
	list := New("mra", fake, impl, FailFast())

	assert.Equal(t, introspection.ChooserStatus{
		Name:  "mra",
		State: "Idle (0/0 available)",
		Peers: []introspection.PeerStatus{},
	}, list.Introspect())

	require.NoError(t, list.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			abstractpeer.Identify("0"),
		},
	}))
	require.NoError(t, list.Start())

	assert.Equal(t, introspection.ChooserStatus{
		Name:  "mra",
		State: "Running (0/1 available)",
		Peers: []introspection.PeerStatus{
			{
				Identifier: "0",
				State:      "Unavailable, 0 pending request(s)",
			},
		},
	}, list.Introspect())

	fake.SimulateConnect(abstractpeer.Identify("0"))

	assert.Equal(t, introspection.ChooserStatus{
		Name:  "mra",
		State: "Running (1/1 available)",
		Peers: []introspection.PeerStatus{
			{
				Identifier: "0",
				State:      "Available, 0 pending request(s)",
			},
		},
	}, list.Introspect())

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Millisecond)
	defer cancel()

	peer, _, err := list.Choose(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, "0", peer.Identifier())

	assert.Equal(t, introspection.ChooserStatus{
		Name:  "mra",
		State: "Running (1/1 available)",
		Peers: []introspection.PeerStatus{
			{
				Identifier: "0",
				State:      "Available, 1 pending request(s)",
			},
		},
	}, list.Introspect())
}

func TestWaitForNeverStarted(t *testing.T) {
	fake := yarpctest.NewFakeTransport(yarpctest.InitialConnectionStatus(peer.Unavailable))
	impl := &mraList{}
	list := New("mra", fake, impl, FailFast())

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	_, _, err := list.Choose(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context finished while waiting for instance to start: context deadline exceeded")
}

func TestDefaultChooseTimeout(t *testing.T) {
	fakeTransport := yarpctest.NewFakeTransport()
	listImplementation := &mraList{}
	req := &transport.Request{}

	list := New("foo-list", fakeTransport, listImplementation, DefaultChooseTimeout(0))
	require.NoError(t, list.Start(), "peer list failed to start")

	err := list.Update(peer.ListUpdates{Additions: []peer.Identifier{
		hostport.PeerIdentifier("foo:peer"),
	}})
	require.NoError(t, err, "could not add fake peer to list")

	// no deadline
	ctx := context.Background()

	_, _, err = list.Choose(ctx, req)
	assert.NoError(t, err, "expected to choose peer without context deadline")
}
