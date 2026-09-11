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
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/peer"
	. "go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/internal/testtime"
	ypeer "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// NoJitter is a transport option only available in tests, to disable jitter
// between connection attempts.
func NoJitter() TransportOption {
	return func(options *transportOptions) {
		options.jitter = func(n int64) int64 {
			return n
		}
	}
}

type peerExpectation struct {
	id          string
	subscribers []string
}

func createPeerIdentifierMap(ids []string) map[string]peer.Identifier {
	pids := make(map[string]peer.Identifier, len(ids))
	for _, id := range ids {
		pids[id] = &testIdentifier{id}
	}
	return pids
}

func TestTransport(t *testing.T) {
	type testStruct struct {
		msg string

		// identifiers defines all the Identifiers that will be used in
		// the actions up from so they can be generated and passed as deps
		identifiers []string

		// subscriberDefs defines all the Subscribers that will be used in
		// the actions up from so they can be generated and passed as deps
		subscriberDefs []SubscriberDefinition

		// actions are the actions that will be applied against the transport
		actions []TransportAction

		// expectedPeers are a list of peers (and those peer's subscribers)
		// that are expected on the transport after the actions
		expectedPeers []peerExpectation
	}
	tests := []testStruct{
		{
			msg:         "one retain",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1"}},
			},
		},
		{
			msg:         "one retain one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s1"},
			},
		},
		{
			msg:         "three retains",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s2", "s3"}},
			},
		},
		{
			msg:         "three retains one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2r"},
				{ID: "s3"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2r", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s2r"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s3"}},
			},
		},
		{
			msg:         "three retains, three release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s2"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s3"},
			},
		},
		{
			msg:         "no retains one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []TransportAction{
				ReleaseAction{
					InputIdentifierID: "i1",
					InputSubscriberID: "s1",
					ExpectedErrType:   peer.ErrTransportHasNoReferenceToPeer{},
				},
			},
		},
		{
			msg:         "one retains, one release (from different subscriber)",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				ReleaseAction{
					InputIdentifierID: "i1",
					InputSubscriberID: "s2",
					ExpectedErrType:   peer.ErrPeerHasNoReferenceToSubscriber{},
				},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1"}},
			},
		},
		{
			msg:         "multi peer retain/release",
			identifiers: []string{"i1", "i2", "i3", "i4r", "i5r"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
				{ID: "s4"},
				{ID: "s5rnd"},
				{ID: "s6rnd"},
				{ID: "s7rnd"},
			},
			actions: []TransportAction{
				// Retains/Releases of i1 (Retain/Release the random peers at the end)
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s5rnd", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s6rnd", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s5rnd"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s6rnd"},

				// Retains/Releases of i2 (Retain then Release then Retain again)
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s2", ExpectedPeerID: "i2"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s3", ExpectedPeerID: "i2"},
				ReleaseAction{InputIdentifierID: "i2", InputSubscriberID: "s2"},
				ReleaseAction{InputIdentifierID: "i2", InputSubscriberID: "s3"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s2", ExpectedPeerID: "i2"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s3", ExpectedPeerID: "i2"},

				// Retains/Releases of i3 (Retain/Release unrelated sub, then retain two)
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s7rnd", ExpectedPeerID: "i3"},
				ReleaseAction{InputIdentifierID: "i3", InputSubscriberID: "s7rnd"},
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s3", ExpectedPeerID: "i3"},
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s4", ExpectedPeerID: "i3"},

				// Retain/Release i4r on random subscriber
				RetainAction{InputIdentifierID: "i4r", InputSubscriberID: "s5rnd", ExpectedPeerID: "i4r"},
				ReleaseAction{InputIdentifierID: "i4r", InputSubscriberID: "s5rnd"},

				// Retain/Release i5r on already used subscriber
				RetainAction{InputIdentifierID: "i5r", InputSubscriberID: "s3", ExpectedPeerID: "i5r"},
				ReleaseAction{InputIdentifierID: "i5r", InputSubscriberID: "s3"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s2"}},
				{id: "i2", subscribers: []string{"s2", "s3"}},
				{id: "i3", subscribers: []string{"s3", "s4"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			transport := NewTransport()
			defer transport.Stop()

			deps := TransportDeps{
				PeerIdentifiers: createPeerIdentifierMap(tt.identifiers),
				Subscribers:     CreateSubscriberMap(mockCtrl, tt.subscriberDefs),
			}
			ApplyTransportActions(t, transport, tt.actions, deps)

			assert.Len(t, transport.peers, len(tt.expectedPeers))
			for _, expectedPeerNode := range tt.expectedPeers {
				p, ok := transport.peers[peerKey{identifier: expectedPeerNode.id}]
				assert.True(t, ok)

				if assert.NotNil(t, p) {
					assert.Equal(t, expectedPeerNode.id, p.Identifier())

					// We can't look at the hostport subscribers directly so we'll
					// attempt to remove subscribers and be sure that it doesn't error
					assert.Len(t, expectedPeerNode.subscribers, p.NumSubscribers())
					for _, sub := range expectedPeerNode.subscribers {
						err := p.Unsubscribe(deps.Subscribers[sub])
						assert.NoError(t, err, "peer %s did not have reference to subscriber %s", p.Identifier(), sub)
					}
				}
			}
		})
	}
}

