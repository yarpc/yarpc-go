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

package transport_test

import (
	"context"
	"net"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
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

func (h handler) register(dispatcher *yarpc.Dispatcher) {
	dispatcher.Register(json.Procedure("echo", h.handleEcho))
	dispatcher.Register(json.Procedure("echoecho", h.handleEchoEcho))
}

func (h handler) handleEcho(ctx context.Context, reqBody *echoReqBody) (*echoResBody, error) {
	h.assertBaggage(ctx)
	return &echoResBody{}, nil
}

func (h handler) handleEchoEcho(ctx context.Context, reqBody *echoReqBody) (*echoResBody, error) {
	h.assertBaggage(ctx)
	var resBody echoResBody
	err := h.client.Call(ctx, "echo", reqBody, &resBody)
	return &resBody, err
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

func (h handler) assertBaggage(ctx context.Context) {
	span := opentracing.SpanFromContext(ctx)
	weapon := span.BaggageItem("weapon")
	assert.Equal(h.t, "knife", weapon, "baggage should propagate")
}

func createGRPCDispatcher(t *testing.T, tracer opentracing.Tracer) *yarpc.Dispatcher {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcTransport := grpc.NewTransport(grpc.Tracer(tracer))
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

func createHTTPDispatcher(tracer opentracing.Tracer) *yarpc.Dispatcher {
	// TODO: Use port 0 once https://github.com/yarpc/yarpc-go/issues/381 is
	// fixed.

	httpTransport := http.NewTransport(http.Tracer(tracer))
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

func createTChannelDispatcher(t *testing.T, tracer opentracing.Tracer) *yarpc.Dispatcher {
	hp := "127.0.0.1:4040"

	tchannelTransport, err := ytchannel.NewChannelTransport(
		ytchannel.ListenAddr(hp),
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

func TestGRPCTracer(t *testing.T) {
	tracer := mocktracer.New()
	dispatcher := createGRPCDispatcher(t, tracer)

	client := json.New(dispatcher.ClientConfig("yarpc-test"))
	handler := handler{client: client, t: t}
	handler.register(dispatcher)

	require.NoError(t, dispatcher.Start())
	defer dispatcher.Stop()

	ctx, cancel := handler.createContextWithBaggage(tracer)
	defer cancel()

	err := handler.echo(ctx)
	assert.NoError(t, err)

	AssertDepth1Spans(t, tracer)
}

func TestHTTPTracer(t *testing.T) {
	tracer := mocktracer.New()
	dispatcher := createHTTPDispatcher(tracer)

	client := json.New(dispatcher.ClientConfig("yarpc-test"))
	handler := handler{client: client, t: t}
	handler.register(dispatcher)

	require.NoError(t, dispatcher.Start())
	defer dispatcher.Stop()

	ctx, cancel := handler.createContextWithBaggage(tracer)
	defer cancel()

	err := handler.echo(ctx)
	assert.NoError(t, err)

	AssertDepth1Spans(t, tracer)
}

func TestTChannelTracer(t *testing.T) {
	t.Skip("TODO this test is flaky, we need to fix")
	tracer := mocktracer.New()
	dispatcher := createTChannelDispatcher(t, tracer)
	// Make this assertion at the end of the defer stack, when the channel has
	// been shut down. This ensures that all message exchanges have been shut
	// down, which means that all spans have been closed.
	defer AssertDepth1Spans(t, tracer)

	client := json.New(dispatcher.ClientConfig("yarpc-test"))
	handler := handler{client: client, t: t}
	handler.register(dispatcher)

	require.NoError(t, dispatcher.Start())
	defer dispatcher.Stop()

	ctx, cancel := handler.createContextWithBaggage(tracer)
	defer cancel()

	err := handler.echo(ctx)
	assert.NoError(t, err)
}

func AssertDepth1Spans(t *testing.T, tracer *mocktracer.MockTracer) {
	assert.Equal(t, 2, len(tracer.FinishedSpans()), "generates inbound and outband spans")
	if len(tracer.FinishedSpans()) != 2 {
		return
	}
	spans := tracer.FinishedSpans()
	parent := spans[0]
	child := spans[1]
	parentctx := parent.Context().(mocktracer.MockSpanContext)
	childctx := child.Context().(mocktracer.MockSpanContext)
	assert.Equal(t, parentctx.TraceID, childctx.TraceID, "parent and child trace ID do not match")
	// Whether the parent and child have the same span id is an implementation
	// detail of the tracer.
	assert.Equal(t, "echo", parent.OperationName, "span has correct operation name")
	assert.Equal(t, "echo", child.OperationName, "span has correct operation name")
}

func TestGRPCTracerDepth2(t *testing.T) {
	tracer := mocktracer.New()
	dispatcher := createGRPCDispatcher(t, tracer)

	client := json.New(dispatcher.ClientConfig("yarpc-test"))
	handler := handler{client: client, t: t}
	handler.register(dispatcher)

	require.NoError(t, dispatcher.Start())
	defer dispatcher.Stop()

	ctx, cancel := handler.createContextWithBaggage(tracer)
	defer cancel()

	err := handler.echoEcho(ctx)
	assert.NoError(t, err)
	AssertDepth2Spans(t, tracer)
}

func TestHTTPTracerDepth2(t *testing.T) {
	tracer := mocktracer.New()
	dispatcher := createHTTPDispatcher(tracer)

	client := json.New(dispatcher.ClientConfig("yarpc-test"))
	handler := handler{client: client, t: t}
	handler.register(dispatcher)

	require.NoError(t, dispatcher.Start())
	defer dispatcher.Stop()

	ctx, cancel := handler.createContextWithBaggage(tracer)
	defer cancel()

	err := handler.echoEcho(ctx)
	assert.NoError(t, err)
	AssertDepth2Spans(t, tracer)
}

func TestTChannelTracerDepth2(t *testing.T) {
	t.Skip("TODO this test is flaky, we need to fix")
	tracer := mocktracer.New()
	dispatcher := createTChannelDispatcher(t, tracer)
	// Make this assertion at the end of the defer stack, when the channel has
	// been shut down. This ensures that all message exchanges have been shut
	// down, which means that all spans have been closed.
	defer AssertDepth2Spans(t, tracer)

	client := json.New(dispatcher.ClientConfig("yarpc-test"))
	handler := handler{client: client, t: t}
	handler.register(dispatcher)

	require.NoError(t, dispatcher.Start())
	defer dispatcher.Stop()

	ctx, cancel := handler.createContextWithBaggage(tracer)
	defer cancel()

	err := handler.echoEcho(ctx)
	assert.NoError(t, err)
}

func AssertDepth2Spans(t *testing.T, tracer *mocktracer.MockTracer) {
	if !assert.Equal(t, 4, len(tracer.FinishedSpans()), "generates inbound and outband spans") {
		return
	}
	spans := tracer.FinishedSpans()
	ids := mapContexts(spans)
	assert.Equal(t, []int{ids[0], ids[0], ids[0], ids[0]}, ids, "spans share a trace id")
	assert.Equal(t, "echo", spans[0].OperationName, "span has correct operation name")
	assert.Equal(t, "echo", spans[1].OperationName, "span has correct operation name")
	assert.Equal(t, "echoecho", spans[2].OperationName, "span has correct operation name")
	assert.Equal(t, "echoecho", spans[3].OperationName, "span has correct operation name")
}

func mapContexts(spans []*mocktracer.MockSpan) []int {
	ids := make([]int, len(spans))
	for i, span := range spans {
		ids[i] = span.Context().(mocktracer.MockSpanContext).TraceID
	}
	return ids
}
