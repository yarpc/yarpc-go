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
	"go.uber.org/zap/zaptest/observer"
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
	first := newMetrics(params)
	require.NotNil(t, first.connectionCount)

	// Registering the exact same name/tags a second time on the same root
	// makes every Gauge/Counter call return an error, exercising the
	// registration-error log branches without a panic. newMetrics is used
	// because NewMetrics hands repeat callers the first set instead.
	second := newMetrics(params)
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

// --- ported from the former transport/grpc metrics tests ---

func TestMetrics_NilScopeLeavesHandlesNil(t *testing.T) {
	m := NewMetrics(MetricsParams{})
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
		NewReporter(m).SetCounts(5, 3, 1)
	})
}

func TestMetrics_ValidScopeRegistersAllHandles(t *testing.T) {
	m := NewMetrics(MetricsParams{Meter: metrics.New().Scope(), Transport: "grpc"})
	require.NotNil(t, m)
	assert.NotNil(t, m.connectionCount)
	assert.NotNil(t, m.drainingConnectionCount)
	assert.NotNil(t, m.idleConnectionCount)
	assert.NotNil(t, m.scaleUpTotal)
	assert.NotNil(t, m.scaleDownTotal)
	assert.NotNil(t, m.idleReactivationTotal)
}

func TestReporter_CounterIncrementValues(t *testing.T) {
	root := metrics.New()
	r := NewReporter(NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "grpc"}))

	r.IncScaleUp()
	r.IncScaleUp()
	r.IncScaleDown()
	r.IncIdleReactivation()
	r.IncIdleReactivation()
	r.IncIdleReactivation()

	counters := make(map[string]int64)
	for _, c := range root.Snapshot().Counters {
		counters[c.Name] = c.Value
	}
	assert.EqualValues(t, 2, counters["conn_pool_scale_up_total"])
	assert.EqualValues(t, 1, counters["conn_pool_scale_down_total"])
	assert.EqualValues(t, 3, counters["conn_pool_idle_reactivation_total"])
}

func gaugeValues(root *metrics.Root) map[string]int64 {
	out := make(map[string]int64)
	for _, g := range root.Snapshot().Gauges {
		out[g.Name] = g.Value
	}
	return out
}

func TestReporter_SetCountsAppliesDeltasNotAccumulation(t *testing.T) {
	root := metrics.New()
	r := NewReporter(NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "grpc"}))

	r.SetCounts(5, 2, 1)
	g := gaugeValues(root)
	assert.EqualValues(t, 5, g["conn_pool_active_connections"])
	assert.EqualValues(t, 2, g["conn_pool_draining_connections"])
	assert.EqualValues(t, 1, g["conn_pool_idle_connections"])

	// Re-publishing for the same pool tracks the latest value rather than
	// accumulating, including shrinking a gauge to zero.
	r.SetCounts(10, 0, 3)
	g = gaugeValues(root)
	assert.EqualValues(t, 10, g["conn_pool_active_connections"])
	assert.EqualValues(t, 0, g["conn_pool_draining_connections"])
	assert.EqualValues(t, 3, g["conn_pool_idle_connections"])
}

func TestReporter_TeardownLeavesOtherPoolsContribution(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "grpc"})
	poolA := NewReporter(shared)
	poolB := NewReporter(shared)

	poolA.SetCounts(3, 1, 0)
	poolB.SetCounts(2, 0, 2)
	g := gaugeValues(root)
	assert.EqualValues(t, 5, g["conn_pool_active_connections"], "active should sum across pools")
	assert.EqualValues(t, 1, g["conn_pool_draining_connections"])
	assert.EqualValues(t, 2, g["conn_pool_idle_connections"])

	poolA.SetCounts(0, 0, 0)
	g = gaugeValues(root)
	assert.EqualValues(t, 2, g["conn_pool_active_connections"])
	assert.EqualValues(t, 0, g["conn_pool_draining_connections"])
	assert.EqualValues(t, 2, g["conn_pool_idle_connections"])
}

func TestMetrics_TagsAreComponentServiceTransportWithoutPeer(t *testing.T) {
	root := metrics.New()
	m := NewMetrics(MetricsParams{Meter: root.Scope(), ServiceName: "test-svc", Transport: "grpc"})

	m.incScaleUp()
	m.addConnectionCount(1)

	want := map[string]string{
		"component": "yarpc",
		"service":   "test-svc",
		"transport": "grpc",
	}
	snap := root.Snapshot()
	require.NotEmpty(t, snap.Counters)
	require.NotEmpty(t, snap.Gauges)
	for _, c := range snap.Counters {
		for k, v := range want {
			assert.Equal(t, v, c.Tags[k], "counter %s: tag %q", c.Name, k)
		}
		assert.NotContains(t, c.Tags, "peer", "counter %s must not carry a peer tag", c.Name)
	}
	for _, g := range snap.Gauges {
		for k, v := range want {
			assert.Equal(t, v, g.Tags[k], "gauge %s: tag %q", g.Name, k)
		}
		assert.NotContains(t, g.Tags, "peer", "gauge %s must not carry a peer tag", g.Name)
	}
}

