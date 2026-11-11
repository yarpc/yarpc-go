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

package connpool

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

// bandCountsByLabel returns the connection count per band tag value of the
// gauge family name.
func bandCountsByLabel(root *metrics.Root, name string) map[string]int64 {
	out := map[string]int64{}
	for _, g := range root.Snapshot().Gauges {
		if g.Name == name {
			out[g.Tags["band"]] = g.Value
		}
	}
	return out
}

func TestBandLabels(t *testing.T) {
	assert.Equal(t,
		[]string{"0", "1", "2", "3-5", "6-10", "11-25", "26-50", "51-100", "101-150", "151-200", "201-250", "251-500", "over-500"},
		bandLabels(connectionLoadBuckets))
	assert.Equal(t,
		[]string{"0", "1-10", "11-25", "26-50", "51-70", "71-80", "81-90", "91-100", "over-100"},
		bandLabels(connectionInFlightUtilizationBuckets))
}

func TestStreamBands(t *testing.T) {
	assert.Equal(t,
		[]string{"0", "1", "2", "3-5", "6-10", "11-25", "26-50", "51-100", "101-150", "151-200", "201-250", "over-250"},
		bandLabels(connectionStreamsBuckets))
	counts := StreamBands([]int{0, 3, 250, 251})
	assert.EqualValues(t, 1, counts[0])
	assert.EqualValues(t, 1, counts[3])
	assert.EqualValues(t, 1, counts[10])
	assert.EqualValues(t, 1, counts[11], "past the peer's limit is the overflow band")
}

func TestStreamBands_PoolTeardownWithdraws(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	p := NewPool(context.Background(), fixedConfig(baseConfig()),
		func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil }, zap.NewNop(), "test-pool", NewReporter(shared))
	attachFakeWatcher(p)
	p.StreamState = func(*fakeConn) (int, int, bool) { return 4, 0, true }
	require.NoError(t, p.Start(2, false))

	p.sampleStreamMetrics()
	require.EqualValues(t, 2, bandCountsByLabel(root, "conn_pool_connections_by_streams")["3-5"])

	p.Stop()
	p.Wait()
	for label, n := range bandCountsByLabel(root, "conn_pool_connections_by_streams") {
		assert.Zero(t, n, "band %s", label)
	}
}

func TestBandIndex(t *testing.T) {
	for value, want := range map[int64]string{0: "0", 1: "1", 2: "2", 3: "3-5", 5: "3-5", 6: "6-10", 250: "201-250", 251: "251-500", 500: "251-500", 501: "over-500"} {
		assert.Equal(t, want, bandLabels(connectionLoadBuckets)[bandIndex(connectionLoadBuckets, value)], "value %d", value)
	}
}

func TestInFlightMetrics_WrapperCountsAggregateAcrossPools(t *testing.T) {
	shared := NewMetrics(MetricsParams{Meter: metrics.New().Scope(), Transport: "http2"})
	a, b := makeWrapper(StateActive, 0), makeWrapper(StateActive, 0)
	a.metrics, b.metrics = NewReporter(shared), NewReporter(shared)

	a.IncStreamCount()
	a.IncStreamCount()
	b.IncStreamCount()
	assert.EqualValues(t, 3, shared.inFlightRequests.Load())
	assert.EqualValues(t, 3, shared.requestsTotal.Load())

	a.DecStreamCount()
	b.DecStreamCount()
	assert.EqualValues(t, 1, shared.inFlightRequests.Load())
	assert.EqualValues(t, 3, shared.requestsTotal.Load(), "total only counts starts")
}

// A stream that ends after its pool has been torn down must still withdraw
// from the shared gauge, or it would stay inflated forever.
func TestInFlightMetrics_StreamFinishingAfterPoolTeardown(t *testing.T) {
	shared, reporter := newTestReporter()
	w := makeWrapper(StateActive, 0)
	p := newBarePool(t, baseConfig(), reporter, w)
	w.metrics = p.metrics

	w.IncStreamCount()
	p.Stop()
	w.DecStreamCount()
	assert.EqualValues(t, 0, shared.inFlightRequests.Load())
}

