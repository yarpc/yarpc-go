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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
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
	observer := newObserver(root.Scope(), zap.NewNop(), "test-svc", "test-transport", yarpctls.Enforced)
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
				"component": "yarpc",
				"mode":      "enforced",
			},
		},
		{
			Name:  "tls_connections",
			Value: 1,
			Tags: metrics.Tags{
				"service":   "test-svc",
				"transport": "test-transport",
				"version":   "1.1",
				"component": "yarpc",
				"mode":      "enforced",
			},
		},
		{
			Name:  "tls_handshake_failures",
			Value: 1,
			Tags: metrics.Tags{
				"service":   "test-svc",
				"transport": "test-transport",
				"component": "yarpc",
				"mode":      "enforced",
			},
		},
	}
	assert.Equal(t, expectedCounters, root.Snapshot().Counters)
}