func TestMetrics_ConcurrentAccessSumsExactly(t *testing.T) {
	root := metrics.New()
	m := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "grpc"})

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for j := range iterations {
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
	counters := make(map[string]int64)
	for _, c := range snap.Counters {
		counters[c.Name] = c.Value
	}
	assert.EqualValues(t, goroutines*iterations, counters["conn_pool_scale_up_total"])
	assert.EqualValues(t, goroutines*iterations, counters["conn_pool_scale_down_total"])
	assert.EqualValues(t, goroutines*iterations, counters["conn_pool_idle_reactivation_total"])

	wantGauge := int64(goroutines * (iterations * (iterations - 1) / 2))
	g := gaugeValues(root)
	assert.Equal(t, wantGauge, g["conn_pool_active_connections"])
	assert.Equal(t, wantGauge, g["conn_pool_draining_connections"])
	assert.Equal(t, wantGauge, g["conn_pool_idle_connections"])
}

func TestMetrics_RegistrationErrorsAreLogged(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	params := MetricsParams{
		Meter:       metrics.New().Scope(),
		Logger:      zap.New(core),
		ServiceName: "test-svc",
		Transport:   "grpc",
	}

	_ = newMetrics(params)
	require.Zero(t, logs.Len(), "first registration must not log")

	// The second registration on the same scope and tags fails for each of the
	// six handles, and each failure is logged.
	_ = newMetrics(params)
	assert.Equal(t, 6, logs.Len(), "expected one warning per duplicate metric registration")
	for _, e := range logs.All() {
		assert.Equal(t, zap.WarnLevel, e.Level)
	}
}

func TestMetrics_NilReceiverSafe(t *testing.T) {
	var m *Metrics
	assert.NotPanics(t, func() {
		m.incScaleUp()
		m.incScaleDown()
		m.incIdleReactivation()
		m.addConnectionCount(5)
		m.addDrainingConnectionCount(3)
		m.addIdleConnectionCount(1)
	})
}

// observedParams returns MetricsParams on the given scope whose warnings can be
// read back from the returned log.
func observedParams(scope *metrics.Scope) (MetricsParams, *observer.ObservedLogs) {
	core, logs := observer.New(zap.WarnLevel)
	return MetricsParams{
		Meter:       scope,
		Logger:      zap.New(core),
		ServiceName: "test-svc",
		Transport:   "grpc",
	}, logs
}

// A process can build several transports on one meter and service name. The
// second used to fail every registration, warn six times, and report nothing.
func TestMetrics_RepeatCallersShareOneSetWithoutWarnings(t *testing.T) {
	params, logs := observedParams(metrics.New().Scope())

	first := NewMetrics(params)
	second := NewMetrics(params)

	assert.Same(t, first, second)
	assert.Zero(t, logs.Len(), "a repeat caller must not log registration warnings")
	require.NotNil(t, second.connectionCount)
	require.NotNil(t, second.scaleUpTotal)

	// Each transport's pools report through their own Reporter; sharing the
	// handles makes the gauge the aggregate across both.
	r1, r2 := NewReporter(first), NewReporter(second)
	r1.SetCounts(2, 1, 0)
	r2.SetCounts(3, 0, 1)
	active, draining, idle := reportedGauges(first)
	assert.Equal(t, int64(5), active)
	assert.Equal(t, int64(1), draining)
	assert.Equal(t, int64(1), idle)

	r1.IncScaleUp()
	r2.IncScaleUp()
	assert.Equal(t, int64(2), first.scaleUpTotal.Load())

	// One transport shutting down only removes its own contribution.
	r1.SetCounts(0, 0, 0)
	active, _, _ = reportedGauges(first)
	assert.Equal(t, int64(3), active)
}