func TestInFlightMetrics_WrapperWithoutReporterIsSafe(t *testing.T) {
	w := makeWrapper(StateActive, 0)
	w.IncStreamCount()
	w.DecStreamCount()
	assert.EqualValues(t, 0, w.StreamCount())
}

func TestInFlightMetrics_ConcurrentStreamsSumExactly(t *testing.T) {
	shared, reporter := newTestReporter()
	w := makeWrapper(StateActive, 0)
	w.metrics = reporter

	const goroutines, per = 16, 500
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < per; j++ {
				w.IncStreamCount()
				w.DecStreamCount()
			}
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 0, shared.inFlightRequests.Load())
	assert.EqualValues(t, goroutines*per, shared.requestsTotal.Load())
}

func TestInFlightMetrics_SampleRecordsActiveConnectionsOnly(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	cfg := baseConfig() // MaxConcurrentStreams 250
	p := newBarePool(t, cfg, NewReporter(shared),
		makeWrapper(StateActive, 0),
		makeWrapper(StateActive, 200),
		makeWrapper(StateDraining, 40),
		makeWrapper(StateIdle, 0),
	)

	p.sampleStreamMetrics()

	byInFlight := bandCountsByLabel(root, "conn_pool_connections_by_in_flight_requests")
	assert.Len(t, byInFlight, 13, "every band is reported, empty ones as 0")
	assert.EqualValues(t, 1, byInFlight["0"])
	assert.EqualValues(t, 1, byInFlight["151-200"])
	// 0% and 200/250 = 80%.
	byUtilization := bandCountsByLabel(root, "conn_pool_connections_by_in_flight_utilization_percent")
	assert.Len(t, byUtilization, 9)
	assert.EqualValues(t, 1, byUtilization["0"])
	assert.EqualValues(t, 1, byUtilization["71-80"])
}

func TestLoadBands_OverLimitLandsInOverflowBand(t *testing.T) {
	_, byUtilization := LoadBands([]int32{300}, 250) // 120%
	assert.EqualValues(t, 1, byUtilization[len(byUtilization)-1])
}

func TestLoadBands_ZeroMaxStreamsSkipsUtilization(t *testing.T) {
	byInFlight, byUtilization := LoadBands([]int32{5}, 0)
	assert.EqualValues(t, 1, byInFlight[3], "5 in flight is the 3-5 band")
	assert.Equal(t, make([]int64, len(connectionInFlightUtilizationBuckets)+1), byUtilization)
}

func TestLoadBands_FollowLatestSampleAndAggregateAcrossPools(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	a, b := NewReporter(shared), NewReporter(shared)
	const name = "conn_pool_connections_by_in_flight_requests"

	a.SetLoadBands(LoadBands([]int32{0, 4}, 250))
	b.SetLoadBands(LoadBands([]int32{4}, 250))
	assert.EqualValues(t, 1, bandCountsByLabel(root, name)["0"])
	assert.EqualValues(t, 2, bandCountsByLabel(root, name)["3-5"])

	// A new sample replaces the pool's earlier one rather than adding to it.
	a.SetLoadBands(LoadBands([]int32{4}, 250))
	assert.EqualValues(t, 0, bandCountsByLabel(root, name)["0"])
	assert.EqualValues(t, 2, bandCountsByLabel(root, name)["3-5"])

	// Withdrawing a pool leaves the others' share.
	a.SetLoadBands(nil, nil)
	assert.EqualValues(t, 1, bandCountsByLabel(root, name)["3-5"])
}

