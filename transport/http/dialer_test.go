// Copyright (c) 2026 Uber Technologies, Inc.
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

package http

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/internal/peeraddr"
	"go.uber.org/yarpc/yarpcconfig"
)

// idSubscriber is a comparable subscriber with a distinct identity, used to
// model distinct outbounds in tests.
type idSubscriber struct{ id int }

func (idSubscriber) NotifyStatusChanged(peer.Identifier) {}

// TestDialersSharePeerByDefault verifies that ordinary dialers retain the
// existing address-based peer sharing behavior.
func TestDialersSharePeerByDefault(t *testing.T) {
	transport := NewTransport()
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{"127.0.0.1:4321"}
	dialer1 := transport.NewDialer()
	dialer2 := transport.NewDialer()
	sub1, sub2 := idSubscriber{1}, idSubscriber{2}

	p1, err := dialer1.RetainPeer(id, sub1)
	require.NoError(t, err)
	p2, err := dialer2.RetainPeer(id, sub2)
	require.NoError(t, err)

	assert.Same(t, p1, p2)
	assert.Len(t, transport.peers, 1)
}

// TestIsolatedDialersDoNotSharePeer verifies that isolated dialers get
// distinct peers, and therefore distinct dedicated HTTP/2 connection pools,
// while subscribers using the same dialer continue to share a peer.
func TestIsolatedDialersDoNotSharePeer(t *testing.T) {
	transport := NewTransport()
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{"127.0.0.1:4321"}
	baseDialer := transport.NewDialer()
	dialer1 := baseDialer.WithConnectionIsolation()
	dialer2 := baseDialer.WithConnectionIsolation()
	sub1, sub2, sub3 := idSubscriber{1}, idSubscriber{2}, idSubscriber{3}

	assert.NotEqual(t, dialer1.dialerID, dialer2.dialerID, "each isolated dialer needs its own debug ID")
	assert.NotEqual(t, baseDialer.dialerID, dialer1.dialerID, "an isolated dialer must not reuse its parent's debug ID")

	p1, err := dialer1.RetainPeer(id, sub1)
	require.NoError(t, err)
	p1Again, err := dialer1.RetainPeer(id, sub2)
	require.NoError(t, err)

	// Request-scoped subscribers within one outbound share its peer.
	assert.Same(t, p1, p1Again)
	assert.Len(t, transport.peers, 1)

	p2, err := dialer2.RetainPeer(id, sub3)
	require.NoError(t, err)

	// Two isolated dialers to the same address get separate peers, and
	// therefore separate dedicated HTTP/2 connection pools (see peer.go).
	assert.NotSame(t, p1, p2)
	assert.Len(t, transport.peers, 2)
	_, ok := p1.(*httpPeer)
	require.True(t, ok)
	_, ok = p2.(*httpPeer)
	require.True(t, ok)

	// Releasing one subscriber leaves the other subscriber and dialer's peer
	// intact.
	require.NoError(t, dialer1.ReleasePeer(id, sub1))
	assert.Len(t, transport.peers, 2)

	require.NoError(t, dialer1.ReleasePeer(id, sub2))
	assert.Len(t, transport.peers, 1)
}

// TestIsolatedDialerWithIndexedIdentifiers verifies that connection isolation
// composes with indexed (duplicate-address) identifiers: the peer map is keyed
// on the full identifier plus the dialer's scope, so every occurrence of an
// address gets its own peer per dialer, while each peer still dials the bare
// network address.
func TestIsolatedDialerWithIndexedIdentifiers(t *testing.T) {
	transport := NewTransport()
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	const addr = "127.0.0.1:4321"
	id1 := testIdentifier{peeraddr.Indexed(addr, 1)}
	id2 := testIdentifier{peeraddr.Indexed(addr, 2)}
	sub, sub2 := idSubscriber{1}, idSubscriber{2}

	isolated := transport.NewDialer().WithConnectionIsolation()
	otherIsolated := transport.NewDialer().WithConnectionIsolation()
	shared := transport.NewDialer()

	retain := func(d *Dialer, id peer.Identifier, sub idSubscriber) *httpPeer {
		p, err := d.RetainPeer(id, sub)
		require.NoError(t, err)
		hp, ok := p.(*httpPeer)
		require.True(t, ok)
		return hp
	}

	p1, p2 := retain(isolated, id1, sub), retain(isolated, id2, sub)
	assert.NotSame(t, p1, p2, "indexed occurrences must stay distinct within one isolated dialer")
	assert.Same(t, p1, retain(isolated, id1, sub2), "the same dialer and identifier share a peer")

	other1 := retain(otherIsolated, id1, sub)
	shared1 := retain(shared, id1, sub)
	assert.NotSame(t, p1, other1, "a different isolated dialer must not share the peer")
	assert.NotSame(t, p1, shared1, "the default dialer must not share an isolated dialer's peer")
	assert.NotSame(t, other1, shared1)
	assert.Len(t, transport.peers, 4)

	for _, p := range []*httpPeer{p1, p2, other1, shared1} {
		assert.Equal(t, addr, p.addr, "every peer must dial the bare address, not the indexed identifier")
	}

	// Releasing resolves the same full-identifier-plus-scope key it was
	// retained under.
	require.NoError(t, isolated.ReleasePeer(id1, sub))
	require.NoError(t, isolated.ReleasePeer(id1, sub2))
	require.NoError(t, isolated.ReleasePeer(id2, sub))
	require.NoError(t, otherIsolated.ReleasePeer(id1, sub))
	require.NoError(t, shared.ReleasePeer(id1, sub))
	assert.Empty(t, transport.peers)
}