func TestMetrics_DifferentKeysRegisterSeparately(t *testing.T) {
	scope := metrics.New().Scope()
	base, logs := observedParams(scope)

	other := func(mod func(*MetricsParams)) *Metrics {
		p := base
		mod(&p)
		return NewMetrics(p)
	}
	m := NewMetrics(base)
	for name, got := range map[string]*Metrics{
		"service":   other(func(p *MetricsParams) { p.ServiceName = "other-svc" }),
		"transport": other(func(p *MetricsParams) { p.Transport = "http2" }),
		"prefix":    other(func(p *MetricsParams) { p.MetricPrefix = "other_conn_pool" }),
		"root":      other(func(p *MetricsParams) { p.Meter = metrics.New().Scope() }),
	} {
		assert.NotSame(t, m, got, name)
		assert.NotNil(t, got.connectionCount, name)
	}
	assert.Zero(t, logs.Len(), "distinct metrics must all register cleanly")
}

func TestMetrics_ConcurrentCallersRegisterOnce(t *testing.T) {
	params, logs := observedParams(metrics.New().Scope())

	const callers = 50
	got := make([]*Metrics, callers)
	var wg sync.WaitGroup
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[i] = NewMetrics(params)
		}()
	}
	wg.Wait()

	for _, m := range got {
		assert.Same(t, got[0], m)
	}
	assert.Zero(t, logs.Len())
}

// A collision with something registered outside NewMetrics is still reported,
// once, and is not repeated for later callers.
func TestMetrics_ExternalRegistrationWarnsOnce(t *testing.T) {
	scope := metrics.New().Scope()
	params, logs := observedParams(scope)
	_, err := scope.Gauge(metrics.Spec{
		Name: "conn_pool_active_connections",
		Help: "registered outside NewMetrics",
		ConstTags: metrics.Tags{
			componentTag: componentYarpc,
			serviceTag:   params.ServiceName,
			transportTag: params.Transport,
		},
	})
	require.NoError(t, err)

	first := NewMetrics(params)
	require.Equal(t, 1, logs.Len(), "only the colliding metric should warn")
	assert.Equal(t, "failed to create active connections gauge", logs.All()[0].Message)
	assert.Nil(t, first.connectionCount)
	assert.NotNil(t, first.scaleUpTotal, "the other metrics register normally")

	assert.Same(t, first, NewMetrics(params))
	assert.Equal(t, 1, logs.Len(), "a repeat caller must not warn again")
}

// Tagged returns a new *Scope on every call, even for identical tags, and
// registering the same metrics through two such scopes collides. Transports
// given separately tagged copies must still register once and share the set.
func TestMetrics_EquivalentScopesShareOneSet(t *testing.T) {
	root := metrics.New()
	params, logs := observedParams(root.Scope().Tagged(metrics.Tags{"env": "staging"}))
	first := NewMetrics(params)
	require.NotNil(t, first.connectionCount)

	copy1 := NewMetrics(withMeter(params, root.Scope().Tagged(metrics.Tags{"env": "staging"})))
	copy2 := NewMetrics(withMeter(params, root.Scope().Tagged(metrics.Tags{"env": "staging"})))
	assert.Same(t, first, copy1, "equal tags register into the same place")
	assert.Same(t, first, copy2)

	// Tag order and the number of Tagged hops must not matter.
	a := root.Scope().Tagged(metrics.Tags{"a": "1"}).Tagged(metrics.Tags{"b": "2"})
	b := root.Scope().Tagged(metrics.Tags{"b": "2", "a": "1"})
	assert.Same(t, NewMetrics(withMeter(params, a)), NewMetrics(withMeter(params, b)))

	assert.Zero(t, logs.Len(), "no scope may warn: %v", logs.All())
}

// A different value for the same tag names is a different series and registers
// its own set, as it always did.
func TestMetrics_SameTagNamesWithOtherValuesStaySeparate(t *testing.T) {
	root := metrics.New()
	params, logs := observedParams(root.Scope().Tagged(metrics.Tags{"shard": "1"}))
	one := NewMetrics(params)
	two := NewMetrics(withMeter(params, root.Scope().Tagged(metrics.Tags{"shard": "2"})))
	twoAgain := NewMetrics(withMeter(params, root.Scope().Tagged(metrics.Tags{"shard": "2"})))

	assert.NotSame(t, one, two)
	assert.Same(t, two, twoAgain)
	require.NotNil(t, one.connectionCount)
	require.NotNil(t, two.connectionCount)
	assert.Zero(t, logs.Len())
}

