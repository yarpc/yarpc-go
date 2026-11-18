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
	"fmt"
	"reflect"
	"sort"
	"strconv"
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
	bandTag        = "band"
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
	// dialsTotal counts every dial attempt that succeeded, aggregated
	// across all peers.
	dialsTotal *metrics.Counter
	// dialFailuresTotal counts every dial attempt that failed, aggregated
	// across all peers.
	dialFailuresTotal *metrics.Counter
	// closedConnectionsTotal counts connections the pool closed after they sat
	// idle for IdleTimeout, aggregated across all peers. Unlike scaleDownTotal,
	// which counts connections marked for draining (and those can be
	// reactivated), this says how many connections actually went away.
	closedConnectionsTotal *metrics.Counter
	// inFlightRequests is the pool's own count of requests handed to its
	// connections that have not finished, summed across all connections of all
	// peers. It includes requests still waiting for a free stream on a
	// connection that is at the peer's stream limit, so it can exceed what the
	// peer is actually serving. It is the count the scaling decisions use.
	inFlightRequests *metrics.Gauge
	// requestsTotal counts every request handed to a pooled connection,
	// aggregated across all peers.
	requestsTotal *metrics.Counter
	// scaleUpBlockedTotal counts scale-up attempts that were skipped because
	// the pool was already at MaxConnections.
	scaleUpBlockedTotal *metrics.Counter
	// connectionsByInFlight holds, per band of connectionLoadBuckets, how many
	// active connections currently carry that many in-flight requests. It is
	// sampled by the scaling monitor. Index i is the band labelled
	// bandLabels(connectionLoadBuckets)[i]; the last band is the overflow.
	connectionsByInFlight []*metrics.Gauge
	// connectionsByUtilization is connectionsByInFlight for in-flight requests
	// as a percentage of MaxConcurrentStreams, banded by
	// connectionInFlightUtilizationBuckets. Above 100 means requests are queued.
	connectionsByUtilization []*metrics.Gauge

	// The metrics below are real HTTP/2 stream counts read from the
	// connections themselves, so unlike the in-flight ones they never exceed
	// what the peer allows. They are reported only by transports whose
	// connections can say how many streams are open (see Pool.StreamState),
	// which today is HTTP/2 only: grpc-go does not expose this, so gRPC reports
	// the in-flight metrics alone. They are sampled by the scaling monitor, so
	// they lag by up to its interval.

	// activeStreams is the number of streams open on the wire, summed across
	// all connections of all peers.
	activeStreams *metrics.Gauge
	// pendingRequests is the number of requests waiting for a free stream
	// because their connection is at the peer's stream limit, summed across
	// all connections of all peers.
	pendingRequests *metrics.Gauge
	// connectionsByStreams holds, per band of connectionStreamsBuckets, how
	// many active connections currently have that many streams open, sampled
	// by the scaling monitor.
	connectionsByStreams []*metrics.Gauge
}

