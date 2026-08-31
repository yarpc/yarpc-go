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

package http

import (
	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

// Tags for the HTTP/2 per-peer connection metrics.
//
// Note: intentionally NOT tagged by peer (host:port) or connection scope.
// Every httpPeer owns a dedicated *http2.Transport (see peer.go), and
// duplicate-peer identifiers or isolated Dialers (see Dialer.WithConnectionIsolation)
// can create many peers for what is logically one destination. Tagging these
// metrics by peer would mean registering a new metric series per peer, which
// go.uber.org/net/metrics rejects on any second registration with the same
// name and tags: transport/grpc's connection pool metrics hit exactly this
// failure mode when they were briefly tagged by peer address (peer churn -
// i.e. releasing and re-retaining a peer - re-registered the same series and
// errored), and were redesigned to aggregate at the transport level instead.
// These gauges/counters follow that same aggregate design from the start.
const (
	_h2ComponentTag = "component"
	_h2ServiceTag   = "service"
	_h2TransportTag = "transport"
)

// Tag values for the HTTP/2 per-peer connection metrics.
const (
	_h2ComponentTagValueYarpc = "yarpc"
	_h2TransportTagValueHTTP  = "http"
)

// http2ConnectionMetricsParams holds parameters needed for creating the
// HTTP/2 per-peer connection metrics.
type http2ConnectionMetricsParams struct {
	Meter       *metrics.Scope
	Logger      *zap.Logger
	ServiceName string
}

// http2ConnectionMetrics holds metric handles for the HTTP/2 per-peer
// dedicated *http2.Transport feature. A single instance is created once per
// Transport (see (o *transportOptions) newTransport) and shared by every
// peer, so registration happens exactly once regardless of how many peers -
// including duplicate peers and peers from isolated Dialers - are created or
// recreated over the Transport's lifetime.
type http2ConnectionMetrics struct {
	// activePeers is the number of peers currently holding a dedicated
	// *http2.Transport, aggregated across all peers.
	activePeers *metrics.Gauge
	// connectionsDialedTotal counts every HTTP/2 connection dial attempt that
	// succeeded, aggregated across all peers.
	connectionsDialedTotal *metrics.Counter
	// connectionDialFailuresTotal counts every HTTP/2 connection dial attempt
	// that failed, aggregated across all peers.
	connectionDialFailuresTotal *metrics.Counter
}

// newHTTP2ConnectionMetrics creates the transport-wide HTTP/2 per-peer
// connection metric handles.
func newHTTP2ConnectionMetrics(p http2ConnectionMetricsParams) *http2ConnectionMetrics {
	m := &http2ConnectionMetrics{}
	if p.Meter == nil {
		return m
	}

	tags := metrics.Tags{
		_h2ComponentTag: _h2ComponentTagValueYarpc,
		_h2ServiceTag:   p.ServiceName,
		_h2TransportTag: _h2TransportTagValueHTTP,
	}

	var err error
	m.activePeers, err = p.Meter.Gauge(metrics.Spec{
		Name:      "http2_peer_dedicated_transports",
		Help:      "Number of peers with a dedicated HTTP/2 transport, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create http2 active peers gauge", zap.Error(err))
	}

	m.connectionsDialedTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      "http2_connections_dialed_total",
		Help:      "Total number of HTTP/2 connections successfully dialed, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create http2 connections dialed counter", zap.Error(err))
	}

	m.connectionDialFailuresTotal, err = p.Meter.Counter(metrics.Spec{
		Name:      "http2_connection_dial_failures_total",
		Help:      "Total number of failed HTTP/2 connection dial attempts, aggregated across all peers.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Warn("failed to create http2 connection dial failures counter", zap.Error(err))
	}

	return m
}

// incActivePeers records that a peer has acquired its dedicated HTTP/2
// transport.
func (m *http2ConnectionMetrics) incActivePeers() {
	if m == nil {
		return
	}
	m.activePeers.Inc()
}

// decActivePeers records that a peer's dedicated HTTP/2 transport has been
// released.
func (m *http2ConnectionMetrics) decActivePeers() {
	if m == nil || m.activePeers == nil {
		return
	}
	m.activePeers.Dec()
}

// incConnectionsDialed records a successful HTTP/2 connection dial.
func (m *http2ConnectionMetrics) incConnectionsDialed() {
	if m == nil {
		return
	}
	m.connectionsDialedTotal.Inc()
}

// incConnectionDialFailures records a failed HTTP/2 connection dial attempt.
func (m *http2ConnectionMetrics) incConnectionDialFailures() {
	if m == nil {
		return
	}
	m.connectionDialFailuresTotal.Inc()
}