func TestLoadBands_PoolTeardownWithdrawsItsBands(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	p := NewPool(context.Background(), fixedConfig(baseConfig()),
		func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil }, zap.NewNop(), "test-pool", NewReporter(shared))
	attachFakeWatcher(p)
	require.NoError(t, p.Start(2, false))

	p.sampleStreamMetrics()
	require.EqualValues(t, 2, bandCountsByLabel(root, "conn_pool_connections_by_in_flight_requests")["0"])

	p.Stop()
	p.Wait()
	for label, n := range bandCountsByLabel(root, "conn_pool_connections_by_in_flight_requests") {
		assert.Zero(t, n, "band %s", label)
	}
}

func TestInFlightMetrics_NilSafe(t *testing.T) {
	var r *Reporter
	r.streamStarted()
	r.streamFinished()
	r.IncScaleUpBlocked()
	r.SetLoadBands([]int64{1}, []int64{1})

	(&Metrics{}).addBands(nil, []int64{1})
	(&Metrics{}).addInFlightRequests(1)
	(&Metrics{}).incRequests()
	(&Metrics{}).addActiveStreams(1)
	(&Metrics{}).addPendingRequests(1)
	r.SetStreamBands([]int64{1})
	r.SetStreams(1, 1)
	(&Metrics{}).incScaleUpBlocked()
}

func TestInFlightMetrics_ScaleUpBlockedAtMaxConnections(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConnections = 2
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	shared, reporter := newTestReporter()
	busy := makeWrapper(StateActive, 1)
	p := newBarePool(t, cfg, reporter, busy, makeWrapper(StateActive, 1))

	p.TryScaleUp(busy)
	require.Eventually(t, func() bool { return shared.scaleUpBlockedTotal.Load() == 1 },
		time.Second, time.Millisecond)

	// Unlike the log line, the counter is not limited to once per episode.
	require.Eventually(t, func() bool {
		p.TryScaleUp(busy)
		return shared.scaleUpBlockedTotal.Load() >= 2
	}, time.Second, time.Millisecond)
	assert.EqualValues(t, 0, shared.scaleUpTotal.Load())
}

func TestInFlightMetrics_RegisteredWithTags(t *testing.T) {
	root := metrics.New()
	m := NewMetrics(MetricsParams{Meter: root.Scope(), ServiceName: "svc", Transport: "grpc", MetricPrefix: "p"})
	m.addInFlightRequests(2)
	m.incRequests()
	m.addActiveStreams(1)
	m.addPendingRequests(1)
	m.addBands(m.connectionsByStreams, []int64{1})
	m.incScaleUpBlocked()
	m.addBands(m.connectionsByInFlight, []int64{1})

	snap := root.Snapshot()
	names := map[string]metrics.Tags{}
	for _, g := range snap.Gauges {
		names[g.Name] = g.Tags
	}
	for _, c := range snap.Counters {
		names[c.Name] = c.Tags
	}
	for _, n := range []string{"p_in_flight_requests", "p_requests_total", "p_scale_up_blocked_total",
		"p_active_streams", "p_pending_requests"} {
		tags, ok := names[n]
		require.True(t, ok, n)
		assert.Equal(t, metrics.Tags{"component": "yarpc", "service": "svc", "transport": "grpc"}, tags, n)
	}
	// The band gauges add the band tag to the same constant tags.
	for _, n := range []string{"p_connections_by_in_flight_requests", "p_connections_by_in_flight_utilization_percent", "p_connections_by_streams"} {
		bands := bandCountsByLabel(root, n)
		assert.NotEmpty(t, bands, n)
		for _, g := range snap.Gauges {
			if g.Name == n {
				assert.Equal(t, "yarpc", g.Tags["component"], n)
				assert.Equal(t, "svc", g.Tags["service"], n)
				assert.Equal(t, "grpc", g.Tags["transport"], n)
				assert.NotEmpty(t, g.Tags["band"], n)
			}
		}
	}
}

