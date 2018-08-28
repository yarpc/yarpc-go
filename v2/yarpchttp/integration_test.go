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

package yarpchttp

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/clientconfig"
	"go.uber.org/yarpc/v2/yarpcerrors"
	"go.uber.org/yarpc/v2/yarpcjson"
)

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
				Procedures: yarpcjson.Procedure("testFoo", testFooHandler),
				Inbound: &Inbound{
					legacyResponseError: !tt.inboundBothResponseError,
				},
				Outbound: &Outbound{
					legacyResponseError: !tt.outboundBothResponseError,
				},
			}, func(t *testing.T, testEnv *testEnv) {
				client := yarpcjson.New(testEnv.ClientConfig)
				var response testFooResponse
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := client.Call(ctx, "testFoo", &testFooRequest{One: "one", Error: "bar"}, &response)

				assert.Equal(t, yarpcerrors.WrapHandlerError(errors.New("bar"), "example", "testFoo"), err)
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
	Dialer       *Dialer
	Inbound      *Inbound
	Outbound     *Outbound
	ClientConfig yarpc.ClientConfig
}

type testEnvOptions struct {
	Procedures []yarpc.Procedure
	Inbound    *Inbound
	Outbound   *Outbound
}

func newTestEnv(options testEnvOptions) (_ *testEnv, err error) {
	dialer := &Dialer{}
	if err := dialer.Start(context.Background()); err != nil {
		return nil, err
	}

	inbound := options.Inbound
	inbound.Address = "127.0.0.1:0"
	inbound.Router = newTestRouter(options.Procedures)
	if err := inbound.Start(context.Background()); err != nil {
		return nil, err
	}

	outbound := options.Outbound
	outbound.Dialer = dialer
	outbound.URL = parseURL("http://" + inbound.Addr().String())

	caller := "example-client"
	service := "example"
	clientConfig := clientconfig.MultiOutbound(
		caller,
		service,
		yarpc.Outbounds{
			ServiceName: caller,
			Unary:       outbound,
		},
	)

	return &testEnv{
		dialer,
		inbound,
		outbound,
		clientConfig,
	}, nil
}

func (e *testEnv) Close() error {
	return multierr.Combine(
		e.Dialer.Stop(context.Background()),
		e.Inbound.Stop(context.Background()),
	)
}

type testRouter struct {
	procedures []yarpc.Procedure
}

func newTestRouter(procedures []yarpc.Procedure) *testRouter {
	return &testRouter{procedures}
}

func (r *testRouter) Procedures() []yarpc.Procedure {
	return r.procedures
}

func (r *testRouter) Choose(_ context.Context, request *yarpc.Request) (yarpc.HandlerSpec, error) {
	for _, procedure := range r.procedures {
		if procedure.Name == request.Procedure {
			return procedure.HandlerSpec, nil
		}
	}
	return yarpc.HandlerSpec{}, fmt.Errorf("no procedure for name %s", request.Procedure)
}
