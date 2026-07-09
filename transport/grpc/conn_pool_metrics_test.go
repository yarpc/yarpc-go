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

package grpc

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const testConnPoolServiceName = "test-svc"

func testMetricsParams(scope *metrics.Scope) connPoolMetricsParams {
	return connPoolMetricsParams{
		Meter:       scope,
		Logger:      zap.NewNop(),
		ServiceName: testConnPoolServiceName,
	}
}

func wantConnPoolMetricTags(serviceName string) map[string]string {
	return map[string]string{
		_componentTag: _componentTagValueYarpc,
		_serviceTag:   serviceName,
		_transportTag: _transportTagValueGrpc,
	}
}

func assertConnPoolMetricTags(t *testing.T, snap *metrics.RootSnapshot, serviceName string) {
	t.Helper()
	want := wantConnPoolMetricTags(serviceName)
	for _, g := range snap.Gauges {
		if !strings.HasPrefix(g.Name, "conn_pool_") {
			continue
		}
		for k, v := range want {
			assert.Equal(t, v, g.Tags[k], "gauge %s: tag %q", g.Name, k)
		}
	}
	for _, c := range snap.Counters {
		if !strings.HasPrefix(c.Name, "conn_pool_") {
			continue
		}
		for k, v := range want {
			assert.Equal(t, v, c.Tags[k], "counter %s: tag %q", c.Name, k)
		}
	}
}

func TestNewConnPoolMetrics_NilScope(t *testing.T) {
	m := newConnPoolMetrics(testMetricsParams(nil))
	require.NotNil(t, m)
	assert.Nil(t, m.connectionCount)
	assert.Nil(t, m.drainingConnectionCount)
	assert.Nil(t, m.idleConnectionCount)
	assert.Nil(t, m.scaleUpTotal)
	assert.Nil(t, m.scaleDownTotal)
	assert.Nil(t, m.idleReactivationTotal)

	assert.NotPanics(t, func() {
		m.incScaleUp()
		m.incScaleDown()
		m.incIdleReactivation()
		m.addConnectionCount(5)
		m.addDrainingConnectionCount(3)
		m.addIdleConnectionCount(1)
		newPeerPoolReporter(m).setCounts(5, 3, 1)
	})
}

func TestNewConnPoolMetrics_ValidScope(t *testing.T) {
	root := metrics.New()
	m := newConnPoolMetrics(testMetricsParams(root.Scope()))
	require.NotNil(t, m)
	assert.NotNil(t, m.connectionCount)
	assert.NotNil(t, m.drainingConnectionCount)
	assert.NotNil(t, m.idleConnectionCount)
	assert.NotNil(t, m.scaleUpTotal)
	assert.NotNil(t, m.scaleDownTotal)
	assert.NotNil(t, m.idleReactivationTotal)
}

func TestConnPoolMetrics_CounterIncrements(t *testing.T) {
	root := metrics.New()
	m := newConnPoolMetrics(testMetricsParams(root.Scope()))

	m.incScaleUp()
	m.incScaleUp()
	m.incScaleDown()
	m.incIdleReactivation()
	m.incIdleReactivation()
	m.incIdleReactivation()

	snap := root.Snapshot()
	countersByName := make(map[string]int64)
	for _, c := range snap.Counters {
		countersByName[c.Name] = c.Value
	}

	assert.Equal(t, int64(2), countersByName["conn_pool_scale_up_total"])
	assert.Equal(t, int64(1), countersByName["conn_pool_scale_down_total"])
	assert.Equal(t, int64(3), countersByName["conn_pool_idle_reactivation_total"])
}