func TestDefaultTransportInitialisation(t *testing.T) {
	transport := NewTransport()

	assert.NotNil(t, transport.h1Transport)
	assert.NotNil(t, transport.h2Transport)
}

func TestTransportClientOpaqueOptions(t *testing.T) {
	// Unfortunately the KeepAlive is obfuscated in the client, so we can't really
	// assert this worked.
	transport := NewTransport(
		KeepAlive(testtime.Second),
		MaxIdleConns(100),
		MaxIdleConnsPerHost(10),
		IdleConnTimeout(1*time.Second),
		DisableCompression(),
		DisableKeepAlives(),
		ResponseHeaderTimeout(1*time.Second),
	)

	assert.NotNil(t, transport.h1Transport)
	assert.NotNil(t, transport.h2Transport)
}

// TestGetOrCreatePeerDoesNotDedupeSuffixedIdentifiers simulates what the
// duplicate-peer-identifier support (layered on top of this transport)
// produces: two distinct peer.Identifiers that resolve to the same real
// address. getOrCreatePeer must not collapse them into one peer.
func TestGetOrCreatePeerDoesNotDedupeSuffixedIdentifiers(t *testing.T) {
	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	// getOrCreatePeer requires the write lock.
	tr.lock.Lock()
	p1 := tr.getOrCreatePeer(testIdentifier{"127.0.0.1:1234#1"}, nil)
	p2 := tr.getOrCreatePeer(testIdentifier{"127.0.0.1:1234#2"}, nil)
	tr.lock.Unlock()

	require.NotSame(t, p1, p2, "duplicate identifiers must not collapse into one peer")
}

