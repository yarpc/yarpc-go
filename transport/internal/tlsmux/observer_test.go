package tlsmux

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/zap"
)

func TestTlsVersionString(t *testing.T) {
	assert.Equal(t, "1.0", tlsVersionString(tls.VersionTLS10), "unexpected version for TLS 1.0")
	assert.Equal(t, "1.1", tlsVersionString(tls.VersionTLS11), "unexpected version for TLS 1.1")
	assert.Equal(t, "1.2", tlsVersionString(tls.VersionTLS12), "unexpected version for TLS 1.2")
	assert.Equal(t, "1.3", tlsVersionString(tls.VersionTLS13), "unexpected version for TLS 1.3")
	assert.Equal(t, "unknown", tlsVersionString(20), "unexpected unknown version")
}

func TestObserver(t *testing.T) {
	root := metrics.New()
	observer := newObserver(root.Scope(), zap.NewNop(), "test-svc", "test-transport")
	require.NotNil(t, observer, "unexpected nil observer")
	assert.NotNil(t, observer.plaintextConnectionsCounter, "unexpected nil counter")
	assert.NotNil(t, observer.tlsConnectionsCounter, "unexpected nil counter")
	assert.NotNil(t, observer.tlsFailuresCounter, "unexpected nil counter")

	observer.incPlaintextConnections()
	observer.incTLSConnections(tls.VersionTLS11)
	observer.incTLSHandshakeFailures()

	expectedCounters := []metrics.Snapshot{
		{
			Name:  "plaintext_connections",
			Value: 1,
			Tags: metrics.Tags{
				"service":   "test-svc",
				"transport": "test-transport",
			},
		},
		{
			Name:  "tls_connections",
			Value: 1,
			Tags: metrics.Tags{
				"service":   "test-svc",
				"transport": "test-transport",
				"version":   "1.1",
			},
		},
		{
			Name:  "tls_handshake_failures",
			Value: 1,
			Tags: metrics.Tags{
				"service":   "test-svc",
				"transport": "test-transport",
			},
		},
	}
	assert.Equal(t, expectedCounters, root.Snapshot().Counters)
}
