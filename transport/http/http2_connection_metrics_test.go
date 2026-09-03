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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const testHTTP2MetricsServiceName = "test-svc"

func testHTTP2ConnectionMetricsParams(scope *metrics.Scope) http2ConnectionMetricsParams {
	return http2ConnectionMetricsParams{
		Meter:       scope,
		Logger:      zap.NewNop(),
		ServiceName: testHTTP2MetricsServiceName,
	}
}

func TestNewHTTP2ConnectionMetrics_NilScope(t *testing.T) {
	m := newHTTP2ConnectionMetrics(testHTTP2ConnectionMetricsParams(nil))
	require.NotNil(t, m)
	assert.Nil(t, m.activePeers)
	assert.Nil(t, m.connectionsDialedTotal)
	assert.Nil(t, m.connectionDialFailuresTotal)

	assert.NotPanics(t, func() {
		m.incActivePeers()
		m.decActivePeers()
		m.incConnectionsDialed()
		m.incConnectionDialFailures()
	})
}

func TestNewHTTP2ConnectionMetrics_ValidScope(t *testing.T) {
	root := metrics.New()
	m := newHTTP2ConnectionMetrics(testHTTP2ConnectionMetricsParams(root.Scope()))
	require.NotNil(t, m)
	assert.NotNil(t, m.activePeers)
	assert.NotNil(t, m.connectionsDialedTotal)
	assert.NotNil(t, m.connectionDialFailuresTotal)
}

func TestHTTP2ConnectionMetrics_CounterAndGaugeValues(t *testing.T) {
	root := metrics.New()
	m := newHTTP2ConnectionMetrics(testHTTP2ConnectionMetricsParams(root.Scope()))

	m.incActivePeers()
	m.incActivePeers()
	m.incActivePeers()
	m.decActivePeers()
	m.incConnectionsDialed()
	m.incConnectionsDialed()
	m.incConnectionDialFailures()

	snap := root.Snapshot()
	countersByName := make(map[string]int64)
	for _, c := range snap.Counters {
		countersByName[c.Name] = c.Value
	}
	gaugesByName := make(map[string]int64)
	for _, g := range snap.Gauges {
		gaugesByName[g.Name] = g.Value
	}

	assert.Equal(t, int64(2), gaugesByName["http2_peer_dedicated_transports"])
	assert.Equal(t, int64(2), countersByName["http2_connections_dialed_total"])
	assert.Equal(t, int64(1), countersByName["http2_connection_dial_failures_total"])
}

// TestHTTP2ConnectionMetrics_NilReceiver verifies every method is a safe
// no-op on a nil *http2ConnectionMetrics, mirroring the nil-safety of
// go.uber.org/net/metrics' own Counter/Gauge types.
func TestHTTP2ConnectionMetrics_NilReceiver(t *testing.T) {
	var m *http2ConnectionMetrics
	assert.NotPanics(t, func() {
		m.incActivePeers()
		m.decActivePeers()
		m.incConnectionsDialed()
		m.incConnectionDialFailures()
	})
}

// TestManyPeersShareHTTP2ConnectionMetrics is a regression test for the
// failure mode transport/grpc's connection pool metrics hit when they were
// briefly tagged by peer address: registering the same metric name and tags
// a second time (e.g. once per duplicate peer, or once per peer recreated
// after being released and re-retained) errors. Because http2ConnectionMetrics
// is created exactly once per Transport (see (o *transportOptions).newTransport)
// and never re-registered per peer, none of that churn should ever log a
// registration error.
func TestManyPeersShareHTTP2ConnectionMetrics(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	root := metrics.New()

	tr := NewTransport(
		Meter(root.Scope()),
		Logger(zap.New(core)),
	)
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	// Simulate duplicate-peer identifiers resolving to the same address, and
	// peer churn (retain, release, retain again) for a single identifier -
	// all of which reuse the one shared http2ConnectionMetrics instance.
	ids := []peer.Identifier{
		testIdentifier{"127.0.0.1:4321#1"},
		testIdentifier{"127.0.0.1:4321#2"},
		testIdentifier{"127.0.0.1:4321#3"},
	}
	sub := idSubscriber{1}
	for _, id := range ids {
		_, err := tr.RetainPeer(id, sub)
		require.NoError(t, err)
	}
	require.NoError(t, tr.ReleasePeer(ids[0], sub))
	_, err := tr.RetainPeer(ids[0], sub)
	require.NoError(t, err)

	for _, entry := range logs.All() {
		assert.NotContains(t, entry.Message, "failed to create",
			"unexpected metric registration failure: %s", entry.Message)
	}

	snap := root.Snapshot()
	for _, g := range snap.Gauges {
		if g.Name == "http2_peer_dedicated_transports" {
			assert.Equal(t, int64(3), g.Value)
		}
	}
}
