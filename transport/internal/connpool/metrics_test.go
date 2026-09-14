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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

func TestMetrics_NilScopeReturnsNilSafeHandles(t *testing.T) {
	m := NewMetrics(MetricsParams{})
	require.NotNil(t, m)
	// Every accessor must be a no-op, not a panic, when Meter was nil.
	m.addConnectionCount(1)
	m.addDrainingConnectionCount(1)
	m.addIdleConnectionCount(1)
	m.incScaleUp()
	m.incScaleDown()
	m.incIdleReactivation()
}

func TestMetrics_ValidScopeRegistersAndTags(t *testing.T) {
	root := metrics.New()
	scope := root.Scope()
	m := NewMetrics(MetricsParams{
		Meter:        scope,
		ServiceName:  "myservice",
		Transport:    "http2",
		MetricPrefix: "http2_conn_pool",
	})

	require.NotNil(t, m.connectionCount)
	require.NotNil(t, m.scaleUpTotal)

	m.addConnectionCount(3)
	m.incScaleUp()

	snap := root.Snapshot()
	var foundGauge, foundCounter bool
	for _, g := range snap.Gauges {
		if g.Name == "http2_conn_pool_active_connections" {
			foundGauge = true
			assert.EqualValues(t, 3, g.Value)
			assert.Equal(t, "http2", g.Tags["transport"])
			assert.Equal(t, "myservice", g.Tags["service"])
		}
	}
	for _, c := range snap.Counters {
		if c.Name == "http2_conn_pool_scale_up_total" {
			foundCounter = true
			assert.EqualValues(t, 1, c.Value)
		}
	}
	assert.True(t, foundGauge, "expected active_connections gauge to be registered")
	assert.True(t, foundCounter, "expected scale_up_total counter to be registered")
}

func TestMetrics_DefaultPrefixWhenUnset(t *testing.T) {
	root := metrics.New()
	m := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "grpc"})
	require.NotNil(t, m.connectionCount)

	snap := root.Snapshot()
	found := false
	for _, g := range snap.Gauges {
		if g.Name == "conn_pool_active_connections" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestMetrics_RegistrationErrorsAreLoggedNotFatal(t *testing.T) {
	root := metrics.New()
	params := MetricsParams{
		Meter:        root.Scope(),
		ServiceName:  "svc",
		Transport:    "http2",
		MetricPrefix: "dup_conn_pool",
		Logger:       zap.NewNop(),
	}
	first := NewMetrics(params)
	require.NotNil(t, first.connectionCount)

	// Registering the exact same name/tags a second time on the same root
	// makes every Gauge/Counter call return an error, exercising the
	// registration-error log branches without a panic.
	second := NewMetrics(params)
	assert.Nil(t, second.connectionCount)
	assert.Nil(t, second.drainingConnectionCount)
	assert.Nil(t, second.idleConnectionCount)
	assert.Nil(t, second.scaleUpTotal)
	assert.Nil(t, second.scaleDownTotal)
	assert.Nil(t, second.idleReactivationTotal)

	// Nil-safe accessors on a Metrics whose handles all failed to register.
	second.addConnectionCount(1)
	second.incScaleUp()
}

func TestReporter_AllIncrementsAndIdleDelta(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	r := NewReporter(shared)

	r.IncScaleUp()
	r.IncScaleDown()
	r.IncIdleReactivation()
	r.SetCounts(0, 0, 5)

	snap := root.Snapshot()
	for _, c := range snap.Counters {
		switch c.Name {
		case "conn_pool_scale_up_total", "conn_pool_scale_down_total", "conn_pool_idle_reactivation_total":
			assert.EqualValues(t, 1, c.Value, c.Name)
		}
	}
	for _, g := range snap.Gauges {
		if g.Name == "conn_pool_idle_connections" {
			assert.EqualValues(t, 5, g.Value)
		}
	}
}

func TestReporter_NilReceiverSafe(t *testing.T) {
	var r *Reporter
	r.SetCounts(1, 2, 3)
	r.IncScaleUp()
	r.IncScaleDown()
	r.IncIdleReactivation()
}

func TestReporter_AggregatesAcrossPools(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})

	r1 := NewReporter(shared)
	r2 := NewReporter(shared)

	r1.SetCounts(2, 0, 0)
	r2.SetCounts(3, 1, 0)

	snap := root.Snapshot()
	for _, g := range snap.Gauges {
		switch g.Name {
		case "conn_pool_active_connections":
			assert.EqualValues(t, 5, g.Value)
		case "conn_pool_draining_connections":
			assert.EqualValues(t, 1, g.Value)
		}
	}

	// A pool tearing down withdraws exactly its own contribution.
	r1.SetCounts(0, 0, 0)
	snap = root.Snapshot()
	for _, g := range snap.Gauges {
		if g.Name == "conn_pool_active_connections" {
			assert.EqualValues(t, 3, g.Value)
		}
	}
}

// TestReporter_ConcurrentIncrementsExactTotals ports transport/grpc's
// TestConnPoolMetrics_ConcurrentAccess: unlike TestReporter_ConcurrentSetCounts
// below (which only checks for data races), this asserts the exact counter
// totals after concurrent increments, verifying no increments are lost.
func TestReporter_ConcurrentIncrementsExactTotals(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	r := NewReporter(shared)

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				r.IncScaleUp()
				r.IncScaleDown()
				r.IncIdleReactivation()
			}
		}()
	}
	wg.Wait()

	countersByName := make(map[string]int64)
	for _, c := range root.Snapshot().Counters {
		countersByName[c.Name] = c.Value
	}
	assert.EqualValues(t, goroutines*iterations, countersByName["conn_pool_scale_up_total"])
	assert.EqualValues(t, goroutines*iterations, countersByName["conn_pool_scale_down_total"])
	assert.EqualValues(t, goroutines*iterations, countersByName["conn_pool_idle_reactivation_total"])
}

func TestReporter_ConcurrentSetCounts(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	r := NewReporter(shared)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(n int64) {
			defer wg.Done()
			r.SetCounts(n, 0, 0)
		}(int64(i))
	}
	wg.Wait()
	// No assertion on the final value (last writer wins non-deterministically);
	// this test exists to catch data races under `go test -race`.
}