func TestConnPoolMetrics_GaugeValues(t *testing.T) {
	root := metrics.New()
	m := newConnPoolMetrics(testMetricsParams(root.Scope()))
	r := newPeerPoolReporter(m)

	r.setCounts(5, 2, 1)

	snap := root.Snapshot()
	gaugesByName := make(map[string]int64)
	for _, g := range snap.Gauges {
		gaugesByName[g.Name] = g.Value
	}

	assert.Equal(t, int64(5), gaugesByName["conn_pool_active_connections"])
	assert.Equal(t, int64(2), gaugesByName["conn_pool_draining_connections"])
	assert.Equal(t, int64(1), gaugesByName["conn_pool_idle_connections"])

	// Re-publishing new counts for the same peer applies deltas so the gauge
	// tracks the latest value rather than accumulating.
	r.setCounts(10, 0, 3)

	snap = root.Snapshot()
	gaugesByName = make(map[string]int64)
	for _, g := range snap.Gauges {
		gaugesByName[g.Name] = g.Value
	}

	assert.Equal(t, int64(10), gaugesByName["conn_pool_active_connections"])
	assert.Equal(t, int64(0), gaugesByName["conn_pool_draining_connections"])
	assert.Equal(t, int64(3), gaugesByName["conn_pool_idle_connections"])
}

// TestPeerPoolReporter_AggregatesAcrossPeers verifies that multiple peers
// sharing one connPoolMetrics contribute to the same gauges (sum across peers)
// and that a peer's contribution returns to zero as its connections are removed
// during teardown.
func TestPeerPoolReporter_AggregatesAcrossPeers(t *testing.T) {
	root := metrics.New()
	m := newConnPoolMetrics(testMetricsParams(root.Scope()))

	peerA := newPeerPoolReporter(m)
	peerB := newPeerPoolReporter(m)

	peerA.setCounts(3, 1, 0)
	peerB.setCounts(2, 0, 2)

	gauges := func() map[string]int64 {
		out := make(map[string]int64)
		for _, g := range root.Snapshot().Gauges {
			out[g.Name] = g.Value
		}
		return out
	}

	g := gauges()
	assert.Equal(t, int64(5), g["conn_pool_active_connections"], "active should sum across peers")
	assert.Equal(t, int64(1), g["conn_pool_draining_connections"])
	assert.Equal(t, int64(2), g["conn_pool_idle_connections"])

	// Peer A tears down (all connections removed): its contribution must drop
	// out of the aggregate, leaving only peer B's counts.
	peerA.setCounts(0, 0, 0)

	g = gauges()
	assert.Equal(t, int64(2), g["conn_pool_active_connections"])
	assert.Equal(t, int64(0), g["conn_pool_draining_connections"])
	assert.Equal(t, int64(2), g["conn_pool_idle_connections"])
}

func TestConnPoolMetrics_Tags(t *testing.T) {
	root := metrics.New()
	m := newConnPoolMetrics(testMetricsParams(root.Scope()))

	m.incScaleUp()
	m.addConnectionCount(1)

	wantTags := wantConnPoolMetricTags(testConnPoolServiceName)

	snap := root.Snapshot()
	for _, c := range snap.Counters {
		for k, v := range wantTags {
			assert.Equal(t, v, c.Tags[k], "counter %s: tag %q", c.Name, k)
		}
		_, hasPeer := c.Tags["peer"]
		assert.False(t, hasPeer, "counter %s must not carry a peer tag", c.Name)
	}
	for _, g := range snap.Gauges {
		for k, v := range wantTags {
			assert.Equal(t, v, g.Tags[k], "gauge %s: tag %q", g.Name, k)
		}
		_, hasPeer := g.Tags["peer"]
		assert.False(t, hasPeer, "gauge %s must not carry a peer tag", g.Name)
	}
}

func TestConnPoolMetrics_ConcurrentAccess(t *testing.T) {
	root := metrics.New()
	m := newConnPoolMetrics(testMetricsParams(root.Scope()))

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m.incScaleUp()
				m.incScaleDown()
				m.incIdleReactivation()
				m.addConnectionCount(int64(j))
				m.addDrainingConnectionCount(int64(j))
				m.addIdleConnectionCount(int64(j))
			}
		}()
	}
	wg.Wait()

	snap := root.Snapshot()
	countersByName := make(map[string]int64)
	for _, c := range snap.Counters {
		countersByName[c.Name] = c.Value
	}

	assert.Equal(t, int64(goroutines*iterations), countersByName["conn_pool_scale_up_total"])
	assert.Equal(t, int64(goroutines*iterations), countersByName["conn_pool_scale_down_total"])
	assert.Equal(t, int64(goroutines*iterations), countersByName["conn_pool_idle_reactivation_total"])
}

