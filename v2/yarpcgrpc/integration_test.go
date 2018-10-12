// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcgrpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpctest"
	"google.golang.org/grpc/credentials"
)

func TestYARPCMaxMsgSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		value                string
		serverMaxRecvMsgSize int
		serverMaxSendMsgSize int
		clientMaxRecvMsgSize int
		clientMaxSendMsgSize int

		errCode yarpcerror.Code
	}{
		{
			name:  "defaults",
			value: "hello!",
		},
		{
			name:                 "larger than server recieve",
			value:                strings.Repeat("a", 10),
			serverMaxRecvMsgSize: 1,
			errCode:              yarpcerror.CodeResourceExhausted,
		},
		{
			name:                 "larger than client send",
			value:                strings.Repeat("a", 10),
			clientMaxSendMsgSize: 1,
			errCode:              yarpcerror.CodeResourceExhausted,
		},
		{
			name:                 "larger than server send",
			value:                strings.Repeat("a", 10),
			serverMaxSendMsgSize: 1,
			errCode:              yarpcerror.CodeResourceExhausted,
		},
		{
			name:                 "larger than client recieve",
			value:                strings.Repeat("a", 10),
			clientMaxRecvMsgSize: 1,
			errCode:              yarpcerror.CodeResourceExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := testEnvOptions{
				Procedures: yarpcjson.Procedure("test-procedure", testEchoHandler),
				Inbound: &Inbound{
					ServerMaxRecvMsgSize: tt.serverMaxRecvMsgSize,
					ServerMaxSendMsgSize: tt.serverMaxSendMsgSize,
				},
				Dialer: &Dialer{
					ClientMaxRecvMsgSize: tt.clientMaxRecvMsgSize,
					ClientMaxSendMsgSize: tt.clientMaxSendMsgSize,
				},
			}

			doWithTestEnv(t, te, func(t *testing.T, testEnv *testEnv) {
				client := yarpcjson.New(testEnv.Client)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				var res testEchoResponse
				err := client.Call(ctx, "test-procedure", &testEchoRequest{Message: tt.value}, &res)
				require.Equal(t, tt.errCode.String(), yarpcerror.FromError(err).Code().String())
			})
		})
	}
}

func TestJSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		inbound  *Inbound
		outbound *Outbound
		dialer   *Dialer

		request   *testEchoRequest
		procedure string

		wantErr string
	}{
		{
			name:      "basic",
			procedure: "test-procedure",
			request: &testEchoRequest{
				Message: "hello",
			},
		},
		{
			name:      "echo err",
			procedure: "test-procedure",
			request: &testEchoRequest{
				Error: "handler error",
			},
			wantErr: "handler error",
		},
		{
			name:      "echo err",
			procedure: "invalid procedure",
			wantErr:   "no procedure for name invalid procedure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doWithTestEnv(t, testEnvOptions{
				Procedures: yarpcjson.Procedure("test-procedure", testEchoHandler),
				Inbound:    tt.inbound,
				Outbound:   tt.outbound,
				Dialer:     tt.dialer,
			}, func(t *testing.T, testEnv *testEnv) {
				client := yarpcjson.New(testEnv.Client)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				var res testEchoResponse
				err := client.Call(ctx, tt.procedure, tt.request, &res)

				if tt.wantErr == "" {
					require.NoError(t, err, "unexpected error")

				} else {
					require.Error(t, err, "expected error")
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			})
		})
	}
}

func TestConcurrentCalls(t *testing.T) {
	t.Parallel()

	options := testEnvOptions{
		Procedures: yarpcjson.Procedure("test-procedure", testEchoHandler),
	}

	doWithTestEnv(t, options, func(t *testing.T, testEnv *testEnv) {
		client := yarpcjson.New(testEnv.Client)

		var (
			wg   sync.WaitGroup
			lock sync.Mutex
			errs error
		)
		start := make(chan struct{}, 0)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				msg := fmt.Sprintf("foo bar %d", i)
				var res testEchoResponse

				<-start
				err := client.Call(ctx, "test-procedure", &testEchoRequest{Message: msg}, &res)

				lock.Lock()
				errs = multierr.Combine(errs, err)
				lock.Unlock()

				wg.Done()
			}(i)
		}
		close(start)
		wg.Wait()
		require.NoError(t, errs)
	})
}

type testEchoRequest struct {
	Message string
	Error   string
}

type testEchoResponse struct {
	Message string
}

func testEchoHandler(_ context.Context, request *testEchoRequest) (*testEchoResponse, error) {
	if request.Error != "" {
		return nil, errors.New(request.Error)
	}
	return &testEchoResponse{
		Message: request.Message,
	}, nil
}

