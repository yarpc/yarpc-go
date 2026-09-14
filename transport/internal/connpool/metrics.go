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
	"reflect"
	"sort"
	"strings"
	"sync"

	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

// Tag names for connection pool metrics.
//
// Note: connection pool metrics are intentionally NOT tagged by peer. Tagging
// by peer (host:port) is unbounded and dynamic -- large fleets and peer churn
// (e.g. rescheduled pods) would explode metric cardinality. The gauges below
// instead hold the aggregate across all of a transport's pools/peers, and the
// counters accumulate pool-wide scaling events.
const (
	componentTag   = "component"
	serviceTag     = "service"
	transportTag   = "transport"
	componentYarpc = "yarpc"
)

// MetricsParams holds parameters needed to create a Metrics instance.
type MetricsParams struct {
	Meter       *metrics.Scope
	Logger      *zap.Logger
	ServiceName string
	// Transport is the value of the "transport" tag, e.g. "http2".
	Transport string
	// MetricPrefix names the metric family, e.g. "conn_pool" or
	// "http2_conn_pool". Combined with a fixed suffix per metric
	// (_active_connections, _scale_up_total, ...).
	MetricPrefix string
}

// Metrics holds metric handles for a connection pool feature. A single
// instance is normally shared by every pool of a transport; each pool
// applies its own deltas via a Reporter so per-pool teardown doesn't require
// re-registering metrics.
type Metrics struct {
	// connectionCount is the number of active connections accepting new
	// streams/requests.
	connectionCount *metrics.Gauge
	// drainingConnectionCount is the number of connections no longer
	// accepting new streams/requests but still completing in-flight ones.
	drainingConnectionCount *metrics.Gauge
	// idleConnectionCount is the number of connections with zero load,
	// waiting for the idle timeout before being closed.
	idleConnectionCount *metrics.Gauge
	// scaleUpTotal counts the number of times a new connection was opened.
	scaleUpTotal *metrics.Counter
	// scaleDownTotal counts the number of times a connection was marked for
	// draining.
	scaleDownTotal *metrics.Counter
	// idleReactivationTotal counts the number of times an idle connection
	// was reactivated rather than opening a new one.
	idleReactivationTotal *metrics.Counter
}

// metricsKey identifies one set of registered metrics. go.uber.org/net/metrics
// rejects a second registration of the same name and tags in a registry, so a
// set is registered once per key and shared by every caller that asks for it.
// Two scopes are the same registration target when they share a registry and
// constant tags, even if they are different *Scope values (Tagged returns a new
// one every call), so the key holds that identity rather than the pointer.
type metricsKey struct {
	scope       scopeIdentity
	prefix      string
	serviceName string
	transport   string
}

// scopeIdentity says which registry and constant tags a *Scope registers into.
type scopeIdentity struct {
	core uintptr // the registry, zero when identityOf could not read it
	tags string  // sorted constant tags, as k=v pairs
	// ptr is set only when identityOf could not read the scope's internals, in
	// which case identity falls back to the *Scope pointer.
	ptr *metrics.Scope
}

// identityOf reads the registry and constant tags of s. go.uber.org/net/metrics
// does not export them, so they are read with reflection; if its Scope no
// longer has the expected shape this falls back to the pointer, which still
// dedupes callers that share one *Scope. TestScopeIdentity fails if that
// happens, so a dependency upgrade cannot silently lose the fix.
func identityOf(s *metrics.Scope) scopeIdentity {
	if id, ok := readScopeIdentity(s); ok {
		return id
	}
	return scopeIdentity{ptr: s}
}

func readScopeIdentity(s *metrics.Scope) (scopeIdentity, bool) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return scopeIdentity{}, false
	}
	e := v.Elem()
	if e.Kind() != reflect.Struct {
		return scopeIdentity{}, false
	}
	core, tags := e.FieldByName("core"), e.FieldByName("constTags")
	if !core.IsValid() || core.Kind() != reflect.Ptr || core.IsNil() ||
		!tags.IsValid() || tags.Kind() != reflect.Map ||
		tags.Type().Key().Kind() != reflect.String || tags.Type().Elem().Kind() != reflect.String {
		return scopeIdentity{}, false
	}
	pairs := make([]string, 0, tags.Len())
	for it := tags.MapRange(); it.Next(); {
		pairs = append(pairs, it.Key().String()+"\x00"+it.Value().String())
	}
	sort.Strings(pairs)
	return scopeIdentity{core: core.Pointer(), tags: strings.Join(pairs, "\x01")}, true
}

