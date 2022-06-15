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

package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"testing"
	"time"

	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/internal/clientconfig"
	pkgerrors "go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/transport/internal/tlsscenario"
)

func TestMuxTLS(t *testing.T) {
	defer goleak.VerifyNone(t)

	scenario := tlsscenario.Create(t, time.Minute, time.Minute)
	serverTLSConfig := &tls.Config{
		GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &tls.Certificate{
				Certificate: [][]byte{scenario.ServerCert.Raw},
				Leaf:        scenario.ServerCert,
				PrivateKey:  scenario.ServerKey,
			}, nil
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  scenario.CAs,
	}
	tests := []struct {
		desc           string
		inboundOptions []InboundOption
		isTLSClient    bool
	}{
		{
			desc: "plaintext_client_mux_tls_server",
			inboundOptions: []InboundOption{
				InboundMuxTLS(serverTLSConfig),
			},
		},
		{
			desc: "tls_client_mux_tls_server",
			inboundOptions: []InboundOption{
				InboundMuxTLS(serverTLSConfig),
			},
			isTLSClient: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			doWithTestEnv(t, testEnvOptions{
				Procedures:     json.Procedure("testFoo", testFooHandler),
				InboundOptions: tt.inboundOptions,
			}, func(t *testing.T, testEnv *testEnv) {
				cc := testEnv.ClientConfig
				if tt.isTLSClient {
					tr := NewTransport(buildClient(func(t *transportOptions) *http.Client {
						return &http.Client{
							Transport: &http.Transport{
								TLSClientConfig: &tls.Config{
									GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
										return &tls.Certificate{
											Certificate: [][]byte{scenario.ClientCert.Raw},
											Leaf:        scenario.ClientCert,
											PrivateKey:  scenario.ClientKey,
										}, nil
									},
									RootCAs: scenario.CAs,
								},
							},
						}
					}))
					require.NoError(t, tr.Start())
					defer tr.Stop()

					outbound := tr.NewSingleOutbound(fmt.Sprintf("http://%s", testEnv.Inbound.Addr().String()))
					require.NoError(t, outbound.Start())
					defer outbound.Stop()

					cc = clientconfig.MultiOutbound(
						"example-client",
						"example",
						transport.Outbounds{
							ServiceName: "example-client",
							Unary:       outbound,
						},
					)
				}

				client := json.New(cc)
				var response testFooResponse
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				err := client.Call(ctx, "testFoo", &testFooRequest{One: "one"}, &response)
				assert.Nil(t, err)
				assert.Equal(t, testFooResponse{One: "one"}, response)
			})
		})
	}
}

func TestBothResponseError(t *testing.T) {
	tests := []struct {
		inboundBothResponseError  bool
		outboundBothResponseError bool
	}{
		{
			inboundBothResponseError:  false,
			outboundBothResponseError: false,
		},
		{
			inboundBothResponseError:  false,
			outboundBothResponseError: true,
		},
		{
			inboundBothResponseError:  true,
			outboundBothResponseError: false,
		},
		{
			inboundBothResponseError:  true,
			outboundBothResponseError: true,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("inbound(%v)-outbound(%v)", tt.inboundBothResponseError, tt.outboundBothResponseError), func(t *testing.T) {
			doWithTestEnv(t, testEnvOptions{
				Procedures: json.Procedure("testFoo", testFooHandler),
				InboundOptions: []InboundOption{
					func(i *Inbound) {
						i.bothResponseError = tt.inboundBothResponseError
					},
				},
				OutboundOptions: []OutboundOption{
					func(o *Outbound) {
						o.bothResponseError = tt.outboundBothResponseError
					},
				},
			}, func(t *testing.T, testEnv *testEnv) {
				client := json.New(testEnv.ClientConfig)
				var response testFooResponse
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := client.Call(ctx, "testFoo", &testFooRequest{One: "one", Error: "bar"}, &response)

				assert.Equal(t, pkgerrors.WrapHandlerError(errors.New("bar"), "example", "testFoo"), err)
				if tt.inboundBothResponseError && tt.outboundBothResponseError {
					assert.Equal(t, "one", response.One)
				} else {
					assert.Empty(t, response.One)
				}
			})
		})
	}
}

type testFooRequest struct {
	One   string
	Error string
}

type testFooResponse struct {
	One string
}

func testFooHandler(_ context.Context, request *testFooRequest) (*testFooResponse, error) {
	var err error
	if request.Error != "" {
		err = errors.New(request.Error)
	}
	return &testFooResponse{
		One: request.One,
	}, err
}

func doWithTestEnv(t *testing.T, options testEnvOptions, f func(*testing.T, *testEnv)) {
	testEnv, err := newTestEnv(options)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, testEnv.Close())
	}()
	f(t, testEnv)
}

type testEnv struct {
	Inbound      *Inbound
	Outbound     *Outbound
	ClientConfig transport.ClientConfig
}

type testEnvOptions struct {
	Procedures       []transport.Procedure
	TransportOptions []TransportOption
	InboundOptions   []InboundOption
	OutboundOptions  []OutboundOption
}

func newTestEnv(options testEnvOptions) (_ *testEnv, err error) {
	t := NewTransport(options.TransportOptions...)
	if err := t.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, t.Stop())
		}
	}()

	inbound := t.NewInbound("127.0.0.1:0", options.InboundOptions...)
	inbound.SetRouter(newTestRouter(options.Procedures))
	if err := inbound.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, inbound.Stop())
		}
	}()

	outbound := t.NewSingleOutbound(fmt.Sprintf("http://%s", inbound.Addr().String()), options.OutboundOptions...)
	if err := outbound.Start(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = multierr.Append(err, outbound.Stop())
		}
	}()

	caller := "example-client"
	service := "example"
	clientConfig := clientconfig.MultiOutbound(
		caller,
		service,
		transport.Outbounds{
			ServiceName: caller,
			Unary:       outbound,
		},
	)

	return &testEnv{
		inbound,
		outbound,
		clientConfig,
	}, nil
}

func (e *testEnv) Close() error {
	return multierr.Combine(
		e.Outbound.Stop(),
		e.Inbound.Stop(),
	)
}

type testRouter struct {
	procedures []transport.Procedure
}

func newTestRouter(procedures []transport.Procedure) *testRouter {
	return &testRouter{procedures}
}

func (r *testRouter) Procedures() []transport.Procedure {
	return r.procedures
}

func (r *testRouter) Choose(_ context.Context, request *transport.Request) (transport.HandlerSpec, error) {
	for _, procedure := range r.procedures {
		if procedure.Name == request.Procedure {
			return procedure.HandlerSpec, nil
		}
	}
	return transport.HandlerSpec{}, fmt.Errorf("no procedure for name %s", request.Procedure)
}
