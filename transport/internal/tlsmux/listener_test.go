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

package tlsmux_test

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/internal/tlsmux"
	"go.uber.org/yarpc/transport/internal/tlsscenario"
	"go.uber.org/zap"
)

func TestMux(t *testing.T) {
	scenario := tlsscenario.Create(t, time.Minute, time.Minute)
	serverTlsConfig := &tls.Config{
		GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &tls.Certificate{
				Certificate: [][]byte{scenario.ServerCert.Raw},
				Leaf:        scenario.ServerCert,
				PrivateKey:  scenario.ServerKey,
			}, nil
		},
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  scenario.CAs,
	}

	tests := []struct {
		desc            string
		clientTlsConfig *tls.Config
		body            []byte
		mode            transport.InboundTLSMode

		expectedCounter metrics.Snapshot

		clientErrorMsgOnDial   string
		clientErrorMsgOnRead   string
		serverErrorMsgOnAccept string
	}{
		{
			desc: "plaintext_connections",
			body: []byte("plaintext_body"),
			expectedCounter: metrics.Snapshot{
				Name: "plaintext_connections",
				Tags: metrics.Tags{
					"service":   "test-svc",
					"transport": "test-transport",
					"component": "yarpc",
					"mode":      "permissive",
				},
				Value: 1,
			},
			mode: transport.Permissive,
		},
		{
			desc: "plaintext_connection_rejection_on_enforced",
			body: []byte("plaintext_body"),
			expectedCounter: metrics.Snapshot{
				Name: "plaintext_connection_rejects",
				Tags: metrics.Tags{
					"service":   "test-svc",
					"transport": "test-transport",
					"component": "yarpc",
					"mode":      "enforced",
				},
				Value: 1,
			},
			mode:                   transport.Enforced,
			clientErrorMsgOnRead:   "connection reset by peer",
			serverErrorMsgOnAccept: "listener closed",
		},
		{
			desc: "tls_client",
			clientTlsConfig: &tls.Config{
				GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
					return &tls.Certificate{
						Certificate: [][]byte{scenario.ClientCert.Raw},
						Leaf:        scenario.ClientCert,
						PrivateKey:  scenario.ClientKey,
					}, nil
				},
				MinVersion: tls.VersionTLS10,
				MaxVersion: tls.VersionTLS13,
				RootCAs:    scenario.CAs,
			},
			body: []byte("tls_body"),
			expectedCounter: metrics.Snapshot{
				Name: "tls_connections",
				Tags: metrics.Tags{
					"service":   "test-svc",
					"transport": "test-transport",
					"version":   "1.3",
					"component": "yarpc",
					"mode":      "permissive",
				},
				Value: 1,
			},
			mode: transport.Permissive,
		},
		{
			desc: "tls_handshake_failure",
			clientTlsConfig: &tls.Config{
				GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
					return &tls.Certificate{
						Certificate: [][]byte{scenario.ClientCert.Raw},
						Leaf:        scenario.ClientCert,
						PrivateKey:  scenario.ClientKey,
					}, nil
				},
				MinVersion: tls.VersionTLS10,
				MaxVersion: tls.VersionTLS11,
				RootCAs:    scenario.CAs,
			},
			expectedCounter: metrics.Snapshot{
				Name: "tls_handshake_failures",
				Tags: metrics.Tags{
					"service":   "test-svc",
					"transport": "test-transport",
					"component": "yarpc",
					"mode":      "permissive",
				},
				Value: 1,
			},
			clientErrorMsgOnDial:   "remote error: tls: protocol version not supported",
			serverErrorMsgOnAccept: "listener closed",
			mode:                   transport.Permissive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			wg := sync.WaitGroup{}
			defer wg.Wait()

			lis, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err, "unexpected error on listening")

			root := metrics.New()
			muxLis := tlsmux.NewListener(tlsmux.Config{
				Listener:      lis,
				TLSConfig:     serverTlsConfig,
				Meter:         root.Scope(),
				Logger:        zap.NewNop(),
				ServiceName:   "test-svc",
				TransportName: "test-transport",
				Mode:          tt.mode,
			})
			defer muxLis.Close()

			wg.Add(1)
			go func() {
				defer wg.Done()
				conn, err := muxLis.Accept()
				if tt.serverErrorMsgOnAccept != "" {
					assert.EqualError(t, err, tt.serverErrorMsgOnAccept)
					return
				}
				require.NoError(t, err)
				defer conn.Close()

				request := make([]byte, len(tt.body))
				n, err := io.ReadFull(conn, request)
				require.NoError(t, err)
				assert.Equal(t, tt.body, request[:n], "unexpected request")

				_, err = conn.Write(request)
				assert.NoError(t, err, "unexpected error")
			}()

			var conn net.Conn
			if tt.clientTlsConfig != nil {
				conn, err = tls.Dial(lis.Addr().Network(), lis.Addr().String(), tt.clientTlsConfig)
			} else {
				conn, err = net.Dial(lis.Addr().Network(), lis.Addr().String())
			}

			if tt.clientErrorMsgOnDial != "" {
				assert.EqualError(t, err, tt.clientErrorMsgOnDial)
				return
			}
			require.NoError(t, err)

			_, err = conn.Write(tt.body)
			require.NoError(t, err, "unexpected error")

			response := make([]byte, len(tt.body))
			n, err := conn.Read(response)
			if tt.clientErrorMsgOnRead != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.clientErrorMsgOnRead)
				return
			}

			assert.NoError(t, err, "unexpected error")
			assert.Equal(t, tt.body, response[:n], "unexpected response")
			assert.Contains(t, root.Snapshot().Counters, tt.expectedCounter, "unexpected counters")
		})
	}
}

