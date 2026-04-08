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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func testMetricsParams(scope *metrics.Scope) connPoolMetricsParams {
	return connPoolMetricsParams{
		Meter:       scope,
		Logger:      zap.NewNop(),
		ServiceName: "test-svc",
		Peer:        "10.0.0.1:8080",
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
		m.setConnectionCount(5)
		m.setDrainingConnectionCount(3)
		m.setIdleConnectionCount(1)
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

	m.setConnectionCount(5)
	m.setDrainingConnectionCount(2)
	m.setIdleConnectionCount(1)

	snap := root.Snapshot()
	gaugesByName := make(map[string]int64)
	for _, g := range snap.Gauges {
		gaugesByName[g.Name] = g.Value
	}

	assert.Equal(t, int64(5), gaugesByName["conn_pool_active_connections"])
	assert.Equal(t, int64(2), gaugesByName["conn_pool_draining_connections"])
	assert.Equal(t, int64(1), gaugesByName["conn_pool_idle_connections"])

	m.setConnectionCount(10)
	m.setDrainingConnectionCount(0)
	m.setIdleConnectionCount(3)

	snap = root.Snapshot()
	gaugesByName = make(map[string]int64)
	for _, g := range snap.Gauges {
		gaugesByName[g.Name] = g.Value
	}

	assert.Equal(t, int64(10), gaugesByName["conn_pool_active_connections"])
	assert.Equal(t, int64(0), gaugesByName["conn_pool_draining_connections"])
	assert.Equal(t, int64(3), gaugesByName["conn_pool_idle_connections"])
}

func TestConnPoolMetrics_Tags(t *testing.T) {
	root := metrics.New()
	m := newConnPoolMetrics(testMetricsParams(root.Scope()))

	m.incScaleUp()
	m.setConnectionCount(1)

	wantTags := map[string]string{
		"component": "yarpc",
		"service":   "test-svc",
		"transport": "grpc",
		// The metrics library sanitizes colons to underscores in tag values.
		"peer": "10.0.0.1_8080",
	}

	snap := root.Snapshot()
	for _, c := range snap.Counters {
		for k, v := range wantTags {
			assert.Equal(t, v, c.Tags[k], "counter %s: tag %q", c.Name, k)
		}
	}
	for _, g := range snap.Gauges {
		for k, v := range wantTags {
			assert.Equal(t, v, g.Tags[k], "gauge %s: tag %q", g.Name, k)
		}
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
				m.setConnectionCount(int64(j))
				m.setDrainingConnectionCount(int64(j))
				m.setIdleConnectionCount(int64(j))
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
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)

	params := connPoolMetricsParams{
		Meter:       scope,
		Logger:      logger,
		ServiceName: "test-svc",
		Peer:        "10.0.0.1:8080",
	}

	// Register the same metrics twice on the same scope+tags to trigger
	// duplicate registration errors.
	_ = newConnPoolMetrics(params)
	_ = newConnPoolMetrics(params)

	assert.NotZero(t, logs.Len(), "expected error logs for duplicate metric registration")
}