// registryKey identifies a metric family within one registry, whatever
// constant tags the scope adds. go.uber.org/net/metrics rejects a metric name
// registered again with a different set of tag names, so a family can only be
// registered once per registry however its callers' scopes are tagged.
type registryKey struct {
	core        uintptr
	ptr         *metrics.Scope
	prefix      string
	serviceName string
	transport   string
}

// sharedMetric is one registered set. scope is kept so the registry whose
// address is in the key stays alive; a freed registry's address could
// otherwise be reused by another one and match this entry by mistake.
type sharedMetric struct {
	metrics *Metrics
	scope   *metrics.Scope
}

// sharedMetrics holds the Metrics registered so far. Entries live for the life
// of the process, like the Scope each one is registered on.
var sharedMetrics = struct {
	sync.Mutex
	byKey      map[metricsKey]sharedMetric  // exactly this registry, tags and family
	byRegistry map[registryKey]sharedMetric // first set registered in this registry
}{
	byKey:      make(map[metricsKey]sharedMetric),
	byRegistry: make(map[registryKey]sharedMetric),
}

func metricPrefix(p MetricsParams) string {
	if p.MetricPrefix == "" {
		return "conn_pool"
	}
	return p.MetricPrefix
}

// NewMetrics returns the connection pool metric handles for the given
// registry, metric prefix, service name and transport, registering them the
// first time they are asked for.
//
// A process may build several transports on the same meter and service name
// (for example one per dispatcher). Each used to register the same metrics, so
// every transport after the first failed registration, logged a warning per
// metric, and was left with nil handles that reported nothing. Later callers
// now share the handles the first one registered:
//
//   - a scope with the same registry and constant tags gets them directly,
//     even when it is a different *Scope value (Tagged returns a new one on
//     every call);
//   - a scope whose tags make it a different series (same tag names, other
//     values) registers its own set, as before;
//   - a scope that cannot register because the registry already holds the
//     family under different tag names gets the first set registered there,
//     since that family can only exist once per registry. Its tags do not apply.
//
// Pools report through a Reporter that applies deltas, so sharing the handles
// makes the gauges the aggregate across all of those transports' pools, which
// is what they are documented to be.
func NewMetrics(p MetricsParams) *Metrics {
	if p.Meter == nil {
		return &Metrics{}
	}
	id := identityOf(p.Meter)
	key := metricsKey{scope: id, prefix: metricPrefix(p), serviceName: p.ServiceName, transport: p.Transport}
	reg := registryKey{core: id.core, ptr: id.ptr, prefix: key.prefix, serviceName: p.ServiceName, transport: p.Transport}

	sharedMetrics.Lock()
	defer sharedMetrics.Unlock()
	if e, ok := sharedMetrics.byKey[key]; ok {
		return e.metrics
	}

	// Register quietly: if nothing can be registered because the registry
	// already has this family, share it instead of warning.
	type failure struct {
		msg string
		err error
	}
	var failures []failure
	m := registerMetrics(p, func(msg string, err error) { failures = append(failures, failure{msg, err}) })
	if len(failures) == metricsCount {
		if e, ok := sharedMetrics.byRegistry[reg]; ok {
			sharedMetrics.byKey[key] = e
			return e.metrics
		}
	}
	for _, f := range failures {
		if p.Logger != nil {
			p.Logger.Warn(f.msg, zap.Error(f.err))
		}
	}

	e := sharedMetric{metrics: m, scope: p.Meter}
	sharedMetrics.byKey[key] = e
	if len(failures) < metricsCount {
		if _, ok := sharedMetrics.byRegistry[reg]; !ok {
			sharedMetrics.byRegistry[reg] = e
		}
	}
	return m
}

// newMetrics registers a new set of metric handles on p.Meter. A registration
// that fails is logged and leaves that handle nil, which every Metrics method
// tolerates.
func newMetrics(p MetricsParams) *Metrics {
	return registerMetrics(p, func(msg string, err error) {
		if p.Logger != nil {
			p.Logger.Warn(msg, zap.Error(err))
		}
	})
}

// metricsCount is how many handles registerMetrics registers.
const metricsCount = 6