type testEnv struct {
	Inbound  *Inbound
	Outbound *Outbound
	Dialer   *Dialer
	Client   yarpc.Client
}

type testEnvOptions struct {
	Procedures []yarpc.Procedure
	Inbound    *Inbound
	Outbound   *Outbound
	Dialer     *Dialer
}

func doWithTestEnv(t *testing.T, options testEnvOptions, f func(*testing.T, *testEnv)) {
	testEnv, err := newTestEnv(options)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, testEnv.Close())
	}()
	f(t, testEnv)
}

func parseURL(in string) *url.URL {
	out, _ := url.Parse(in)
	return out
}

func newTestEnv(options testEnvOptions) (_ *testEnv, err error) {
	dialer := options.Dialer
	if dialer == nil {
		dialer = &Dialer{}
	}
	if err := dialer.Start(context.Background()); err != nil {
		return nil, err
	}

	inbound := options.Inbound
	if inbound == nil {
		inbound = &Inbound{}
	}

	inbound.Addr = "127.0.0.1:0"
	inbound.Router = yarpctest.NewFakeRouter(options.Procedures)
	if err := inbound.Start(context.Background()); err != nil {
		return nil, err
	}

	outbound := options.Outbound
	if outbound == nil {
		outbound = &Outbound{}
	}
	outbound.Dialer = dialer
	outbound.URL = parseURL("http://" + inbound.Listener.Addr().String())

	client := yarpc.Client{
		Service: "test-service",
		Caller:  "test-caller",
		Unary:   outbound,
	}

	return &testEnv{
		Inbound:  inbound,
		Outbound: outbound,
		Dialer:   dialer,
		Client:   client,
	}, nil
}

func (e *testEnv) Close() error {
	return multierr.Combine(
		e.Dialer.Stop(context.Background()),
		e.Inbound.Stop(context.Background()),
	)
}

func TestTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		clientValidity      time.Duration
		serverValidity      time.Duration
		expectedErrContains string
	}{
		{
			name:           "valid certs both sides",
			clientValidity: time.Minute,
			serverValidity: time.Minute,
		},
		{
			name:                "invalid server cert",
			clientValidity:      time.Minute,
			serverValidity:      -1,
			expectedErrContains: "transport: authentication handshake failed: x509: certificate has expired or is not yet valid",
		},
		{
			name:                "invalid client cert",
			clientValidity:      -1,
			serverValidity:      time.Minute,
			expectedErrContains: "remote error: tls: bad certificate",
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
				Procedures: yarpcjson.Procedure("test-procedure", testEchoHandler),
				Inbound: &Inbound{
					Credentials: serverCreds,
				},
				Dialer: &Dialer{
					Credentials: clientCreds,
				},
			}
			doWithTestEnv(t, te, func(t *testing.T, testEnv *testEnv) {
				client := yarpcjson.New(testEnv.Client)

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				var res testEchoResponse
				request := &testEchoRequest{
					Message: "hello security!",
				}
				err := client.Call(ctx, "test-procedure", request, &res)

				if test.expectedErrContains == "" {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					assert.Contains(t, err.Error(), test.expectedErrContains)
				}
			})
		})
	}
}

type tlsScenario struct {
	CAs        *x509.CertPool
	ServerCert *x509.Certificate
	ServerKey  *ecdsa.PrivateKey
	ClientCert *x509.Certificate
	ClientKey  *ecdsa.PrivateKey
}

func createTLSScenario(t *testing.T, clientValidity time.Duration, serverValidity time.Duration) tlsScenario {
	now := time.Now()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "test ca",
			},
			SerialNumber:          big.NewInt(1),
			BasicConstraintsValid: true,
			IsCA:      true,
			KeyUsage:  x509.KeyUsageCertSign,
			NotBefore: now,
			NotAfter:  now.Add(10 * time.Minute),
		},
		&x509.Certificate{},
		caKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(caBytes)
	require.NoError(t, err)

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	serverCertBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "server",
			},
			NotAfter:     now.Add(serverValidity),
			SerialNumber: big.NewInt(2),
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		},
		ca,
		serverKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	serverCert, err := x509.ParseCertificate(serverCertBytes)
	require.NoError(t, err)

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	clientCertBytes, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			Subject: pkix.Name{
				CommonName: "client",
			},
			NotAfter:     now.Add(clientValidity),
			SerialNumber: big.NewInt(3),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		},
		ca,
		clientKey.Public(),
		caKey,
	)
	require.NoError(t, err)
	clientCert, err := x509.ParseCertificate(clientCertBytes)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	pool.AddCert(ca)

	return tlsScenario{
		CAs:        pool,
		ServerCert: serverCert,
		ServerKey:  serverKey,
		ClientCert: clientCert,
		ClientKey:  clientKey,
	}
}
