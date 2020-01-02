// Copyright (c) 2020 Uber Technologies, Inc.
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

// +build !darwin

// On Darwin (Mac OS architecture), the error for failing to establish a TLS
// connection through gRPC is not as informative as the error we expect on
// other architectures.
// https://github.com/yarpc/yarpc-go/issues/1854

package grpc

import (
	"crypto/tls"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/credentials"
)

func TestTLSWithYARPCAndGRPC(t *testing.T) {

	tests := []struct {
		clientValidity      time.Duration
		serverValidity      time.Duration
		expectedErrContains string
		name                string
	}{
		{
			clientValidity: time.Minute,
			serverValidity: time.Minute,
			name:           "valid certs both sides",
		},
		{
			clientValidity:      time.Minute,
			serverValidity:      -1,
			expectedErrContains: "transport: authentication handshake failed: x509: certificate has expired or is not yet valid",
			name:                "invalid server cert",
		},
		{
			clientValidity:      -1,
			serverValidity:      time.Minute,
			expectedErrContains: "remote error: tls: bad certificate",
			name:                "invalid client cert",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scenario := createTLSScenario(t, test.clientValidity, test.serverValidity)

			serverCreds := credentials.NewTLS(&tls.Config{
				GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
					return &tls.Certificate{
						Certificate: [][]byte{scenario.ServerCert.Raw},
						Leaf:        scenario.ServerCert,
						PrivateKey:  scenario.ServerKey,
					}, nil
				},
				ClientAuth: tls.RequireAndVerifyClientCert,
				ClientCAs:  scenario.CAs,
			})

			clientCreds := credentials.NewTLS(&tls.Config{
				GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
					return &tls.Certificate{
						Certificate: [][]byte{scenario.ClientCert.Raw},
						Leaf:        scenario.ClientCert,
						PrivateKey:  scenario.ClientKey,
					}, nil
				},
				RootCAs: scenario.CAs,
			})

			te := testEnvOptions{
				InboundOptions: []InboundOption{InboundCredentials(serverCreds)},
				DialOptions:    []DialOption{DialerCredentials(clientCreds)},
			}
			te.do(t, func(t *testing.T, e *testEnv) {
				err := e.SetValueYARPC(context.Background(), "foo", "bar")
				expectErrorContains(t, err, test.expectedErrContains)

				err = e.SetValueGRPC(context.Background(), "foo", "bar")
				expectErrorContains(t, err, test.expectedErrContains)
			})
		})
	}
}
