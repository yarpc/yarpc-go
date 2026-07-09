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

	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

// Tags for the connection pool metrics.
//
// Note: connection pool metrics are intentionally NOT tagged by peer. Tagging
// by peer (host:port) is unbounded and dynamic — large fleets and peer churn
// (e.g. rescheduled pods) would explode metric cardinality, inflate TSDB cost,
// and risk the aggregation tier rate-limiting or dropping the emitter. The
// gauges below instead hold the aggregate across all of a transport's peers,
// and the counters accumulate pool-wide scaling events, which is sufficient to
// tell whether dynamic scaling is behaving correctly at the service level.
const (
	_componentTag = "component"
	_serviceTag   = "service"
	_transportTag = "transport"
)

// Tag values for the connection pool metrics.
const (
	_componentTagValueYarpc = "yarpc"
	_transportTagValueGrpc  = "grpc"
)

// connPoolMetricsParams holds parameters needed for creating connection pool metrics.
type connPoolMetricsParams struct {
	Meter       *metrics.Scope
	Logger      *zap.Logger
	ServiceName string
}

// connPoolMetrics holds metric handles for the gRPC connection pool. A single
// instance is shared by all peers of a transport. Gauges hold the aggregate
// connection counts across peers (each peer applies deltas via a
// peerPoolReporter) and counters accumulate pool-wide scaling events.
type connPoolMetrics struct {
	// connectionCount is the number of active connections accepting new streams.
	connectionCount *metrics.Gauge
	// drainingConnectionCount is the number of connections that are no longer
	// accepting new streams but still have in-flight streams completing.
	drainingConnectionCount *metrics.Gauge
	// idleConnectionCount is the number of connections with zero streams,
	// waiting for the idle timeout before being closed.
	idleConnectionCount *metrics.Gauge
	// scaleUpTotal counts the number of times a new connection was opened.
	scaleUpTotal *metrics.Counter
	// scaleDownTotal counts the number of times a connection was marked for draining.
	scaleDownTotal *metrics.Counter
	// idleReactivationTotal counts the number of times a draining connection
	// was reactivated rather than opening a new one.
	idleReactivationTotal *metrics.Counter
}

// newConnPoolMetrics creates the transport-wide connection pool metric handles.
func newConnPoolMetrics(p connPoolMetricsParams) *connPoolMetrics {
	m := &connPoolMetrics{}
	if p.Meter == nil {
		return m
	}

	tags := metrics.Tags{
		_componentTag: _componentTagValueYarpc,
		_serviceTag:   p.ServiceName,
		_transportTag: _transportTagValueGrpc,
	}

	var err error
	m.connectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      "conn_pool_active_connections",
		Help:      "Number of active connections accepting new streams, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create active connections gauge", zap.Error(err))
	}

	m.drainingConnectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      "conn_pool_draining_connections",
		Help:      "Number of connections draining in-flight streams, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create draining connections gauge", zap.Error(err))
	}

	m.idleConnectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      "conn_pool_idle_connections",
		Help:      "Number of idle connections waiting for timeout before closing, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create idle connections gauge", zap.Error(err))
	}

	m.scaleUpTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      "conn_pool_scale_up_total",
		Help:      "Total number of times the pool opened a new connection, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create scale up counter", zap.Error(err))
	}

	m.scaleDownTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      "conn_pool_scale_down_total",
		Help:      "Total number of times the pool marked a connection for draining, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create scale down counter", zap.Error(err))
	}

	m.idleReactivationTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      "conn_pool_idle_reactivation_total",
		Help:      "Total number of times an idle connection was reactivated instead of opening a new one, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create idle reactivation counter", zap.Error(err))
	}

	return m
}

// addConnectionCount applies a delta to the aggregate active-connections gauge.
func (m *connPoolMetrics) addConnectionCount(delta int64) {
	if m == nil || m.connectionCount == nil || delta == 0 {
		return
	}
	m.connectionCount.Add(delta)
}

// addDrainingConnectionCount applies a delta to the aggregate draining gauge.
func (m *connPoolMetrics) addDrainingConnectionCount(delta int64) {
	if m == nil || m.drainingConnectionCount == nil || delta == 0 {
		return
	}
	m.drainingConnectionCount.Add(delta)
}

// addIdleConnectionCount applies a delta to the aggregate idle gauge.
func (m *connPoolMetrics) addIdleConnectionCount(delta int64) {
	if m == nil || m.idleConnectionCount == nil || delta == 0 {
		return
	}
	m.idleConnectionCount.Add(delta)
}

func (m *connPoolMetrics) incScaleUp() {
	if m == nil || m.scaleUpTotal == nil {
		return
	}
	m.scaleUpTotal.Inc()
}

func (m *connPoolMetrics) incScaleDown() {
	if m == nil || m.scaleDownTotal == nil {
		return
	}
	m.scaleDownTotal.Inc()
}

func (m *connPoolMetrics) incIdleReactivation() {
	if m == nil || m.idleReactivationTotal == nil {
		return
	}
	m.idleReactivationTotal.Inc()
}

// peerPoolReporter applies a single peer's connection-state counts to the
// shared, transport-wide gauges. Because the gauges are not tagged by peer,
// every peer contributes to the same series; this reporter tracks the values it
// last published and applies the difference so the gauges always reflect the
// sum across all peers. During normal operation, deltas telescope as
// connections are added or removed. On peer shutdown, pool goroutines are
// drained and setCounts(0, 0, 0) withdraws this peer's last published
// contribution so a late refreshPoolMetrics snapshot cannot leave residual
// values on the shared gauges. Counter events are pool-wide and forwarded directly.
type peerPoolReporter struct {
	shared *connPoolMetrics

	mu           sync.Mutex
	lastActive   int64
	lastDraining int64
	lastIdle     int64
}

// newPeerPoolReporter creates a reporter that feeds the shared transport metrics.
func newPeerPoolReporter(shared *connPoolMetrics) *peerPoolReporter {
	return &peerPoolReporter{shared: shared}
}

// setCounts publishes this peer's current active/draining/idle connection
// counts, applying the difference from the previously reported values to the
// shared aggregate gauges. Safe for concurrent use.
func (r *peerPoolReporter) setCounts(active, draining, idle int64) {
	if r == nil {
		return
	}
	// Compute deltas under the lock so concurrent callers for the same peer
	// chain their updates consistently. The gauge writes themselves are atomic
	// and commutative, so they can happen outside the lock.
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

func (r *peerPoolReporter) incScaleUp() {
	if r == nil {
		return
	}
	r.shared.incScaleUp()
}

func (r *peerPoolReporter) incScaleDown() {
	if r == nil {
		return
	}
	r.shared.incScaleDown()
}

func (r *peerPoolReporter) incIdleReactivation() {
	if r == nil {
		return
	}
	r.shared.incIdleReactivation()
}