// Band bounds. Each bound is the inclusive upper edge of a band, and anything
// above the last bound falls into a final overflow band.
var (
	// connectionLoadBuckets spans an idle connection up to the common
	// HTTP/2 SETTINGS_MAX_CONCURRENT_STREAMS values (100, 250). The 500
	// bucket is for in-flight requests, which can pass the stream limit while
	// they queue; open streams never do.
	connectionLoadBuckets = []int64{0, 1, 2, 5, 10, 25, 50, 100, 150, 200, 250, 500}
	// connectionStreamsBuckets are for open streams, which stop at the peer's
	// limit, so there is no bucket past the largest common limit.
	connectionStreamsBuckets = []int64{0, 1, 2, 5, 10, 25, 50, 100, 150, 200, 250}
	// connectionInFlightUtilizationBuckets are percentages of MaxConcurrentStreams,
	// dense around the default scale-up threshold. The overflow band holds
	// connections over their configured limit.
	connectionInFlightUtilizationBuckets = []int64{0, 10, 25, 50, 70, 80, 90, 100}
)

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
const metricsCount = 17

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

	m.dialsTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_dials_total",
		Help:      "Total number of connection dial attempts that succeeded, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create dials counter", err)
	}

	m.dialFailuresTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_dial_failures_total",
		Help:      "Total number of connection dial attempts that failed, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create dial failures counter", err)
	}

	m.closedConnectionsTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_closed_connections_total",
		Help:      "Total number of connections the pool closed after they sat idle, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create closed connections counter", err)
	}

	m.inFlightRequests, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_in_flight_requests",
		Help:      "Number of requests handed to pooled connections that have not finished, including any queued behind the stream limit, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create in-flight requests gauge", err)
	}

	m.requestsTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_requests_total",
		Help:      "Total number of requests handed to pooled connections, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create requests counter", err)
	}

	m.scaleUpBlockedTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_scale_up_blocked_total",
		Help:      "Total number of scale-up attempts skipped because the pool was at its maximum connections, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create scale up blocked counter", err)
	}

	m.connectionsByInFlight, err = registerBandGauges(p.Meter, metrics.Spec{
		Name:      prefix + "_connections_by_in_flight_requests",
		Help:      "Number of active connections per band of in-flight requests (including any queued behind the stream limit), sampled by the scaling monitor across all peers. The band tag is a whole-number range such as 3-5.",
		ConstTags: tags,
	}, connectionLoadBuckets)
	if err != nil {
		warn("failed to create connections by in-flight requests gauges", err)
	}

	m.connectionsByUtilization, err = registerBandGauges(p.Meter, metrics.Spec{
		Name:      prefix + "_connections_by_in_flight_utilization_percent",
		Help:      "Number of active connections per band of in-flight requests as a percentage of the maximum concurrent streams (over-100 means requests are queued), sampled by the scaling monitor across all peers. The band tag is a whole-number percentage range such as 51-70.",
		ConstTags: tags,
	}, connectionInFlightUtilizationBuckets)
	if err != nil {
		warn("failed to create connections by in-flight utilization gauges", err)
	}

	m.activeStreams, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_active_streams",
		Help:      "Number of streams open on pooled HTTP/2 connections, aggregated across all peers and sampled by the scaling monitor. HTTP/2 only; not reported for gRPC.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create active streams gauge", err)
	}

	m.pendingRequests, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_pending_requests",
		Help:      "Number of requests waiting for a free stream on pooled HTTP/2 connections at the peer's stream limit, aggregated across all peers and sampled by the scaling monitor. HTTP/2 only; not reported for gRPC.",
		ConstTags: tags,
	})
	if err != nil {
		warn("failed to create pending requests gauge", err)
	}

	m.connectionsByStreams, err = registerBandGauges(p.Meter, metrics.Spec{
		Name:      prefix + "_connections_by_streams",
		Help:      "Number of active HTTP/2 connections per band of open streams, sampled by the scaling monitor across all peers. The band tag is a whole-number range such as 3-5. HTTP/2 only; not reported for gRPC.",
		ConstTags: tags,
	}, connectionStreamsBuckets)
	if err != nil {
		warn("failed to create connections by streams gauges", err)
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

func (m *Metrics) incDial() {
	if m == nil || m.dialsTotal == nil {
		return
	}
	m.dialsTotal.Inc()
}

func (m *Metrics) incDialFailure() {
	if m == nil || m.dialFailuresTotal == nil {
		return
	}
	m.dialFailuresTotal.Inc()
}

func (m *Metrics) incClosed() {
	if m == nil || m.closedConnectionsTotal == nil {
		return
	}
	m.closedConnectionsTotal.Inc()
}

func (m *Metrics) addInFlightRequests(delta int64) {
	if m == nil || m.inFlightRequests == nil || delta == 0 {
		return
	}
	m.inFlightRequests.Add(delta)
}

func (m *Metrics) incRequests() {
	if m == nil || m.requestsTotal == nil {
		return
	}
	m.requestsTotal.Inc()
}

func (m *Metrics) incScaleUpBlocked() {
	if m == nil || m.scaleUpBlockedTotal == nil {
		return
	}
	m.scaleUpBlockedTotal.Inc()
}

func (m *Metrics) addActiveStreams(delta int64) {
	if m == nil || m.activeStreams == nil || delta == 0 {
		return
	}
	m.activeStreams.Add(delta)
}

func (m *Metrics) addPendingRequests(delta int64) {
	if m == nil || m.pendingRequests == nil || delta == 0 {
		return
	}
	m.pendingRequests.Add(delta)
}

// bandLabels names the bands of bounds as whole-number ranges: "0", "1",
// "3-5", ..., and "over-<last>" for the overflow band. Values are integers, so
// a band's lower edge is one above the previous bound.
func bandLabels(bounds []int64) []string {
	labels := make([]string, 0, len(bounds)+1)
	for i, hi := range bounds {
		lo := int64(0)
		if i > 0 {
			lo = bounds[i-1] + 1
		}
		if lo >= hi {
			labels = append(labels, strconv.FormatInt(hi, 10))
		} else {
			labels = append(labels, fmt.Sprintf("%d-%d", lo, hi))
		}
	}
	return append(labels, "over-"+strconv.FormatInt(bounds[len(bounds)-1], 10))
}

// bandIndex returns the band of bounds that value falls in; len(bounds) is the
// overflow band.
func bandIndex(bounds []int64, value int64) int {
	return sort.Search(len(bounds), func(i int) bool { return value <= bounds[i] })
}

// registerBandGauges registers one gauge per band of bounds under spec, tagged
// "band". Every band is registered up front so each reports 0 until used.
func registerBandGauges(meter *metrics.Scope, spec metrics.Spec, bounds []int64) ([]*metrics.Gauge, error) {
	spec.VarTags = []string{bandTag}
	vec, err := meter.GaugeVector(spec)
	if err != nil {
		return nil, err
	}
	labels := bandLabels(bounds)
	gauges := make([]*metrics.Gauge, len(labels))
	for i, label := range labels {
		if gauges[i], err = vec.Get(bandTag, label); err != nil {
			return nil, err
		}
	}
	return gauges, nil
}

// addBands applies per-band deltas to a set of band gauges.
func (m *Metrics) addBands(gauges []*metrics.Gauge, deltas []int64) {
	if m == nil {
		return
	}
	for i, d := range deltas {
		if d != 0 && i < len(gauges) && gauges[i] != nil {
			gauges[i].Add(d)
		}
	}
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
	lastStreams  int64
	lastPending  int64
	// lastInFlightBands and lastUtilizationBands are the per-band connection
	// counts this pool last published.
	lastInFlightBands    []int64
	lastUtilizationBands []int64
	lastStreamBands      []int64
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

// IncDial records that a pool's dial attempt succeeded.
func (r *Reporter) IncDial() {
	if r == nil {
		return
	}
	r.shared.incDial()
}

// IncClosed records that a pool closed a connection after it sat idle.
func (r *Reporter) IncClosed() {
	if r == nil {
		return
	}
	r.shared.incClosed()
}

// IncScaleUpBlocked records that a pool needed another connection but was at
// MaxConnections.
func (r *Reporter) IncScaleUpBlocked() {
	if r == nil {
		return
	}
	r.shared.incScaleUpBlocked()
}

// streamStarted records a stream starting on one of the pool's connections.
func (r *Reporter) streamStarted() {
	if r == nil {
		return
	}
	r.shared.addInFlightRequests(1)
	r.shared.incRequests()
}

// streamFinished records a stream ending on one of the pool's connections.
func (r *Reporter) streamFinished() {
	if r == nil {
		return
	}
	r.shared.addInFlightRequests(-1)
}

// SetLoadBands publishes how many of this pool's active connections fall in
// each in-flight and utilization band (see LoadBands), applying the difference
// from the previously reported counts to the shared gauges, as SetCounts does
// for connection counts. Nil counts withdraw the pool's contribution. Safe for
// concurrent use.
func (r *Reporter) SetLoadBands(inFlight, utilization []int64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	dInFlight := bandDeltas(&r.lastInFlightBands, inFlight)
	dUtilization := bandDeltas(&r.lastUtilizationBands, utilization)
	r.mu.Unlock()

	r.shared.addBands(r.shared.connectionsByInFlight, dInFlight)
	r.shared.addBands(r.shared.connectionsByUtilization, dUtilization)
}

// LoadBands bands each active connection's in-flight request count, and that
// count as a percentage of maxStreams, for SetLoadBands. Utilization is left at
// zero connections when maxStreams is not positive.
func LoadBands(inFlight []int32, maxStreams int32) (byInFlight, byUtilization []int64) {
	byInFlight = make([]int64, len(connectionLoadBuckets)+1)
	byUtilization = make([]int64, len(connectionInFlightUtilizationBuckets)+1)
	for _, n := range inFlight {
		byInFlight[bandIndex(connectionLoadBuckets, int64(n))]++
		if maxStreams > 0 {
			byUtilization[bandIndex(connectionInFlightUtilizationBuckets, int64(n)*100/int64(maxStreams))]++
		}
	}
	return byInFlight, byUtilization
}

// bandDeltas returns now minus *last for every band and stores now as the new
// *last. Bands missing from now count as zero.
func bandDeltas(last *[]int64, now []int64) []int64 {
	n := max(len(*last), len(now))
	deltas := make([]int64, n)
	next := make([]int64, n)
	for i := range n {
		if i < len(now) {
			next[i] = now[i]
		}
		deltas[i] = next[i]
		if i < len(*last) {
			deltas[i] -= (*last)[i]
		}
	}
	*last = next
	return deltas
}

// SetStreamBands publishes how many of this pool's active connections fall in
// each open-stream band (see StreamBands), as SetLoadBands does for in-flight
// load. Nil counts withdraw the pool's contribution. Safe for concurrent use.
func (r *Reporter) SetStreamBands(streams []int64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	d := bandDeltas(&r.lastStreamBands, streams)
	r.mu.Unlock()

	r.shared.addBands(r.shared.connectionsByStreams, d)
}

// StreamBands bands each active connection's open stream count for
// SetStreamBands.
func StreamBands(open []int) []int64 {
	counts := make([]int64, len(connectionStreamsBuckets)+1)
	for _, n := range open {
		counts[bandIndex(connectionStreamsBuckets, int64(n))]++
	}
	return counts
}

// SetStreams publishes this pool's current open stream and pending request
// counts, applying the difference from the previously reported values to the
// shared aggregate gauges, as SetCounts does for connection counts. Safe for
// concurrent use.
func (r *Reporter) SetStreams(active, pending int64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	dActive := active - r.lastStreams
	dPending := pending - r.lastPending
	r.lastStreams = active
	r.lastPending = pending
	r.mu.Unlock()

	r.shared.addActiveStreams(dActive)
	r.shared.addPendingRequests(dPending)
}

// IncDialFailure records that a pool's dial attempt failed.
func (r *Reporter) IncDialFailure() {
	if r == nil {
		return
	}
	r.shared.incDialFailure()
}