func TestClosedConnections_CountsIdleTimeoutClosesOnly(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = time.Minute
	shared, reporter := newTestReporter()
	expired := makeIdleWrapper(time.Now().Add(-2 * time.Minute))
	fresh := makeIdleWrapper(time.Now())
	p := newBarePool(t, cfg, reporter, makeWrapper(StateActive, 0), expired, fresh)

	p.CleanupIdleConns()
	assert.EqualValues(t, 1, shared.closedConnectionsTotal.Load(), "only the expired idle connection closes")
	assert.Equal(t, StateClosing, expired.GetState())

	p.CleanupIdleConns()
	assert.EqualValues(t, 1, shared.closedConnectionsTotal.Load(), "a closing connection is not counted twice")
}

func TestClosedConnections_ReactivatedConnectionIsNotCounted(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = time.Minute
	shared, reporter := newTestReporter()
	w := makeIdleWrapper(time.Now().Add(-2 * time.Minute))
	p := newBarePool(t, cfg, reporter, w)

	require.True(t, w.TransitionState(StateIdle, StateActive))
	p.CleanupIdleConns()
	assert.EqualValues(t, 0, shared.closedConnectionsTotal.Load())
}

func TestClosedConnections_NilSafeAndTagged(t *testing.T) {
	var r *Reporter
	r.IncClosed()
	(&Metrics{}).incClosed()

	root := metrics.New()
	m := NewMetrics(MetricsParams{Meter: root.Scope(), ServiceName: "svc", Transport: "http2"})
	m.incClosed()
	for _, c := range root.Snapshot().Counters {
		if c.Name == "conn_pool_closed_connections_total" {
			assert.EqualValues(t, 1, c.Value)
			assert.Equal(t, metrics.Tags{"component": "yarpc", "service": "svc", "transport": "http2"}, c.Tags)
			return
		}
	}
	t.Fatal("closed_connections_total not registered")
}

// fakeStreamState returns a StreamState reading per-connection counts from m.
// A connection missing from m reports ok=false.
func fakeStreamState(m map[*fakeConn][2]int) func(*fakeConn) (int, int, bool) {
	return func(c *fakeConn) (int, int, bool) {
		v, ok := m[c]
		return v[0], v[1], ok
	}
}

// The in-flight count and the real stream count are separate: a connection
// holding more requests than the peer allows streams reports the excess as
// pending, not as streams.
func TestStreams_InFlightAndOpenStreamsAreReportedSeparately(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 10

	saturated := makeWrapper(StateActive, 14) // 10 on the wire, 4 queued
	calm := makeWrapper(StateActive, 2)
	draining := makeWrapper(StateDraining, 3)
	p := newBarePool(t, cfg, NewReporter(shared), saturated, calm, draining)
	p.StreamState = fakeStreamState(map[*fakeConn][2]int{
		saturated.Conn: {10, 4},
		calm.Conn:      {2, 0},
		draining.Conn:  {3, 0},
	})

	p.sampleStreamMetrics()

	assert.EqualValues(t, 15, shared.activeStreams.Load(), "open streams across all connections, draining included")
	assert.EqualValues(t, 4, shared.pendingRequests.Load())
	// The in-flight bands are the pool's own count over active connections, so
	// the saturated one (14) is above the stream limit; streams never are.
	byInFlight := bandCountsByLabel(root, "conn_pool_connections_by_in_flight_requests")
	assert.EqualValues(t, 1, byInFlight["2"])
	assert.EqualValues(t, 1, byInFlight["11-25"])
	// 20% and 140%; 140% is past the last bound, so it is in the overflow band.
	byUtilization := bandCountsByLabel(root, "conn_pool_connections_by_in_flight_utilization_percent")
	assert.EqualValues(t, 1, byUtilization["11-25"])
	assert.EqualValues(t, 1, byUtilization["over-100"])
	// Open streams are capped at the stream limit: 10 and 2 (draining excluded).
	byStreams := bandCountsByLabel(root, "conn_pool_connections_by_streams")
	assert.EqualValues(t, 1, byStreams["2"])
	assert.EqualValues(t, 1, byStreams["6-10"])
}

