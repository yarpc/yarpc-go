// Copyright (c) 2022 Uber Technologies, Inc.
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

package tlsmux

import (
	"crypto/tls"

	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

const (
	_serviceTag   = "service"
	_transportTag = "transport"
	_versionTag   = "version"
)

type observer struct {
	plaintextConnectionsCounter *metrics.Counter
	tlsConnectionsCounter       *metrics.CounterVector
	tlsFailuresCounter          *metrics.Counter
}

func newObserver(meter *metrics.Scope, logger *zap.Logger, serviceName, transportName string) *observer {
	tags := metrics.Tags{
		"service":   serviceName,
		"transport": transportName,
		"component": "yarpc",
	}

	plaintextConns, err := meter.Counter(metrics.Spec{
		Name:      "plaintext_connections",
		Help:      "Total number of plaintext connections established.",
		ConstTags: tags,
	})
	if err != nil {
		logger.Error("Failed to create plaintext connections counter", zap.Error(err))
	}

	tlsConns, err := meter.CounterVector(metrics.Spec{
		Name:      "tls_connections",
		Help:      "Total number of TLS connections established.",
		ConstTags: tags,
		VarTags:   []string{_versionTag},
	})
	if err != nil {
		logger.Error("Failed to create plaintext connections counter", zap.Error(err))
	}

	tlsHandshakeFailures, err := meter.Counter(metrics.Spec{
		Name:      "tls_handshake_failures",
		Help:      "Total number of TLS handshake failures.",
		ConstTags: tags,
	})
	if err != nil {
		logger.Error("Failed to create plaintext connections counter", zap.Error(err))
	}

	return &observer{
		tlsConnectionsCounter:       tlsConns,
		tlsFailuresCounter:          tlsHandshakeFailures,
		plaintextConnectionsCounter: plaintextConns,
	}
}

func (o *observer) incPlaintextConnections() {
	o.plaintextConnectionsCounter.Inc()
}

func (o *observer) incTLSConnections(version uint16) {
	o.tlsConnectionsCounter.MustGet(_versionTag, tlsVersionString(version)).Inc()
}

func (o *observer) incTLSHandshakeFailures() {
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