func TestNewConnPoolMetrics_LogsRegistrationErrors(t *testing.T) {
	root := metrics.New()
	scope := root.Scope()
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	params := connPoolMetricsParams{
		Meter:       scope,
		Logger:      logger,
		ServiceName: "test-svc",
	}

	// Register the same metrics twice on the same scope+tags to trigger
	// duplicate registration errors.
	_ = newConnPoolMetrics(params)
	_ = newConnPoolMetrics(params)

	assert.NotZero(t, logs.Len(), "expected error logs for duplicate metric registration")
}

// TestConnPoolMetrics_NilReceiver verifies that calling any method on a nil
// *connPoolMetrics pointer does not panic.
func TestConnPoolMetrics_NilReceiver(t *testing.T) {
	var m *connPoolMetrics
	assert.NotPanics(t, func() {
		m.incScaleUp()
		m.incScaleDown()
		m.incIdleReactivation()
		m.addConnectionCount(5)
		m.addDrainingConnectionCount(3)
		m.addIdleConnectionCount(1)
	})

	var r *peerPoolReporter
	assert.NotPanics(t, func() {
		r.setCounts(5, 3, 1)
		r.incScaleUp()
		r.incScaleDown()
		r.incIdleReactivation()
	})
}

func testTransportWithConnPoolMetrics(t *testing.T, opts ...TransportOption) (*Transport, *metrics.Root) {
	t.Helper()
	root := metrics.New()
	baseOpts := []TransportOption{
		Meter(root.Scope()),
		ServiceName(testConnPoolServiceName),
		Logger(zap.NewNop()),
	}
	return NewTransport(append(baseOpts, opts...)...), root
}

func assertConnPoolGaugesZero(t *testing.T, root *metrics.Root) {
	t.Helper()
	g := gaugesFromSnapshot(root.Snapshot())
	assert.Equal(t, int64(0), g["conn_pool_active_connections"])
	assert.Equal(t, int64(0), g["conn_pool_draining_connections"])
	assert.Equal(t, int64(0), g["conn_pool_idle_connections"])
}

// TestConnPoolMetrics_PeerTeardownZerosSharedGauges builds a peer with multiple
// real connections and shared transport metrics, then verifies that stopping
// the peer drives all conn-pool gauges back to zero.
func TestConnPoolMetrics_PeerTeardownZerosSharedGauges(t *testing.T) {
	t.Parallel()

	tr, root := testTransportWithConnPoolMetrics(t)
	p, err := tr.newPeer("127.0.0.1:1", emptyDialOpts)
	require.NoError(t, err)

	require.NoError(t, p.addConn())
	require.NoError(t, p.addConn())

	g := gaugesFromSnapshot(root.Snapshot())
	require.Greater(t, g["conn_pool_active_connections"], int64(0))

	p.stop()
	p.wait()

	assertConnPoolGaugesZero(t, root)
	assertConnPoolMetricTags(t, root.Snapshot(), testConnPoolServiceName)
}

// TestConnPoolMetrics_DynamicScalingTeardownRace hammers scale-up and scale-down
// paths while stopping a dynamically scaled peer and checks that shared gauges
// return to zero. Run with -race -count=1000 to stress the teardown race.
func TestConnPoolMetrics_DynamicScalingTeardownRace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping teardown race stress test in short mode")
	}

	const iterations = 50

	for i := 0; i < iterations; i++ {
		tr, root := testTransportWithConnPoolMetrics(t,
			WithDynamicConnectionScaling(true),
			MinConnections(2),
			MaxConnections(4),
		)
		p, err := tr.newPeer("127.0.0.1:1", emptyDialOpts)
		require.NoError(t, err)

		var wg sync.WaitGroup
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					conn := p.pickConn()
					if conn == nil {
						continue
					}
					p.tryScaleUp(conn)
					p.evaluateScaling()
					atomic.StoreInt32(&conn.streamCount, 85)
				}
			}()
		}

		time.Sleep(5 * time.Millisecond)
		p.stop()
		wg.Wait()
		p.wait()

		assertConnPoolGaugesZero(t, root)
		assertConnPoolMetricTags(t, root.Snapshot(), testConnPoolServiceName)
	}
}