func TestPeersGetIndependentHTTP2Transports(t *testing.T) {
	// Count new TCP connections accepted by the server so we can prove two
	// httpPeers actually open independent HTTP/2 connections, rather than
	// just asserting their *http2.Transport pointers differ.
	var newConns atomic.Int32
	server := newH2TestServer(func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			newConns.Add(1)
		}
	})
	defer server.Close()

	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	addr := strings.TrimPrefix(server.URL, "http://")

	// Build two httpPeer instances for the same address directly: newPeer
	// itself (unlike getOrCreatePeer's map) never dedupes, so this proves
	// each httpPeer's dedicated connection pool opens its own connection.
	p1 := newPeer(addr, tr)
	p2 := newPeer(addr, tr)
	t.Cleanup(func() {
		if pool := p1.loadH2Pool(); pool != nil {
			pool.Stop()
		}
		if pool := p2.loadH2Pool(); pool != nil {
			pool.Stop()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	for _, p := range []*httpPeer{p1, p2} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		sender, err := p.h2Sender()
		require.NoError(t, err)
		res, err := sender.Do(req)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
	}

	require.NotSame(t, p1.loadH2Pool(), p2.loadH2Pool(),
		"each httpPeer must own an independent HTTP/2 connection pool")

	assert.EqualValues(t, 2, newConns.Load(),
		"each peer's dedicated http2.Transport should open its own connection instead of sharing one")
}

// TestH2PoolMetricsTrackDialAndTeardown is a regression test for the
// leak-detection gap flagged in review on PR #2539 ("we are not emitting
// any new metric with total live connections... worth doing it so we can
// detect possible leaks like the one in the peer teardown"): the shared
// connpool.Metrics active-connection gauge must go up on a real dial and
// back down to 0 once the connection is actually torn down -- even though
// dynamic scaling (the only other path that updates these gauges) is
// disabled -- because AddConn calls RefreshMetrics unconditionally and
// watchH2Conn now does the same after Remove.
func TestH2PoolMetricsTrackDialAndTeardown(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()

	root := metrics.New()
	tr := NewTransport(Meter(root.Scope()), ServiceName("test-svc"))
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	addr := strings.TrimPrefix(server.URL, "http://")
	p := newPeer(addr, tr)

	activeConnGauge := func() int64 {
		snap := root.Snapshot()
		for _, g := range snap.Gauges {
			if g.Name == "conn_pool_active_connections" {
				return g.Value
			}
		}
		return 0
	}

	assert.EqualValues(t, 0, activeConnGauge(), "no connection dialed yet")

	sender, err := p.h2Sender()
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	res, err := sender.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	assert.EqualValues(t, 1, activeConnGauge(), "dialing the peer's first connection should increment the gauge")

	pool := p.loadH2Pool()
	require.NotNil(t, pool)
	pool.Stop()
	pool.Wait()

	assert.Eventually(t, func() bool {
		return activeConnGauge() == 0
	}, testtime.Second, testtime.Millisecond*10, "tearing down the connection should bring the gauge back to 0")
}

// TestTransportStopStopsPeerH2Pools is a regression test for
// stopPeerH2Pools: it must be exercised through Transport.Stop's real
// lifecycle -- iterating a.peers, stopping every peer's pool, and waiting
// for each to tear down -- not just through a peer created and stopped
// directly (as most other tests in this file do, bypassing the Transport's
// peer map entirely).
func TestTransportStopStopsPeerH2Pools(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()

	tr := NewTransport()
	require.NoError(t, tr.Start())

	addr := strings.TrimPrefix(server.URL, "http://")
	p := newPeer(addr, tr)
	_, err := p.h2Sender()
	require.NoError(t, err)

	// Register the peer directly in the Transport's map, the way
	// getOrCreatePeer would, but without spawning MaintainConn: that
	// goroutine only exits via Release or an Unavailable peer's backoff
	// loop noticing Transport.once.Stopping, neither of which this test
	// needs in order to exercise stopPeerH2Pools itself.
	tr.lock.Lock()
	tr.peers[peerKey{identifier: addr}] = p
	tr.lock.Unlock()

	require.NoError(t, tr.Stop())

	pool := p.loadH2Pool()
	require.NotNil(t, pool)
	assert.Nil(t, pool.PickConn(), "Transport.Stop should have torn down the peer's HTTP/2 pool")
}

// TestDialH2ConnDialFailure covers dialH2Conn's first error branch: the raw
// TCP dial failing.
func TestDialH2ConnDialFailure(t *testing.T) {
	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	// Bind and immediately release a port so nothing is listening on it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	_, err = tr.dialH2Conn(ctx, addr)
	require.Error(t, err)
}

// TestDialH2ConnNewClientConnFailure covers dialH2Conn's second error
// branch: DialTLSContext succeeds, but wrapping the connection with
// NewClientConn fails. http2.Transport.NewClientConn writes the client
// preface and initial SETTINGS frame synchronously, so handing it an
// already-closed net.Conn (via a custom DialContext) makes that write fail
// deterministically, without needing a real misbehaving server.
func TestDialH2ConnNewClientConnFailure(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	require.NoError(t, serverConn.Close())
	require.NoError(t, clientConn.Close())

	tr := NewTransport(DialContext(func(ctx context.Context, network, addr string) (net.Conn, error) {
		return clientConn, nil
	}))
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	_, err := tr.dialH2Conn(context.Background(), "unused:0")
	require.Error(t, err)
}

// TestH2SenderReusesExistingConnection drives two requests through the same
// peer and asserts a single dial: the second h2Sender call must return via
// pool.PickConn's fast path (checked both before and after acquiring
// h2DialMu) instead of dialing again.
func TestH2SenderReusesExistingConnection(t *testing.T) {
	var dials atomic.Int32
	server := newH2TestServer(func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			dials.Add(1)
		}
	})
	defer server.Close()

	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	addr := strings.TrimPrefix(server.URL, "http://")
	p := newPeer(addr, tr)
	t.Cleanup(func() {
		if pool := p.loadH2Pool(); pool != nil {
			pool.Stop()
		}
	})

	for range 2 {
		sender, err := p.h2Sender()
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		res, err := sender.Do(req)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
	}

	assert.EqualValues(t, 1, dials.Load(), "a second request on the same peer must reuse the dialed connection")
}

