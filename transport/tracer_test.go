// Copyright (c) 2016 Uber Technologies, Inc.
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/http"
	ytchannel "go.uber.org/yarpc/transport/tchannel"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

type echoReqBody struct{}
type echoResBody struct{}

type handler struct {
	client json.Client
	t      *testing.T
}

func (h handler) register(dispatcher yarpc.Dispatcher) {
	dispatcher.Register(json.Procedure("echo", h.handleEcho))
	dispatcher.Register(json.Procedure("echoecho", h.handleEchoEcho))
}

func (h handler) handleEcho(ctx context.Context, reqMeta yarpc.ReqMeta, reqBody *echoReqBody) (*echoResBody, yarpc.ResMeta, error) {
	h.assertBaggage(ctx)
	return &echoResBody{}, nil, nil
}

func (h handler) handleEchoEcho(ctx context.Context, reqMeta yarpc.ReqMeta, reqBody *echoReqBody) (*echoResBody, yarpc.ResMeta, error) {
	h.assertBaggage(ctx)
	var resBody echoResBody
	_, err := h.client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echo"),
		reqBody,
		&resBody,
	)
	if err != nil {
		return nil, nil, err
	}
	return &resBody, nil, nil
}

func (h handler) echo(ctx context.Context) error {
	var resBody echoResBody
	_, err := h.client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echo"),
		&echoReqBody{},
		&resBody,
	)
	return err
}

func (h handler) echoEcho(ctx context.Context) error {
	var resBody echoResBody
	_, err := h.client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echoecho"),
		&echoReqBody{},
		&resBody,
	)
	return err
}

func (h handler) createContextWithBaggage(tracer opentracing.Tracer) (context.Context, func()) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)

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

func createHTTPDispatcher(tracer opentracing.Tracer) yarpc.Dispatcher {
	// TODO: Use port 0 once https://github.com/yarpc/yarpc-go/issues/381 is
	// fixed.
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			http.NewInbound(":18080"),
		},
		Outbounds: transport.Outbounds{
			"yarpc-test": http.NewOutbound("http://127.0.0.1:18080"),
		},
		Tracer: tracer,
	})

	return dispatcher
}

func createTChannelDispatcher(tracer opentracing.Tracer, t *testing.T) yarpc.Dispatcher {
	// Establish the TChannel
	ch, err := tchannel.NewChannel("yarpc-test", &tchannel.ChannelOptions{
		Tracer: tracer,
	})
	assert.NoError(t, err)
	hp := "127.0.0.1:4040"
	ch.ListenAndServe(hp)

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			ytchannel.NewInbound(ch),
		},
		Outbounds: transport.Outbounds{
			"yarpc-test": ytchannel.NewOutbound(ch, ytchannel.HostPort(hp)),
		},
		Tracer: tracer,
	})

	return dispatcher
}

func TestHTTPTracer(t *testing.T) {
	tracer := mocktracer.New()
	dispatcher := createHTTPDispatcher(tracer)

	client := json.New(dispatcher.Channel("yarpc-test"))
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
	tracer := mocktracer.New()
	dispatcher := createTChannelDispatcher(tracer, t)

	client := json.New(dispatcher.Channel("yarpc-test"))
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

func TestHTTPTracerDepth2(t *testing.T) {
	tracer := mocktracer.New()
	dispatcher := createHTTPDispatcher(tracer)

	client := json.New(dispatcher.Channel("yarpc-test"))
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
	tracer := mocktracer.New()
	dispatcher := createTChannelDispatcher(tracer, t)

	client := json.New(dispatcher.Channel("yarpc-test"))
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