// registerMetrics registers the metric handles on p.Meter, calling warn for
// each one that fails to register and leaving that handle nil.
func registerMetrics(p MetricsParams, warn func(msg string, err error)) *Metrics {
	m := &Metrics{}
	prefix := metricPrefix(p)

	tags := metrics.Tags{
		componentTag: componentYarpc,
		serviceTag:   p.ServiceName,
		transportTag: p.Transport,
	}

	var err error
	m.connectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_active_connections",
		Help:      "Number of active connections accepting new streams, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create active connections gauge", err)
	}

	m.drainingConnectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_draining_connections",
		Help:      "Number of connections draining in-flight streams, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create draining connections gauge", err)
	}

	m.idleConnectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_idle_connections",
		Help:      "Number of idle connections waiting for timeout before closing, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create idle connections gauge", err)
	}

	m.scaleUpTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_scale_up_total",
		Help:      "Total number of times the pool opened a new connection, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create scale up counter", err)
	}

	m.scaleDownTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_scale_down_total",
		Help:      "Total number of times the pool marked a connection for draining, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create scale down counter", err)
	}

	m.idleReactivationTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_idle_reactivation_total",
		Help:      "Total number of times an idle connection was reactivated instead of opening a new one, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create idle reactivation counter", err)
	}

	return m
}

func (m *Metrics) addConnectionCount(delta int64) {
	if m == nil || m.connectionCount == nil || delta == 0 {
		return
	}
	m.connectionCount.Add(delta)
}

func (m *Metrics) addDrainingConnectionCount(delta int64) {
	if m == nil || m.drainingConnectionCount == nil || delta == 0 {
		return
	}
	m.drainingConnectionCount.Add(delta)
}

func (m *Metrics) addIdleConnectionCount(delta int64) {
	if m == nil || m.idleConnectionCount == nil || delta == 0 {
		return
	}
	m.idleConnectionCount.Add(delta)
}

func (m *Metrics) incScaleUp() {
	if m == nil || m.scaleUpTotal == nil {
		return
	}
	m.scaleUpTotal.Inc()
}

func (m *Metrics) incScaleDown() {
	if m == nil || m.scaleDownTotal == nil {
		return
	}
	m.scaleDownTotal.Inc()
}

func (m *Metrics) incIdleReactivation() {
	if m == nil || m.idleReactivationTotal == nil {
		return
	}
	m.idleReactivationTotal.Inc()
}

// Reporter applies a single pool's connection-state counts to the shared,
// transport-wide gauges of a Metrics. Because those gauges are not tagged
// per-pool, every pool contributes to the same series; Reporter tracks the
// values it last published and applies the difference, so the gauges always
// reflect the sum across all pools and a pool's teardown can cleanly
// withdraw its contribution via SetCounts(0, 0, 0).
type Reporter struct {
	shared *Metrics

	mu           sync.Mutex
	lastActive   int64
	lastDraining int64
	lastIdle     int64
}

// NewReporter creates a Reporter that feeds the shared Metrics.
func NewReporter(shared *Metrics) *Reporter {
	return &Reporter{shared: shared}
}

// SetCounts publishes this pool's current active/draining/idle connection
// counts, applying the difference from the previously reported values to
// the shared aggregate gauges. Safe for concurrent use.
func (r *Reporter) SetCounts(active, draining, idle int64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	dActive := active - r.lastActive
	dDraining := draining - r.lastDraining
	dIdle := idle - r.lastIdle
	r.lastActive = active
	r.lastDraining = draining
	r.lastIdle = idle
	r.mu.Unlock()

	r.shared.addConnectionCount(dActive)
	r.shared.addDrainingConnectionCount(dDraining)
	r.shared.addIdleConnectionCount(dIdle)
}

// IncScaleUp records that a pool opened a new connection.
func (r *Reporter) IncScaleUp() {
	if r == nil {
		return
	}
	r.shared.incScaleUp()
}

// IncScaleDown records that a pool marked a connection for draining.
func (r *Reporter) IncScaleDown() {
	if r == nil {
		return
	}
	r.shared.incScaleDown()
}

// IncIdleReactivation records that a pool reactivated an idle connection
// instead of dialing a new one.
func (r *Reporter) IncIdleReactivation() {
	if r == nil {
		return
	}
	r.shared.incIdleReactivation()
}