func TestConcurrentConnections(t *testing.T) {
	defer goleak.VerifyNone(t)

	scenario := tlsscenario.Create(t, time.Minute, time.Minute)
	serverTlsConfig := &tls.Config{
		GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &tls.Certificate{
				Certificate: [][]byte{scenario.ServerCert.Raw},
				Leaf:        scenario.ServerCert,
				PrivateKey:  scenario.ServerKey,
			}, nil
		},
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  scenario.CAs,
	}
	clientTlsConfig := &tls.Config{
		GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &tls.Certificate{
				Certificate: [][]byte{scenario.ClientCert.Raw},
				Leaf:        scenario.ClientCert,
				PrivateKey:  scenario.ClientKey,
			}, nil
		},
		MinVersion: tls.VersionTLS10,
		MaxVersion: tls.VersionTLS13,
		RootCAs:    scenario.CAs,
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "unexpected error on listening")

	muxLis := tlsmux.NewListener(tlsmux.Config{
		Listener:      lis,
		TLSConfig:     serverTlsConfig,
		Meter:         metrics.New().Scope(),
		Logger:        zap.NewNop(),
		ServiceName:   "test-svc",
		TransportName: "test-transport",
		Mode:          transport.Permissive,
	})
	defer muxLis.Close()

	msg := "hello world"
	totalConnections := 100

	var wg sync.WaitGroup
	defer wg.Wait()
	for i := 0; i < totalConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			var conn net.Conn
			var err error
			if id%2 == 0 {
				conn, err = tls.Dial(lis.Addr().Network(), lis.Addr().String(), clientTlsConfig)
			} else {
				conn, err = net.Dial(lis.Addr().Network(), lis.Addr().String())
			}

			require.NoError(t, err, "unexpected error on dial")
			defer conn.Close()

			n, err := conn.Write([]byte(msg))
			require.NoError(t, err, "unexpected error on client write")
			assert.Equal(t, len(msg), n, "unexpected write length")

			buf := make([]byte, len(msg))
			n, err = io.ReadFull(conn, buf)
			require.NoError(t, err)
			assert.Equal(t, len(msg), n)
			assert.Equal(t, msg, string(buf))
		}(i)
	}

	for i := 0; i < totalConnections; i++ {
		conn, err := muxLis.Accept()
		require.NoError(t, err)

		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()

			buf := make([]byte, len(msg))
			n, err := io.ReadFull(c, buf)
			require.NoError(t, err)
			assert.Equal(t, n, len(msg))

			n, err = c.Write(buf)
			require.NoError(t, err)
			assert.Equal(t, n, len(msg))
		}(conn)
	}

	wg.Wait()
}
