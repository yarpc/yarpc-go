// Copyright (c) 2025 Uber Technologies, Inc.
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

package metrics

import (
	"crypto/tls"

	"go.uber.org/net/metrics"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/zap"
)

const (
	_serviceTag   = "service"
	_transportTag = "transport"
	_versionTag   = "version"
	_destTag      = "dest"
	_directionTag = "direction"
	_modeTag      = "mode"
	_componentTag = "component"
)

// Params holds parameters needed for creating new observer.
type Params struct {
	Meter         *metrics.Scope
	Logger        *zap.Logger
	ServiceName   string // Name of the service.
	TransportName string // Name of the transport.
	Direction     string // Inbound or Outbound.
	Dest          string // Name of the dest, only for outbound.
	Mode          yarpctls.Mode
}

// Observer holds connection metrics.
type Observer struct {
	plaintextConnectionsCounter *metrics.Counter
	tlsConnectionsCounter       *metrics.CounterVector
	tlsFailuresCounter          *metrics.Counter
}

// NewObserver returns observer for emitting connection metrics.
func NewObserver(p Params) *Observer {
	tags := metrics.Tags{
		_componentTag: "yarpc",
		_serviceTag:   p.ServiceName,
		_transportTag: p.TransportName,
		_modeTag:      p.Mode.String(),
		_directionTag: p.Direction,
		_destTag:      p.Dest,
	}

	plaintextConns, err := p.Meter.Counter(metrics.Spec{
		Name:      "plaintext_connections",
		Help:      "Total number of plaintext connections established.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Error("failed to create plaintext connections counter", zap.Error(err))
	}

	tlsConns, err := p.Meter.CounterVector(metrics.Spec{
		Name:      "tls_connections",
		Help:      "Total number of TLS connections established.",
		ConstTags: tags,
		VarTags:   []string{_versionTag},
	})
	if err != nil {
		p.Logger.Error("failed to create tls connections counter", zap.Error(err))
	}

	tlsHandshakeFailures, err := p.Meter.Counter(metrics.Spec{
		Name:      "tls_handshake_failures",
		Help:      "Total number of TLS handshake failures.",
		ConstTags: tags,
	})
	if err != nil {
		p.Logger.Error("failed to create tls handshake failures counter", zap.Error(err))
	}

	return &Observer{
		tlsConnectionsCounter:       tlsConns,
		tlsFailuresCounter:          tlsHandshakeFailures,
		plaintextConnectionsCounter: plaintextConns,
	}
}

// IncPlaintextConnections increments plaintext connections metric.
func (o *Observer) IncPlaintextConnections() {
	o.plaintextConnectionsCounter.Inc()
}

// IncTLSConnections increments TLS connections metric.
func (o *Observer) IncTLSConnections(version uint16) {
	o.tlsConnectionsCounter.MustGet(_versionTag, tlsVersionString(version)).Inc()
}

// IncTLSHandshakeFailures increments TLS handshake failures metric.
func (o *Observer) IncTLSHandshakeFailures() {
	o.tlsFailuresCounter.Inc()
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "1.0"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS13:
		return "1.3"
	default:
		return "unknown"
	}
}