// TestH2SenderConcurrentFirstDial hammers a single cold peer's h2Sender
// concurrently. Only one goroutine should win the race to create the pool
// and dial the first connection; every other goroutine must take one of
// h2Sender's two PickConn fast paths (the unlocked check before acquiring
// h2DialMu, and the re-check immediately after acquiring it) rather than
// attempting its own dial or pool creation.
func TestH2SenderConcurrentFirstDial(t *testing.T) {
	var dials atomic.Int32
	server := newH2TestServer(func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			dials.Add(1)
		}
	})
	defer server.Close()

	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	addr := strings.TrimPrefix(server.URL, "http://")
	p := newPeer(addr, tr)
	t.Cleanup(func() {
		if pool := p.loadH2Pool(); pool != nil {
			pool.Stop()
		}
	})

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = p.h2Sender()
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		assert.NoError(t, err)
	}
	assert.EqualValues(t, 1, dials.Load(), "only one goroutine should dial the peer's first connection")
}

// TestWatchH2ConnEvictsUnhealthyConnection covers watchH2Conn's periodic
// health-poll branch: unlike TestH2PoolMetricsTrackDialAndTeardown (which
// tears the connection down via an explicit pool.Stop, hitting the
// Context().Done() branch), this test lets the connection go unhealthy on
// its own -- by closing the server out from under it -- so the ticker
// branch is what notices and evicts it.
func TestWatchH2ConnEvictsUnhealthyConnection(t *testing.T) {
	origInterval := defaultH2ConnHealthPollInterval
	defaultH2ConnHealthPollInterval = testtime.Millisecond * 10
	t.Cleanup(func() { defaultH2ConnHealthPollInterval = origInterval })

	// h2c.NewHandler hijacks each connection, and httptest.Server.Close does
	// not close hijacked connections, so track them to sever explicitly.
	var (
		connsMu     sync.Mutex
		serverConns []net.Conn
	)
	server := newH2TestServer(func(c net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connsMu.Lock()
			serverConns = append(serverConns, c)
			connsMu.Unlock()
		}
	})

	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	addr := strings.TrimPrefix(server.URL, "http://")
	p := newPeer(addr, tr)
	t.Cleanup(func() {
		if pool := p.loadH2Pool(); pool != nil {
			pool.Stop()
		}
	})

	sender, err := p.h2Sender()
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	res, err := sender.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	pool := p.loadH2Pool()
	require.NotNil(t, pool)

	// Take the server out from under the live connection: the health poller
	// (not an external pool.Stop) is what must notice it can no longer take
	// new requests and evict it.
	server.Close()
	connsMu.Lock()
	for _, c := range serverConns {
		c.Close()
	}
	connsMu.Unlock()

	assert.Eventually(t, func() bool {
		return pool.PickConn() == nil
	}, testtime.Second, testtime.Millisecond*5, "health poll should evict the connection once the server goes away")
}

// TestWatchH2ConnKeepsSaturatedConnection checks that the health poll does
// not evict a healthy connection that has merely reached the server's
// concurrent-stream limit: it cannot take a new request right now
// (CanTakeNewRequest is false), but it is not broken, and evicting it would
// abort the request in flight on it.
func TestWatchH2ConnKeepsSaturatedConnection(t *testing.T) {
	origInterval := defaultH2ConnHealthPollInterval
	defaultH2ConnHealthPollInterval = testtime.Millisecond * 10
	t.Cleanup(func() { defaultH2ConnHealthPollInterval = origInterval })

	slowStarted := make(chan struct{}, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseSlow := func() { releaseOnce.Do(func() { close(release) }) }

	server := httptest.NewUnstartedServer(h2c.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/slow" {
				slowStarted <- struct{}{}
				<-release
			}
			w.WriteHeader(http.StatusOK)
		}),
		&http2.Server{MaxConcurrentStreams: 1, IdleTimeout: defaultIdleConnTimeout},
	))
	server.Start()
	t.Cleanup(func() { releaseSlow(); server.Close() })

	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	addr := strings.TrimPrefix(server.URL, "http://")
	p := newPeer(addr, tr)
	t.Cleanup(func() {
		if pool := p.loadH2Pool(); pool != nil {
			pool.Stop()
			pool.Wait()
		}
	})

	sender, err := p.h2Sender()
	require.NoError(t, err)
	inFlight := doAsync(t, sender, server.URL+"/slow")
	<-slowStarted

	w := p.loadH2Pool().PickConn()
	require.NotNil(t, w)
	require.Eventually(t, func() bool { return !w.Conn.CanTakeNewRequest() },
		testtime.Second, testtime.Millisecond,
		"precondition: the one-stream connection must be saturated")

	// Give the 10ms health poll many chances to (wrongly) evict it.
	time.Sleep(20 * defaultH2ConnHealthPollInterval)
	assert.True(t, w.IsActive(), "a saturated but healthy connection must stay in service")
	assert.NoError(t, w.Context().Err(), "a saturated but healthy connection must not be cancelled")

	releaseSlow()
	res := <-inFlight
	require.NoError(t, res.err, "the in-flight request must complete")
	require.NoError(t, res.resp.Body.Close())
}