func TestStreams_NoStreamStateReportsOnlyInFlight(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "grpc"})
	p := newBarePool(t, baseConfig(), NewReporter(shared), makeWrapper(StateActive, 3))

	p.sampleStreamMetrics()

	assert.EqualValues(t, 1, bandCountsByLabel(root, "conn_pool_connections_by_in_flight_requests")["3-5"])
	for label, n := range bandCountsByLabel(root, "conn_pool_connections_by_streams") {
		assert.Zero(t, n, "band %s", label)
	}
	assert.EqualValues(t, 0, shared.activeStreams.Load())
	assert.EqualValues(t, 0, shared.pendingRequests.Load())
}

func TestStreams_ConnectionThatCannotReportIsSkipped(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	good, closed := makeWrapper(StateActive, 1), makeWrapper(StateActive, 1)
	p := newBarePool(t, baseConfig(), NewReporter(shared), good, closed)
	p.StreamState = fakeStreamState(map[*fakeConn][2]int{good.Conn: {1, 0}})

	p.sampleStreamMetrics()

	assert.EqualValues(t, 1, shared.activeStreams.Load())
	byStreams := bandCountsByLabel(root, "conn_pool_connections_by_streams")
	assert.EqualValues(t, 1, byStreams["1"], "only the connection that reported is banded")
	var total int64
	for _, n := range byStreams {
		total += n
	}
	assert.EqualValues(t, 1, total)
}

func TestStreams_GaugesFollowTheLatestSampleNotAccumulate(t *testing.T) {
	shared, reporter := newTestReporter()
	w := makeWrapper(StateActive, 0)
	state := map[*fakeConn][2]int{w.Conn: {8, 3}}
	p := newBarePool(t, baseConfig(), reporter, w)
	p.StreamState = fakeStreamState(state)

	p.sampleStreamMetrics()
	p.sampleStreamMetrics()
	assert.EqualValues(t, 8, shared.activeStreams.Load())
	assert.EqualValues(t, 3, shared.pendingRequests.Load())

	state[w.Conn] = [2]int{2, 0}
	p.sampleStreamMetrics()
	assert.EqualValues(t, 2, shared.activeStreams.Load())
	assert.EqualValues(t, 0, shared.pendingRequests.Load())
}

func TestStreams_SetStreamsAggregatesAcrossPools(t *testing.T) {
	shared := NewMetrics(MetricsParams{Meter: metrics.New().Scope(), Transport: "http2"})
	a, b := NewReporter(shared), NewReporter(shared)

	a.SetStreams(5, 1)
	b.SetStreams(3, 2)
	assert.EqualValues(t, 8, shared.activeStreams.Load())
	assert.EqualValues(t, 3, shared.pendingRequests.Load())

	a.SetStreams(0, 0)
	assert.EqualValues(t, 3, shared.activeStreams.Load(), "a pool's teardown leaves the others' share")
	assert.EqualValues(t, 2, shared.pendingRequests.Load())
}

func TestStreams_PoolTeardownWithdrawsItsStreams(t *testing.T) {
	shared, reporter := newTestReporter()
	p := NewPool(context.Background(), fixedConfig(baseConfig()),
		func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil }, zap.NewNop(), "test-pool", reporter)
	attachFakeWatcher(p)
	p.StreamState = func(*fakeConn) (int, int, bool) { return 4, 2, true }
	require.NoError(t, p.Start(1, false))

	p.sampleStreamMetrics()
	require.EqualValues(t, 4, shared.activeStreams.Load())

	p.Stop()
	p.Wait()
	assert.EqualValues(t, 0, shared.activeStreams.Load())
	assert.EqualValues(t, 0, shared.pendingRequests.Load())
}
