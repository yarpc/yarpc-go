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

package yarpctchannel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaltesttime"
	"go.uber.org/yarpc/v2/yarpcrouter"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestInboundInvalidAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
	defer cancel()

	inbound := &Inbound{
		Addr: "invalid",
	}
	if !assert.Error(t, inbound.Start(ctx)) {
		require.NoError(t, inbound.Stop(ctx))
	}
}

type whoSentYouHandler struct{}

func (whoSentYouHandler) Handle(ctx context.Context, req *yarpc.Request, reqBody *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	return &yarpc.Response{}, yarpc.NewBufferString(req.Service), nil
}

func TestInboundSubServices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
	defer cancel()

	whoSentYouHandlerSpec := yarpc.NewUnaryTransportHandlerSpec(whoSentYouHandler{})
	router := yarpcrouter.NewMapRouter("myservice", []yarpc.TransportProcedure{
		{
			Name:        "hello",
			Encoding:    "raw",
			HandlerSpec: whoSentYouHandlerSpec,
		},
		{
			Service:     "subservice",
			Name:        "hello",
			Encoding:    yarpc.Encoding("raw"),
			HandlerSpec: whoSentYouHandlerSpec,
		},
		{
			Service:     "subservice",
			Name:        "world",
			Encoding:    yarpc.Encoding("raw"),
			HandlerSpec: whoSentYouHandlerSpec,
		},
		{
			Service:     "subservice2",
			Name:        "hello",
			Encoding:    yarpc.Encoding("raw"),
			HandlerSpec: whoSentYouHandlerSpec,
		},
		{
			Service:     "subservice2",
			Name:        "monde",
			Encoding:    yarpc.Encoding("raw"),
			HandlerSpec: whoSentYouHandlerSpec,
		},
	})

	t.Logf("Router %#v\n", router)

	inbound := &Inbound{
		Service: "myservice",
		Addr:    "127.0.0.1:0",
		Router:  router,
	}
	require.NoError(t, inbound.Start(ctx))
	defer inbound.Stop(ctx)

	dialer := &Dialer{
		Caller: "caller",
	}
	require.NoError(t, dialer.Start(ctx))
	defer dialer.Stop(ctx)

	outbound := &Outbound{
		Dialer: dialer,
		Addr:   inbound.Listener.Addr().String(),
	}

	for _, tt := range []struct {
		service   string
		procedure string
	}{
		{"myservice", "hello"},
		{"subservice", "hello"},
		{"subservice", "world"},
		{"subservice2", "hello"},
		{"subservice2", "monde"},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 200*internaltesttime.Millisecond)
		defer cancel()
		res, resBody, err := outbound.Call(
			ctx,
			&yarpc.Request{
				Caller:    "caller",
				Service:   tt.service,
				Procedure: tt.procedure,
				Encoding:  yarpc.Encoding("raw"),
			},
			yarpc.NewBufferBytes(nil),
		)
		if !assert.NoError(t, err, "failed to make call") {
			continue
		}
		if !assert.Nil(t, res.ApplicationErrorInfo, "not application error") {
			continue
		}
		assert.Equal(t, resBody, yarpc.NewBufferString(tt.service))
	}
}

func TestArbitraryInboundServiceOutboundCallerName(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
	defer cancel()

	router := yarpctest.EchoRouter{}
	inbound := &Inbound{
		Service: "service",
		Router:  router,
		Addr:    "127.0.0.1:0",
	}
	require.NoError(t, inbound.Start(ctx))
	defer inbound.Stop(ctx)

	dialer := &Dialer{
		Caller: "caller",
	}
	require.NoError(t, dialer.Start(ctx))
	defer dialer.Stop(ctx)

	outbound := &Outbound{
		Dialer: dialer,
		Addr:   inbound.Listener.Addr().String(),
	}

	tests := []struct {
		msg             string
		caller, service string
	}{
		{"from service to foo", "service", "foo"},
		{"from bar to service", "bar", "service"},
		{"from foo to bar", "foo", "bar"},
		{"from bar to foo", "bar", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*internaltesttime.Millisecond)
			defer cancel()
			res, resBody, err := outbound.Call(
				ctx,
				&yarpc.Request{
					Caller:    tt.caller,
					Service:   tt.service,
					Encoding:  yarpc.Encoding("raw"),
					Procedure: "procedure",
				},
				yarpc.NewBufferString(tt.msg),
			)
			assert.NotNil(t, res)
			if !assert.NoError(t, err, "call success") {
				return
			}
			assert.NoError(t, err, "read response body")
			assert.Equal(t, resBody, yarpc.NewBufferString(tt.msg), "response echoed")
		})
	}
}
