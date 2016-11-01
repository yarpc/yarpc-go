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

package ctxpropagation

import (
	"context"
	js "encoding/json"
	"fmt"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/crossdock/client/params"
	server "go.uber.org/yarpc/crossdock/server/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/transport"
	ht "go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"

	"github.com/crossdock/crossdock-go"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/tchannel-go"
)

// Run verifies that opentracing context is propagated across multiple hops.
//
// Behavior parameters:
//
// - ctxclient: Address of this client.
// - ctxserver: Address of the crossdock test subject server.
// - transport: The transport to make requests to the test subject with.
//
// This behavior sets up a server in-process which the Phone procedure on the
// test subject is responsible for calling.
//
// Outgoing calls to the Phone procedure will be made using the transport
// specified as a parameter, and incoming calls from the Phone procedure will
// be received over a different transport.
func Run(t crossdock.T) {
	checks := crossdock.Checks(t)
	assert := crossdock.Assert(t)
	fatals := crossdock.Fatals(t)

	tracer, closer := jaeger.NewTracer("crossdock", jaeger.NewConstSampler(true), jaeger.NewNullReporter())
	defer closer.Close()
	opentracing.InitGlobalTracer(tracer)

	tests := []struct {
		desc      string
		initCtx   context.Context
		handlers  map[string]handler
		procedure string
	}{
		{
			desc: "no baggage",
			handlers: map[string]handler{
				"hello": &singleHopHandler{
					t:           t,
					wantBaggage: map[string]string{},
				},
			},
		},
		{
			desc: "existing baggage",
			initCtx: func() context.Context {
				span := opentracing.GlobalTracer().StartSpan("existing baggage")
				span.SetBaggageItem("token", "42")
				return opentracing.ContextWithSpan(context.Background(), span)
			}(),
			handlers: map[string]handler{
				"hello": &singleHopHandler{
					t:           t,
					wantBaggage: map[string]string{"token": "42"},
				},
			},
		},
		{
			desc:      "add baggage",
			procedure: "one",
			handlers: map[string]handler{
				"one": &multiHopHandler{
					t:           t,
					phoneCallTo: "two",
					addBaggage:  map[string]string{"x": "1"},
					wantBaggage: map[string]string{},
				},
				"two": &multiHopHandler{
					t:           t,
					phoneCallTo: "three",
					addBaggage:  map[string]string{"y": "2"},
					wantBaggage: map[string]string{"x": "1"},
				},
				"three": &singleHopHandler{
					t:           t,
					wantBaggage: map[string]string{"x": "1", "y": "2"},
				},
			},
		},
		{
			desc: "add baggage: existing baggage",
			initCtx: func() context.Context {
				span := opentracing.GlobalTracer().StartSpan("existing baggage")
				span.SetBaggageItem("token", "123")
				return opentracing.ContextWithSpan(context.Background(), span)
			}(),
			procedure: "one",
			handlers: map[string]handler{
				"one": &multiHopHandler{
					t:           t,
					phoneCallTo: "two",
					addBaggage:  map[string]string{"hello": "world"},
					wantBaggage: map[string]string{"token": "123"},
				},
				"two": &singleHopHandler{
					t:           t,
					wantBaggage: map[string]string{"token": "123", "hello": "world"},
				},
			},
		},
		{
			desc: "overwrite baggage",
			initCtx: func() context.Context {
				span := opentracing.GlobalTracer().StartSpan("existing baggage")
				span.SetBaggageItem("x", "1")
				return opentracing.ContextWithSpan(context.Background(), span)
			}(),
			procedure: "one",
			handlers: map[string]handler{
				"one": &multiHopHandler{
					t:           t,
					phoneCallTo: "two",
					addBaggage:  map[string]string{"x": "2", "y": "3"},
					wantBaggage: map[string]string{"x": "1"},
				},
				"two": &multiHopHandler{
					t:           t,
					phoneCallTo: "three",
					addBaggage:  map[string]string{"y": "4"},
					wantBaggage: map[string]string{"x": "2", "y": "3"},
				},
				"three": &singleHopHandler{
					t:           t,
					wantBaggage: map[string]string{"x": "2", "y": "4"},
				},
			},
		},
	}

	for _, tt := range tests {
		func() {
			procedure := tt.procedure
			if procedure == "" {
				if !assert.Len(tt.handlers, 1,
					"%v: invalid test: starting procedure must be provided", tt.desc) {
					return
				}
				for k := range tt.handlers {
					procedure = k
				}
			}

			dispatcher, tconfig := buildDispatcher(t)
			fatals.NoError(dispatcher.Start(), "%v: Dispatcher failed to start", tt.desc)
			defer dispatcher.Stop()

			jsonClient := json.New(dispatcher.Channel("yarpc-test"))
			for name, handler := range tt.handlers {
				handler.SetClient(jsonClient)
				handler.SetTransport(tconfig)
				dispatcher.Register(json.UnaryProcedure(name, handler.Handle))
			}

			ctx := context.Background()
			if tt.initCtx != nil {
				ctx = tt.initCtx
			}
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			var resp js.RawMessage
			_, err := jsonClient.CallUnary(
				ctx,
				yarpc.NewReqMeta().Procedure("phone"),
				&server.PhoneRequest{
					Service:   "ctxclient",
					Procedure: procedure,
					Transport: tconfig,
					Body:      &js.RawMessage{'{', '}'},
				}, &resp)

			checks.NoError(err, "%v: request failed", tt.desc)
		}()
	}
}

