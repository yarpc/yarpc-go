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
	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

// Tags for the connection pool metrics.
const (
	_componentTag = "component"
	_serviceTag   = "service"
	_transportTag = "transport"
	_peerTag      = "peer"
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
	Peer        string
}

// connPoolMetrics holds metric handles for the gRPC connection pool.
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

// newConnPoolMetrics creates metric handles scoped to the given peer address.
func newConnPoolMetrics(p connPoolMetricsParams) *connPoolMetrics {
	m := &connPoolMetrics{}
	if p.Meter == nil {
		return m
	}

	tags := metrics.Tags{
		_componentTag: _componentTagValueYarpc,
		_serviceTag:   p.ServiceName,
		_transportTag: _transportTagValueGrpc,
		_peerTag:      p.Peer,
	}

	var err error
	m.connectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      "conn_pool_active_connections",
		Help:      "Number of active connections accepting new streams for this peer.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Error("failed to create active connections gauge", zap.Error(err))
	}

	m.drainingConnectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      "conn_pool_draining_connections",
		Help:      "Number of connections draining in-flight streams for this peer.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Error("failed to create draining connections gauge", zap.Error(err))
	}

	m.idleConnectionCount, err = p.Meter.Gauge(metrics.Spec{
		Name:      "conn_pool_idle_connections",
		Help:      "Number of idle connections waiting for timeout before closing for this peer.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Error("failed to create idle connections gauge", zap.Error(err))
	}

	m.scaleUpTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      "conn_pool_scale_up_total",
		Help:      "Total number of times the pool opened a new connection for this peer.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Error("failed to create scale up counter", zap.Error(err))
	}

	m.scaleDownTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      "conn_pool_scale_down_total",
		Help:      "Total number of times the pool marked a connection for draining for this peer.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Error("failed to create scale down counter", zap.Error(err))
	}

	m.idleReactivationTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      "conn_pool_idle_reactivation_total",
		Help:      "Total number of times an idle connection was reactivated instead of opening a new one.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Error("failed to create idle reactivation counter", zap.Error(err))
	}

	return m
}

func (m *connPoolMetrics) setConnectionCount(n int64) {
	if m == nil || m.connectionCount == nil {
		return
	}
	m.connectionCount.Store(n)
}

func (m *connPoolMetrics) setDrainingConnectionCount(n int64) {
	if m == nil || m.drainingConnectionCount == nil {
		return
	}
	m.drainingConnectionCount.Store(n)
}

func (m *connPoolMetrics) setIdleConnectionCount(n int64) {
	if m == nil || m.idleConnectionCount == nil {
		return
	}
	m.idleConnectionCount.Store(n)
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