// TestH2ActivePeersGauge pins two things about the h2ActivePeers gauge
// (Transport.h2ActivePeers, incremented/decremented from peer.go's h2Sender
// and Release):
//
//  1. It only reflects peers that actually hold a dedicated HTTP/2
//     connection pool, not every httpPeer that exists -- newPeer alone (like
//     RetainPeer alone) must not create a pool or move the gauge.
//  2. It is registered exactly once per Transport and shared across
//     multiple peers for the same address (as duplicate-peer identifiers or
//     isolated Dialers produce), which go.uber.org/net/metrics would
//     otherwise reject as a duplicate registration if it were created once
//     per peer instead.
func TestH2ActivePeersGauge(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()

	root := metrics.New()
	tr := NewTransport(Meter(root.Scope()), ServiceName("test-svc"))
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	activePeersGauge := func() int64 {
		snap := root.Snapshot()
		for _, g := range snap.Gauges {
			if g.Name == "http2_peer_dedicated_transports" {
				return g.Value
			}
		}
		return 0
	}

	addr := strings.TrimPrefix(server.URL, "http://")
	p1 := newPeer(addr, tr)
	p2 := newPeer(addr, tr)

	assert.Zero(t, activePeersGauge(), "a bare httpPeer must not create an HTTP/2 pool")

	_, err := p1.h2Sender()
	require.NoError(t, err)
	assert.EqualValues(t, 1, activePeersGauge(), "dialing a peer's first connection should count it as active")

	_, err = p2.h2Sender()
	require.NoError(t, err)
	assert.EqualValues(t, 2, activePeersGauge(),
		"a second, independent peer for the same address must add its own count, not error on re-registration")

	pool1 := p1.loadH2Pool()
	p1.Release()
	pool1.Wait()
	assert.EqualValues(t, 1, activePeersGauge(), "releasing a peer with a pool should decrement the gauge")

	pool2 := p2.loadH2Pool()
	p2.Release()
	pool2.Wait()
	assert.Zero(t, activePeersGauge())
}

// TestH2ActivePeersGaugeIgnoresFailedDial pins that a peer whose dial fails
// (so its pool exists but holds no connection) is not counted by the
// h2ActivePeers gauge, and that releasing it does not decrement the gauge.
func TestH2ActivePeersGaugeIgnoresFailedDial(t *testing.T) {
	// Reserve an address nothing listens on.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	root := metrics.New()
	tr := NewTransport(Meter(root.Scope()), ServiceName("test-svc"))
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	activePeersGauge := func() int64 {
		for _, g := range root.Snapshot().Gauges {
			if g.Name == "http2_peer_dedicated_transports" {
				return g.Value
			}
		}
		return 0
	}

	p := newPeer(addr, tr)
	_, err = p.h2Sender()
	require.Error(t, err, "dialing a closed port must fail")
	require.NotNil(t, p.loadH2Pool(), "the pool exists even though it has no connection")
	assert.Zero(t, activePeersGauge(), "a failed dial must not count the peer")

	pool := p.loadH2Pool()
	p.Release()
	pool.Wait()
	assert.Zero(t, activePeersGauge(), "releasing a never-connected peer must not decrement the gauge")
}

// TestCountH2ActivePeerAfterRelease covers the race where Release runs between
// a peer's successful dial and its count: the count must be undone so the
// gauge does not leak.
func TestCountH2ActivePeerAfterRelease(t *testing.T) {
	root := metrics.New()
	tr := NewTransport(Meter(root.Scope()), ServiceName("test-svc"))
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	p := newPeer("127.0.0.1:1", tr)
	p.Release() // closes p.released; no pool, so nothing was counted.
	p.countH2ActivePeer()

	assert.False(t, p.h2Counted.Load(), "a released peer must not stay counted")
	for _, g := range root.Snapshot().Gauges {
		if g.Name == "http2_peer_dedicated_transports" {
			assert.Zero(t, g.Value)
		}
	}
}

