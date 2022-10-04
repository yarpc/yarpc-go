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

package dialer

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/transport/internal/tls/testscenario"
	"go.uber.org/zap"
)

func TestDialer(t *testing.T) {
	tests := []struct {
		desc                string
		withCustomDialer    bool
		shouldFailHandshake bool
		data                string
	}{
		{desc: "without_custom_dialer", data: "test_no_dialer"},
		{desc: "with_custom_dialer", data: "test_with_dialer", withCustomDialer: true},
		{desc: "with_handshake_failure", shouldFailHandshake: true},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			root := metrics.New()
			serverTLSConfig, clientTLSConfig := tlsConfigs(t)
			lis, err := net.Listen("tcp", "localhost:0")
			require.NoError(t, err)
			var wg sync.WaitGroup
			defer wg.Wait()
			defer lis.Close()
			wg.Add(1)
			go func() {
				defer wg.Done()
				conn, err := lis.Accept()
				require.NoError(t, err)
				if tt.shouldFailHandshake {
					conn.Close()
					return
				}

				defer conn.Close()
				tlsConn := tls.Server(conn, serverTLSConfig)

				buf := make([]byte, len(tt.data))
				n, err := tlsConn.Read(buf)
				require.NoError(t, err)
				_, err = tlsConn.Write(buf[:n])
				assert.NoError(t, err)
			}()

			params := Params{
				Config:        clientTLSConfig,
				Meter:         root.Scope(),
				Logger:        zap.NewNop(),
				ServiceName:   "test-svc",
				TransportName: "test-transport",
				Dest:          "test-dest",
			}
			// used for assertion whether passed custom dialer is used.
			var customDialerInvoked bool
			if tt.withCustomDialer {
				params.Dialer = func(ctx context.Context, network, address string) (net.Conn, error) {
					customDialerInvoked = true
					return (&net.Dialer{}).DialContext(ctx, network, address)
				}
			}
			dialer := NewTLSDialer(params)
			conn, err := dialer.DialContext(context.Background(), "tcp", lis.Addr().String())
			if tt.shouldFailHandshake {
				require.Error(t, err)
				assertMetrics(t, root, true)
				return
			}

			require.NoError(t, err)
			_, ok := conn.(*tls.Conn)
			assert.True(t, ok)

			n, err := conn.Write([]byte(tt.data))
			require.NoError(t, err)
			assert.Len(t, tt.data, n)

			buf := make([]byte, len(tt.data))
			_, err = conn.Read(buf)
			require.NoError(t, err)
			assert.Equal(t, tt.data, string(buf))
			assertMetrics(t, root, false)
			if tt.withCustomDialer {
				assert.True(t, customDialerInvoked)
			}
		})
	}
}

func assertMetrics(t *testing.T, root *metrics.Root, handshakeFailure bool) {
	expectedCounter := metrics.Snapshot{
		Tags: metrics.Tags{
			"service":   "test-svc",
			"transport": "test-transport",
			"component": "yarpc",
			"mode":      "Enforced",
			"direction": "outbound",
			"dest":      "test-dest",
		},
		Value: 1,
	}
	if handshakeFailure {
		expectedCounter.Name = "tls_handshake_failures"
	} else {
		expectedCounter.Tags["version"] = "1.3"
		expectedCounter.Name = "tls_connections"
	}
	assert.Contains(t, root.Snapshot().Counters, expectedCounter)
}

func tlsConfigs(t *testing.T) (serverConfig *tls.Config, clientConfig *tls.Config) {
	scenario := testscenario.Create(t, time.Minute, time.Minute)
	serverConfig = &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &tls.Certificate{
				Certificate: [][]byte{scenario.ServerCert.Raw},
				Leaf:        scenario.ServerCert,
				PrivateKey:  scenario.ServerKey,
			}, nil
		},
		ServerName: "127.0.0.1",
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  scenario.CAs,
	}
	clientConfig = &tls.Config{
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &tls.Certificate{
				Certificate: [][]byte{scenario.ClientCert.Raw},
				Leaf:        scenario.ClientCert,
				PrivateKey:  scenario.ClientKey,
			}, nil
		},
		ServerName: "127.0.0.1",
		RootCAs:    scenario.CAs,
	}
	return serverConfig, clientConfig
}
