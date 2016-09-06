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
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	ytchannel "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

type echoReqBody struct{}
type echoResBody struct{}

func echo(ctx context.Context, reqMeta yarpc.ReqMeta, reqBody *echoReqBody) (*echoResBody, yarpc.ResMeta, error) {
	return &echoResBody{}, nil, nil
}

func AssertDepth1Spans(t *testing.T, tracer *mocktracer.MockTracer) {
	assert.Equal(t, 2, len(tracer.FinishedSpans()), "generates inbound and outband spans")
	if len(tracer.FinishedSpans()) != 2 {
		return
	}
	spans := tracer.FinishedSpans()
	parent := spans[0]
	child := spans[1]
	// parentctx := parent.Context().(mocktracer.MockSpanContext)
	// childctx := child.Context().(mocktracer.MockSpanContext)
	// assert.Equal(t, parentctx.TraceID, childctx.TraceID)
	// Whether the parent and child have the same span id is an implementation
	// detail of the tracer.
	assert.Equal(t, "echo", parent.OperationName, "span has correct operation name")
	assert.Equal(t, "echo", child.OperationName, "span has correct operation name")
}

func TestHttpTracer(t *testing.T) {
	tracer := mocktracer.New()
	rpc := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			http.NewInbound(":8080"),
		},
		Outbounds: transport.Outbounds{
			"yarpc-test": http.NewOutbound("http://127.0.0.1:8080"),
		},
		Tracer: tracer,
	})

	json.Register(rpc, json.Procedure("echo", echo))

	client := json.New(rpc.Channel("yarpc-test"))

	rpc.Start()
	defer rpc.Stop()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	var resBody echoResBody
	_, err := client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echo"),
		&echoReqBody{},
		&resBody,
	)
	assert.NoError(t, err)

	AssertDepth1Spans(t, tracer)
}

func TestTChannelTracer(t *testing.T) {
	tracer := mocktracer.New()

	// Establish the TChannel
	ch, err := tchannel.NewChannel("yarpc-test", &tchannel.ChannelOptions{
		Tracer: tracer,
	})
	assert.NoError(t, err)
	hp := "127.0.0.1:4040"
	ch.ListenAndServe(hp)

	rpc := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			ytchannel.NewInbound(ch),
		},
		Outbounds: transport.Outbounds{
			"yarpc-test": ytchannel.NewOutbound(ch, ytchannel.HostPort(hp)),
		},
		Tracer: tracer,
	})

	json.Register(rpc, json.Procedure("echo", echo))

	client := json.New(rpc.Channel("yarpc-test"))

	rpc.Start()
	defer rpc.Stop()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	var resBody echoResBody
	_, err = client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echo"),
		&echoReqBody{},
		&resBody,
	)
	assert.NoError(t, err)

	AssertDepth1Spans(t, tracer)
}

func TestHttpTracerDepth2(t *testing.T) {
	tracer := mocktracer.New()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	span := tracer.StartSpan("test")
	// no defer span.Finish()
	span.SetBaggageItem("weapon", "knife")
	ctx = opentracing.ContextWithSpan(ctx, span)

	rpc := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			http.NewInbound(":8080"),
		},
		Outbounds: transport.Outbounds{
			"yarpc-test": http.NewOutbound("http://127.0.0.1:8080"),
		},
		Tracer: tracer,
	})

	client := json.New(rpc.Channel("yarpc-test"))

	echo := func(ctx context.Context, reqMeta yarpc.ReqMeta, reqBody *echoReqBody) (*echoResBody, yarpc.ResMeta, error) {
		span := opentracing.SpanFromContext(ctx)
		weapon := span.BaggageItem("weapon")
		assert.Equal(t, "knife", weapon, "baggage should propagate")
		return &echoResBody{}, nil, nil
	}

	echoecho := func(ctx context.Context, reqMeta yarpc.ReqMeta, reqBody *echoReqBody) (*echoResBody, yarpc.ResMeta, error) {
		var resBody echoResBody
		_, err := client.Call(
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

	json.Register(rpc, json.Procedure("echo", echo))
	json.Register(rpc, json.Procedure("echoecho", echoecho))

	rpc.Start()
	defer rpc.Stop()

	var resBody echoResBody
	_, err := client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echoecho"),
		&echoReqBody{},
		&resBody,
	)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(tracer.FinishedSpans()), "generates inbound and outband spans")
	if len(tracer.FinishedSpans()) != 4 {
		return
	}
	spans := tracer.FinishedSpans()
	assert.Equal(t, "echo", spans[0].OperationName, "span has correct operation name")
	assert.Equal(t, "echo", spans[1].OperationName, "span has correct operation name")
	assert.Equal(t, "echoecho", spans[2].OperationName, "span has correct operation name")
	assert.Equal(t, "echoecho", spans[3].OperationName, "span has correct operation name")
}

func TestTChannelTracerDepth2(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	tracer := mocktracer.New()

	span := tracer.StartSpan("test")
	// no defer span.Finish()
	span.SetBaggageItem("weapon", "knife")
	ctx = opentracing.ContextWithSpan(ctx, span)

	// Establish the TChannel
	ch, err := tchannel.NewChannel("yarpc-test", &tchannel.ChannelOptions{
		Tracer: tracer,
	})
	assert.NoError(t, err)
	hp := "127.0.0.1:4040"
	ch.ListenAndServe(hp)

	rpc := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			ytchannel.NewInbound(ch),
		},
		Outbounds: transport.Outbounds{
			"yarpc-test": ytchannel.NewOutbound(ch, ytchannel.HostPort(hp)),
		},
		Tracer: tracer,
	})

	client := json.New(rpc.Channel("yarpc-test"))

	echo := func(ctx context.Context, reqMeta yarpc.ReqMeta, reqBody *echoReqBody) (*echoResBody, yarpc.ResMeta, error) {
		span := opentracing.SpanFromContext(ctx)
		weapon := span.BaggageItem("weapon")
		assert.Equal(t, "knife", weapon, "baggage should propagate")
		return &echoResBody{}, nil, nil
	}

	echoecho := func(ctx context.Context, reqMeta yarpc.ReqMeta, reqBody *echoReqBody) (*echoResBody, yarpc.ResMeta, error) {
		var resBody echoResBody
		_, err := client.Call(
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

	json.Register(rpc, json.Procedure("echo", echo))
	json.Register(rpc, json.Procedure("echoecho", echoecho))

	rpc.Start()
	defer rpc.Stop()

	var resBody echoResBody
	_, err = client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echoecho"),
		&echoReqBody{},
		&resBody,
	)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(tracer.FinishedSpans()), "generates inbound and outband spans")
	if len(tracer.FinishedSpans()) != 4 {
		return
	}
	spans := tracer.FinishedSpans()
	assert.Equal(t, "echo", spans[0].OperationName, "span has correct operation name")
	assert.Equal(t, "echo", spans[1].OperationName, "span has correct operation name")
	assert.Equal(t, "echoecho", spans[2].OperationName, "span has correct operation name")
	assert.Equal(t, "echoecho", spans[3].OperationName, "span has correct operation name")
}
