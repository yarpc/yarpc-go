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
	// Transport is the value of the "transport" tag, e.g. "grpc" or "http2".
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
	// across all pools.
	dialsTotal *metrics.Counter
	// dialFailuresTotal counts every dial attempt that failed, aggregated
	// across all pools.
	dialFailuresTotal *metrics.Counter
}

// NewMetrics creates the transport-wide connection pool metric handles.
func NewMetrics(p MetricsParams) *Metrics {
	m := &Metrics{}
	if p.Meter == nil {
		return m
	}

	prefix := p.MetricPrefix
	if prefix == "" {
		prefix = "conn_pool"
	}

	tags := metrics.Tags{
		componentTag: componentYarpc,
		serviceTag:   p.ServiceName,
		transportTag: p.Transport,
	}

	var err error
	m.connectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_active_connections",
		Help:      "Number of active connections accepting new streams/requests, aggregated across all pools.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("connpool: failed to create active connections gauge", zap.Error(err))
	}

	m.drainingConnectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_draining_connections",
		Help:      "Number of connections draining in-flight streams/requests, aggregated across all pools.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("connpool: failed to create draining connections gauge", zap.Error(err))
	}

	m.idleConnectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      prefix + "_idle_connections",
		Help:      "Number of idle connections waiting for timeout before closing, aggregated across all pools.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("connpool: failed to create idle connections gauge", zap.Error(err))
	}

	m.scaleUpTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_scale_up_total",
		Help:      "Total number of times a pool opened a new connection, aggregated across all pools.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("connpool: failed to create scale up counter", zap.Error(err))
	}

	m.scaleDownTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_scale_down_total",
		Help:      "Total number of times a pool marked a connection for draining, aggregated across all pools.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("connpool: failed to create scale down counter", zap.Error(err))
	}

	m.idleReactivationTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_idle_reactivation_total",
		Help:      "Total number of times an idle connection was reactivated instead of opening a new one, aggregated across all pools.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("connpool: failed to create idle reactivation counter", zap.Error(err))
	}

	m.dialsTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_dials_total",
		Help:      "Total number of connection dial attempts that succeeded, aggregated across all pools.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("connpool: failed to create dials counter", zap.Error(err))
	}

	m.dialFailuresTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      prefix + "_dial_failures_total",
		Help:      "Total number of connection dial attempts that failed, aggregated across all pools.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("connpool: failed to create dial failures counter", zap.Error(err))
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

// IncDial records that a pool's dial attempt succeeded.
func (r *Reporter) IncDial() {
	if r == nil {
		return
	}
	r.shared.incDial()
}

// IncDialFailure records that a pool's dial attempt failed.
func (r *Reporter) IncDialFailure() {
	if r == nil {
		return
	}
	r.shared.incDialFailure()
}
