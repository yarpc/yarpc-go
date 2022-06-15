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
	"go.uber.org/yarpc/transport/internal/tlsmux"
	"go.uber.org/yarpc/transport/internal/tlsscenario"
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

		expectError    bool
		clientErrorMsg string
	}{
		{
			desc: "plaintext_client",
			body: []byte("plaintext_body"),
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
			expectError:    true,
			clientErrorMsg: "remote error: tls: protocol version not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			wg := sync.WaitGroup{}
			defer wg.Wait()

			lis, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err, "unexpected error on listening")

			muxLis := tlsmux.NewListener(lis, serverTlsConfig)
			defer muxLis.Close()

			wg.Add(1)
			go func() {
				defer wg.Done()
				conn, err := muxLis.Accept()
				if tt.expectError {
					require.Error(t, err, "unexpected empty error")
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

			if tt.expectError {
				require.EqualError(t, err, tt.clientErrorMsg)
				return
			}
			require.NoError(t, err)

			_, err = conn.Write(tt.body)
			require.NoError(t, err, "unexpected error")

			response := make([]byte, len(tt.body))
			n, err := conn.Read(response)
			assert.NoError(t, err, "unexpected error")
			assert.Equal(t, tt.body, response[:n], "unexpected response")
		})
	}
}
