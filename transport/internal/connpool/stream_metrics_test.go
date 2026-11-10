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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
)

func histogramValues(root *metrics.Root, name string) []int64 {
	for _, h := range root.Snapshot().Histograms {
		if h.Name == name {
			return h.Values
		}
	}
	return nil
}

func TestStreamMetrics_WrapperCountsAggregateAcrossPools(t *testing.T) {
	shared := NewMetrics(MetricsParams{Meter: metrics.New().Scope(), Transport: "http2"})
	a, b := makeWrapper(StateActive, 0), makeWrapper(StateActive, 0)
	a.metrics, b.metrics = NewReporter(shared), NewReporter(shared)

	a.IncStreamCount()
	a.IncStreamCount()
	b.IncStreamCount()
	assert.EqualValues(t, 3, shared.activeStreams.Load())
	assert.EqualValues(t, 3, shared.streamsTotal.Load())

	a.DecStreamCount()
	b.DecStreamCount()
	assert.EqualValues(t, 1, shared.activeStreams.Load())
	assert.EqualValues(t, 3, shared.streamsTotal.Load(), "total only counts starts")
}

// A stream that ends after its pool has been torn down must still withdraw
// from the shared gauge, or it would stay inflated forever.
func TestStreamMetrics_StreamFinishingAfterPoolTeardown(t *testing.T) {
	shared, reporter := newTestReporter()
	w := makeWrapper(StateActive, 0)
	p := newBarePool(t, baseConfig(), reporter, w)
	w.metrics = p.metrics

	w.IncStreamCount()
	p.Stop()
	w.DecStreamCount()
	assert.EqualValues(t, 0, shared.activeStreams.Load())
}

func TestStreamMetrics_WrapperWithoutReporterIsSafe(t *testing.T) {
	w := makeWrapper(StateActive, 0)
	w.IncStreamCount()
	w.DecStreamCount()
	assert.EqualValues(t, 0, w.StreamCount())
}

func TestStreamMetrics_ConcurrentStreamsSumExactly(t *testing.T) {
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
	assert.EqualValues(t, 0, shared.activeStreams.Load())
	assert.EqualValues(t, goroutines*per, shared.streamsTotal.Load())
}

func TestStreamMetrics_SampleRecordsActiveConnectionsOnly(t *testing.T) {
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

	assert.ElementsMatch(t, []int64{0, 200}, histogramValues(root, "conn_pool_connection_streams"))
	// 0% and 200/250 = 80%.
	assert.ElementsMatch(t, []int64{0, 80}, histogramValues(root, "conn_pool_connection_stream_utilization_percent"))
}

func TestStreamMetrics_OverLimitLandsInCatchAllBucket(t *testing.T) {
	root := metrics.New()
	m := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	m.observeConnection(300, 250) // 120%

	vals := histogramValues(root, "conn_pool_connection_stream_utilization_percent")
	require.Len(t, vals, 1)
	assert.Greater(t, vals[0], int64(100))
}

func TestStreamMetrics_ZeroMaxStreamsSkipsUtilization(t *testing.T) {
	root := metrics.New()
	m := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	m.observeConnection(5, 0)

	assert.Len(t, histogramValues(root, "conn_pool_connection_streams"), 1)
	assert.Empty(t, histogramValues(root, "conn_pool_connection_stream_utilization_percent"))
}

func TestStreamMetrics_NilSafe(t *testing.T) {
	var r *Reporter
	r.streamStarted()
	r.streamFinished()
	r.IncScaleUpBlocked()
	r.ObserveConnection(1, 250)

	(&Metrics{}).observeConnection(1, 250)
	(&Metrics{}).addActiveStreams(1)
	(&Metrics{}).incStreams()
	(&Metrics{}).incScaleUpBlocked()
}

func TestStreamMetrics_ScaleUpBlockedAtMaxConnections(t *testing.T) {
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

func TestStreamMetrics_RegisteredWithTags(t *testing.T) {
	root := metrics.New()
	m := NewMetrics(MetricsParams{Meter: root.Scope(), ServiceName: "svc", Transport: "grpc", MetricPrefix: "p"})
	m.addActiveStreams(2)
	m.incStreams()
	m.incScaleUpBlocked()
	m.observeConnection(1, 100)

	snap := root.Snapshot()
	names := map[string]metrics.Tags{}
	for _, g := range snap.Gauges {
		names[g.Name] = g.Tags
	}
	for _, c := range snap.Counters {
		names[c.Name] = c.Tags
	}
	for _, h := range snap.Histograms {
		names[h.Name] = h.Tags
	}
	for _, n := range []string{"p_active_streams", "p_streams_total", "p_scale_up_blocked_total",
		"p_connection_streams", "p_connection_stream_utilization_percent"} {
		tags, ok := names[n]
		require.True(t, ok, n)
		assert.Equal(t, metrics.Tags{"component": "yarpc", "service": "svc", "transport": "grpc"}, tags, n)
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
