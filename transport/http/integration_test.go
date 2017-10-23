// Copyright (c) 2017 Uber Technologies, Inc.
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/clientconfig"
)

func TestBothResponseError(t *testing.T) {
	doWithTestEnv(t, testEnvOptions{}, func(t *testing.T, testEnv *testEnv) {

	})
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
