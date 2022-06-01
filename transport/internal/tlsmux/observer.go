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
