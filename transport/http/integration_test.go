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
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/internal/clientconfig"
	pkgerrors "go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/transport/internal/tls/testscenario"
)

func TestInboundTLS(t *testing.T) {
	defer goleak.VerifyNone(t)

	scenario := testscenario.Create(t, time.Minute, time.Minute)
	tests := []struct {
		desc             string
		inboundOptions   []InboundOption
		transportOptions []TransportOption
		isTLSClient      bool
	}{
		{
			desc: "plaintext_client_permissive_tls_server",
			inboundOptions: []InboundOption{
				InboundTLSConfiguration(scenario.ServerTLSConfig()),
				InboundTLSMode(yarpctls.Permissive),
			},
		},
		{
			desc: "tls_client_enforced_tls_server",
			inboundOptions: []InboundOption{
				InboundTLSConfiguration(scenario.ServerTLSConfig()),
				InboundTLSMode(yarpctls.Enforced),
			},
			transportOptions: []TransportOption{
				DialContext(func(ctx context.Context, network, addr string) (net.Conn, error) {
					return tls.Dial(network, addr, scenario.ClientTLSConfig())
				}),
			},
			isTLSClient: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			doWithTestEnv(t, testEnvOptions{
				Procedures:       json.Procedure("testFoo", testFooHandler),
				InboundOptions:   tt.inboundOptions,
				TransportOptions: tt.transportOptions,
			}, func(t *testing.T, testEnv *testEnv) {
				client := json.New(testEnv.ClientConfig)
				var response testFooResponse
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				err := client.Call(ctx, "testFoo", &testFooRequest{One: "one"}, &response)
				require.Nil(t, err)
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