// The registry rejects a metric name registered again with different tag
// names, so a scope tagged differently from the one that registered first can
// never register its own copy. It must get the first set, not warn six times
// and be left with nil handles.
func TestMetrics_DifferentTagNamesShareTheRegisteredFamily(t *testing.T) {
	root := metrics.New()
	params, logs := observedParams(root.Scope())
	first := NewMetrics(params)
	require.NotNil(t, first.connectionCount)

	for name, scope := range map[string]*metrics.Scope{
		"extra tag":      root.Scope().Tagged(metrics.Tags{"env": "staging"}),
		"other tag name": root.Scope().Tagged(metrics.Tags{"zone": "a"}),
		"two tags":       root.Scope().Tagged(metrics.Tags{"env": "staging", "zone": "a"}),
	} {
		got := NewMetrics(withMeter(params, scope))
		assert.Same(t, first, got, name)
		assert.Same(t, got, NewMetrics(withMeter(params, scope)), "%s: a repeat caller gets the same set", name)
	}
	assert.Zero(t, logs.Len(), "no scope may warn: %v", logs.All())

	// The shared handles aggregate every transport's pools.
	r1, r2 := NewReporter(first), NewReporter(NewMetrics(withMeter(params, root.Scope().Tagged(metrics.Tags{"env": "staging"}))))
	r1.SetCounts(1, 0, 0)
	r2.SetCounts(2, 0, 0)
	active, _, _ := reportedGauges(first)
	assert.Equal(t, int64(3), active)
}

func TestMetrics_ScopesOverDifferentRegistriesStaySeparate(t *testing.T) {
	params, logs := observedParams(metrics.New().Scope())
	first := NewMetrics(params)
	second := NewMetrics(withMeter(params, metrics.New().Scope()))
	secondTagged := NewMetrics(withMeter(params, metrics.New().Scope().Tagged(metrics.Tags{"a": "1"})))

	assert.NotSame(t, first, second)
	assert.NotSame(t, second, secondTagged)
	require.NotNil(t, second.connectionCount)
	require.NotNil(t, secondTagged.connectionCount)
	assert.Zero(t, logs.Len(), "each registry registers its own set without warnings")
}

func TestMetrics_ConcurrentEquivalentScopesRegisterOnce(t *testing.T) {
	root := metrics.New()
	params, logs := observedParams(root.Scope())

	const callers = 50
	got := make([]*Metrics, callers)
	var wg sync.WaitGroup
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Half the callers use another tag name, so both the exact and the
			// registry-level sharing paths run concurrently.
			tags := metrics.Tags{"k": "v"}
			if i%2 == 1 {
				tags = metrics.Tags{"other": "v"}
			}
			got[i] = NewMetrics(withMeter(params, root.Scope().Tagged(tags)))
		}()
	}
	wg.Wait()

	for _, m := range got {
		assert.Same(t, got[0], m)
	}
	assert.Zero(t, logs.Len())
}

func withMeter(p MetricsParams, scope *metrics.Scope) MetricsParams {
	p.Meter = scope
	return p
}

// TestScopeIdentity guards the reflection in readScopeIdentity. If a
// go.uber.org/net/metrics upgrade changes the shape of Scope, identityOf
// silently falls back to comparing pointers and separately tagged scopes would
// register twice and warn again; this test fails first.
func TestScopeIdentity(t *testing.T) {
	root, other := metrics.New(), metrics.New()

	id, ok := readScopeIdentity(root.Scope())
	require.True(t, ok, "metrics.Scope no longer has the fields identityOf reads; update readScopeIdentity")
	require.NotZero(t, id.core)

	same := func(a, b *metrics.Scope) bool {
		x, okx := readScopeIdentity(a)
		y, oky := readScopeIdentity(b)
		require.True(t, okx && oky)
		return x == y
	}
	assert.True(t, same(root.Scope(), root.Scope().Tagged(nil)))
	assert.True(t, same(root.Scope().Tagged(metrics.Tags{"a": "1"}), root.Scope().Tagged(metrics.Tags{"a": "1"})))
	assert.False(t, same(root.Scope(), root.Scope().Tagged(metrics.Tags{"a": "1"})))
	assert.False(t, same(root.Scope().Tagged(metrics.Tags{"a": "1"}), root.Scope().Tagged(metrics.Tags{"a": "2"})))
	assert.False(t, same(root.Scope(), other.Scope()))

	// The identity must match the library's own notion of a collision: scopes
	// that compare equal really do reject each other's registrations.
	spec := metrics.Spec{Name: "identity_probe", Help: "probe"}
	_, err := root.Scope().Tagged(metrics.Tags{"a": "1"}).Gauge(spec)
	require.NoError(t, err)
	_, err = root.Scope().Tagged(metrics.Tags{"a": "1"}).Gauge(spec)
	assert.Error(t, err, "equal identity must mean a registration collision")
	_, err = other.Scope().Tagged(metrics.Tags{"a": "1"}).Gauge(spec)
	assert.NoError(t, err, "different registries must not collide")

	// A nil scope has no identity to read and must not panic.
	_, ok = readScopeIdentity(nil)
	assert.False(t, ok)
	assert.NotPanics(t, func() { identityOf(nil) })
}