// TestNewH2ActivePeersGaugeRegistrationFailure covers newH2ActivePeersGauge's
// error branch: go.uber.org/net/metrics rejects registering a second gauge
// with the same name and constant tag values, so pre-registering
// "http2_peer_dedicated_transports" forces NewTransport's call to fail. A
// nil gauge from that failure must not make h2Sender/Release panic (Gauge's
// Inc/Dec are nil-safe), just silently no-op the metric.
func TestNewH2ActivePeersGaugeRegistrationFailure(t *testing.T) {
	root := metrics.New()
	scope := root.Scope()

	_, err := scope.Gauge(metrics.Spec{
		Name: "http2_peer_dedicated_transports",
		Help: "pre-registered to force a collision",
		ConstTags: metrics.Tags{
			"component": "yarpc",
			"service":   "test-svc",
			"transport": "http",
		},
	})
	require.NoError(t, err)

	tr := NewTransport(Meter(scope), ServiceName("test-svc"))
	require.Nil(t, tr.h2ActivePeers, "a registration failure should leave the gauge nil")

	assert.NotPanics(t, func() {
		tr.h2ActivePeers.Inc()
		tr.h2ActivePeers.Dec()
	})
}

func TestDialContext(t *testing.T) {
	errMsg := "my custom dialer error message"
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, errors.New(errMsg)
	}

	transport := NewTransport(DialContext(dialContext))

	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	req, err := http.NewRequest("GET", "http://foo.bar", nil)
	require.NoError(t, err)

	outbound := transport.NewOutbound(ypeer.NewSingle(hostport.Identify("foo"), transport))
	require.NoError(t, outbound.Start())
	defer func() { assert.NoError(t, outbound.Stop()) }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = outbound.RoundTrip(req.WithContext(ctx))
	require.Error(t, err)
	assert.Contains(t, err.Error(), errMsg)
}

type testIdentifier struct {
	id string
}

func (i testIdentifier) Identifier() string {
	return i.id
}

func TestRetainDuplicatePeers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	trans := NewTransport()
	require.NoError(t, trans.Start())
	defer func() { assert.NoError(t, trans.Stop()) }()

	const address = "127.0.0.1:8080"
	// Each peer's connection loop notifies its subscribers of status changes
	// from its own goroutine. An unexpected gomock call would Fatalf there and
	// exit that goroutine before it signals the transport's WaitGroup, hanging
	// Stop, so accept any number of notifications.
	sub := NewMockSubscriber(mockCtrl)
	sub.EXPECT().NotifyStatusChanged(gomock.Any()).AnyTimes()

	// The same address listed twice must yield two peers, each maintaining its
	// own connection, but both dialing the real address.
	first, err := trans.RetainPeer(hostport.PeerIdentifier(address+"#1"), sub)
	require.NoError(t, err)
	second, err := trans.RetainPeer(hostport.PeerIdentifier(address+"#2"), sub)
	require.NoError(t, err)

	assert.NotSame(t, first, second, "duplicate peers must not be shared")
	assert.Len(t, trans.peers, 2)

	firstHTTP, ok := first.(*httpPeer)
	require.True(t, ok)
	secondHTTP, ok := second.(*httpPeer)
	require.True(t, ok)
	assert.Equal(t, address, firstHTTP.addr, "peer must dial the bare address")
	assert.Equal(t, address, secondHTTP.addr, "peer must dial the bare address")

	// Retaining the same identifier again reuses the existing peer.
	firstAgain, err := trans.RetainPeer(hostport.PeerIdentifier(address+"#1"), sub)
	require.NoError(t, err)
	assert.Same(t, first, firstAgain)
	assert.Len(t, trans.peers, 2)

	// Releasing one occurrence leaves the other alone.
	require.NoError(t, trans.ReleasePeer(hostport.PeerIdentifier(address+"#1"), sub))
	assert.Len(t, trans.peers, 1)
	require.NoError(t, trans.ReleasePeer(hostport.PeerIdentifier(address+"#2"), sub))
	assert.Empty(t, trans.peers)
}