// TestIsolatedDialersConcurrent hammers RetainPeer/ReleasePeer from many
// independently isolated dialers. Run with -race to detect data races on the
// shared peers map.
func TestIsolatedDialersConcurrent(t *testing.T) {
	transport := NewTransport()
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{"127.0.0.1:4321"}

	const (
		goroutines = 16
		iterations = 50
	)
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			dialer := transport.NewDialer().WithConnectionIsolation()
			sub := idSubscriber{g}
			for range iterations {
				if _, err := dialer.RetainPeer(id, sub); !assert.NoError(t, err) {
					return
				}
				assert.NoError(t, dialer.ReleasePeer(id, sub))
			}
		}(g)
	}
	wg.Wait()
}

// peerCount starts the transport and the outbounds, and returns how many peers
// the transport ended up retaining.
func peerCount(t *testing.T, tr *Transport, outbounds ...*Outbound) int {
	t.Helper()
	require.NoError(t, tr.Start())
	for _, ob := range outbounds {
		require.NoError(t, ob.Start())
	}
	t.Cleanup(func() {
		for _, ob := range outbounds {
			assert.NoError(t, ob.Stop())
		}
		assert.NoError(t, tr.Stop())
	})
	tr.lock.Lock()
	defer tr.lock.Unlock()
	return len(tr.peers)
}

// TestIsolateConnectionsPerOutbound covers outbounds the transport builds
// itself, where the caller cannot pass a Dialer: NewSingleOutbound and
// outbounds built from configuration, whether from a bare url or a peer.
func TestIsolateConnectionsPerOutbound(t *testing.T) {
	const addr = "127.0.0.1:4321"

	t.Run("NewSingleOutbound shares a peer by default", func(t *testing.T) {
		tr := NewTransport()
		a := tr.NewSingleOutbound("http://" + addr + "/rpc")
		b := tr.NewSingleOutbound("http://" + addr + "/rpc")
		assert.Equal(t, 1, peerCount(t, tr, a, b))
	})

	t.Run("NewSingleOutbound isolates when enabled", func(t *testing.T) {
		tr := NewTransport(IsolateConnectionsPerOutbound(true))
		a := tr.NewSingleOutbound("http://" + addr + "/rpc")
		b := tr.NewSingleOutbound("http://" + addr + "/rpc")
		assert.Equal(t, 2, peerCount(t, tr, a, b), "each outbound has its own peer")
	})

	// build loads config with two outbounds to the same address and returns
	// the unary outbounds.
	build := func(t *testing.T, outbounds map[string]interface{}, opts ...Option) (*Transport, []*Outbound) {
		t.Helper()
		configurator := yarpcconfig.New()
		require.NoError(t, configurator.RegisterTransport(TransportSpec(opts...)))
		cfg, err := configurator.LoadConfig("foo", map[string]interface{}{"outbounds": outbounds})
		require.NoError(t, err)
		var obs []*Outbound
		for _, name := range []string{"a", "b"} {
			obs = append(obs, cfg.Outbounds[name].Unary.(*Outbound))
		}
		require.Same(t, obs[0].transport, obs[1].transport)
		return obs[0].transport, obs
	}

	forms := map[string]map[string]interface{}{
		"url only": {
			"a": map[string]interface{}{"http": map[string]interface{}{"url": "http://" + addr + "/rpc"}},
			"b": map[string]interface{}{"http": map[string]interface{}{"url": "http://" + addr + "/rpc"}},
		},
		"peer": {
			"a": map[string]interface{}{"http": map[string]interface{}{"url": "http://x/rpc", "peer": addr}},
			"b": map[string]interface{}{"http": map[string]interface{}{"url": "http://x/rpc", "peer": addr}},
		},
	}
	for name, outbounds := range forms {
		t.Run("config "+name+" shares a peer by default", func(t *testing.T) {
			tr, obs := build(t, outbounds)
			assert.Equal(t, 1, peerCount(t, tr, obs...))
		})
		t.Run("config "+name+" isolates when enabled", func(t *testing.T) {
			tr, obs := build(t, outbounds, IsolateConnectionsPerOutbound(true))
			assert.Equal(t, 2, peerCount(t, tr, obs...), "each configured outbound has its own peer")
		})
	}

	t.Run("requests within one outbound still share its peer", func(t *testing.T) {
		tr := NewTransport(IsolateConnectionsPerOutbound(true))
		a := tr.NewSingleOutbound("http://" + addr + "/rpc")
		assert.Equal(t, 1, peerCount(t, tr, a))
	})
}