type handler interface {
	SetClient(json.Client)
	SetTransport(server.TransportConfig)
	Handle(context.Context, yarpc.ReqMeta, interface{}) (interface{}, yarpc.ResMeta, error)
}

func assertBaggageMatches(ctx context.Context, t crossdock.T, want map[string]string) bool {
	assert := crossdock.Assert(t)
	got := getOpenTracingBaggage(ctx)

	if len(want) == 0 {
		// len check to handle nil vs empty cases gracefully.
		return assert.Empty(got, "baggage must be empty: %v", got)
	}

	return assert.Equal(want, got, "baggage must match")
}

func getOpenTracingBaggage(ctx context.Context) map[string]string {
	headers := make(map[string]string)

	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return headers
	}

	spanContext := span.Context()
	if spanContext == nil {
		return headers
	}

	spanContext.ForeachBaggageItem(func(k, v string) bool {
		headers[k] = v
		return true
	})

	return headers
}

// singleHopHandler provides a JSON handler which verifies that it receives the
// specified baggage.
type singleHopHandler struct {
	t           crossdock.T
	wantBaggage map[string]string
}

func (*singleHopHandler) SetClient(json.Client)               {}
func (*singleHopHandler) SetTransport(server.TransportConfig) {}

func (h *singleHopHandler) Handle(ctx context.Context, reqMeta yarpc.ReqMeta, body interface{}) (interface{}, yarpc.ResMeta, error) {
	assertBaggageMatches(ctx, h.t, h.wantBaggage)
	resMeta := yarpc.NewResMeta().Headers(reqMeta.Headers())
	return map[string]interface{}{}, resMeta, nil
}

// multiHopHandler provides a JSON handler which verfiies that it receives the
// specified baggage, adds new baggage to the context, and makes a Phone request
// to the Test Subject, requesting a call to a different procedure.
type multiHopHandler struct {
	t crossdock.T

	phoneClient        json.Client
	phoneCallTo        string
	phoneCallTransport server.TransportConfig

	addBaggage  map[string]string
	wantBaggage map[string]string
}

func (h *multiHopHandler) SetClient(c json.Client) {
	h.phoneClient = c
}

func (h *multiHopHandler) SetTransport(tc server.TransportConfig) {
	h.phoneCallTransport = tc
}

func (h *multiHopHandler) Handle(ctx context.Context, reqMeta yarpc.ReqMeta, body interface{}) (interface{}, yarpc.ResMeta, error) {
	if h.phoneClient == nil {
		panic("call SetClient() and SetTransport() first")
	}

	assertBaggageMatches(ctx, h.t, h.wantBaggage)

	span := opentracing.SpanFromContext(ctx)
	for key, value := range h.addBaggage {
		span.SetBaggageItem(key, value)
	}
	ctx = opentracing.ContextWithSpan(ctx, span)

	var resp js.RawMessage
	phoneResMeta, err := h.phoneClient.CallUnary(
		ctx,
		yarpc.NewReqMeta().Procedure("phone").Headers(reqMeta.Headers()),
		&server.PhoneRequest{
			Service:   "ctxclient",
			Procedure: h.phoneCallTo,
			Transport: h.phoneCallTransport,
			Body:      &js.RawMessage{'{', '}'},
		}, &resp)

	resMeta := yarpc.NewResMeta().Headers(phoneResMeta.Headers())
	return map[string]interface{}{}, resMeta, err
}

func buildDispatcher(t crossdock.T) (dispatcher yarpc.Dispatcher, tconfig server.TransportConfig) {
	fatals := crossdock.Fatals(t)

	self := t.Param("ctxclient")
	subject := t.Param("ctxserver")
	fatals.NotEmpty(self, "ctxclient is required")
	fatals.NotEmpty(subject, "ctxserver is required")

	ch, err := tchannel.NewChannel("ctxclient", nil)
	fatals.NoError(err, "failed to create TChannel")

	var outbound transport.UnaryOutbound
	switch trans := t.Param(params.Transport); trans {
	case "http":
		outbound = ht.NewOutbound(fmt.Sprintf("http://%s:8081", subject))
		tconfig.TChannel = &server.TChannelTransport{Host: self, Port: 8087}
	case "tchannel":
		outbound = tch.NewOutbound(ch, tch.HostPort(fmt.Sprintf("%s:8082", subject)))
		tconfig.HTTP = &server.HTTPTransport{Host: self, Port: 8086}
	default:
		fatals.Fail("", "unknown transport %q", trans)
	}

	dispatcher = yarpc.NewDispatcher(yarpc.Config{
		Name: "ctxclient",
		Inbounds: []transport.Inbound{
			tch.NewInbound(ch, tch.ListenAddr(":8087")),
			ht.NewInbound(":8086"),
		},
		Outbounds: yarpc.Outbounds{
			"yarpc-test": {
				Unary:  outbound,
				Oneway: nil,
			},
		},
	})

	return dispatcher, tconfig
}
