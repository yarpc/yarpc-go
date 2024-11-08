// Copyright (c) 2024 Uber Technologies, Inc.
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

package transport_test

import (
	"context"
	"net"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	ytchannel "go.uber.org/yarpc/transport/tchannel"
)

type echoReqBody struct{}
type echoResBody struct{}

type handler struct {
	client json.Client
	t      *testing.T
}

func (h handler) echo(ctx context.Context) error {
	return h.client.Call(ctx, "echo", &echoReqBody{}, &echoResBody{})
}

func (h handler) echoEcho(ctx context.Context) error {
	return h.client.Call(ctx, "echoecho", &echoReqBody{}, &echoResBody{})
}

func (h handler) createContextWithBaggage(tracer opentracing.Tracer) (context.Context, func()) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, testtime.Second)

	span := tracer.StartSpan("test")
	// no defer span.Finish()
	span.SetBaggageItem("weapon", "knife")
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ctx, cancel
}

func createGRPCDispatcher(t *testing.T, tracer opentracing.Tracer, enableTracingMiddleware bool) *yarpc.Dispatcher {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcTransport := grpc.NewTransport(grpc.Tracer(tracer), grpc.TracingInterceptorEnabled(enableTracingMiddleware))
	return yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: yarpc.Inbounds{
			grpcTransport.NewInbound(listener),
		},
		Outbounds: yarpc.Outbounds{
			"yarpc-test": {
				Unary: grpcTransport.NewSingleOutbound(yarpctest.ZeroAddrToHostPort(listener.Addr())),
			},
		},
	})
}

func createHTTPDispatcher(tracer opentracing.Tracer, enableTracingMiddleware bool) *yarpc.Dispatcher {
	// TODO: Use port 0 once https://github.com/yarpc/yarpc-go/issues/381 is
	httpTransport := http.NewTransport(http.Tracer(tracer), http.TracingInterceptorEnabled(enableTracingMiddleware))
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: yarpc.Inbounds{
			httpTransport.NewInbound("127.0.0.1:18080"),
		},
		Outbounds: yarpc.Outbounds{
			"yarpc-test": {
				Unary: httpTransport.NewSingleOutbound("http://127.0.0.1:18080"),
			},
		},
	})

	return dispatcher
}

//lint:ignore U1000 Ignore "method not used" lint as this is invoked by skipped tests.
func createTChannelDispatcher(t *testing.T, tracer opentracing.Tracer, enableTracingMiddleware bool) *yarpc.Dispatcher {
	hp := "127.0.0.1:4040"
	tchannelTransport, err := ytchannel.NewChannelTransport(
		ytchannel.ListenAddr(hp),
		ytchannel.Tracer(tracer),
		ytchannel.TracingInterceptorEnabled(enableTracingMiddleware),
		ytchannel.Tracer(tracer),
		ytchannel.ServiceName("yarpc-test"),
	)
	require.NoError(t, err)

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: yarpc.Inbounds{
			tchannelTransport.NewInbound(),
		},
		Outbounds: yarpc.Outbounds{
			"yarpc-test": {
				Unary: tchannelTransport.NewSingleOutbound(hp),
			},
		},
	})

	return dispatcher
}

var tests = []struct {
	name                    string
	enableTracingMiddleware bool
}{
	{
		name:                    "disable-tracing-middleware",
		enableTracingMiddleware: false,
	},
	{
		name:                    "enable-tracing-middleware",
		enableTracingMiddleware: true,
	},
}

func mapContexts(spans []*mocktracer.MockSpan) []int {
	ids := make([]int, len(spans))
	for i, span := range spans {
		ids[i] = span.Context().(mocktracer.MockSpanContext).TraceID
	}
	return ids
}
