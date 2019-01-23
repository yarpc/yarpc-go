// Copyright (c) 2019 Uber Technologies, Inc.
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
	"strings"
	"time"

	"github.com/crossdock/crossdock-go"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	server "go.uber.org/yarpc/internal/crossdock/server/yarpc"
	"go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"
)

// Run verifies that opentracing context is propagated across multiple hops.
//
// Behavior parameters:
//
// - ctxclient: Address of this client.
//
// - ctxserver: Address of the crossdock test subject server.
//
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

			jsonClient := json.New(dispatcher.ClientConfig("yarpc-test"))
			for name, handler := range tt.handlers {
				handler.SetClient(jsonClient)
				handler.SetTransport(tconfig)
				dispatcher.Register(json.Procedure(name, handler.Handle))
			}

			fatals.NoError(dispatcher.Start(), "%v: Dispatcher failed to start", tt.desc)
			defer dispatcher.Stop()

			ctx := context.Background()
			if tt.initCtx != nil {
				ctx = tt.initCtx
			}
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			var resp js.RawMessage
			err := jsonClient.Call(
				ctx,
				"phone",
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
	Handle(context.Context, interface{}) (interface{}, error)
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

func (h *singleHopHandler) Handle(ctx context.Context, body interface{}) (interface{}, error) {
	assertBaggageMatches(ctx, h.t, h.wantBaggage)
	call := yarpc.CallFromContext(ctx)
	for _, k := range call.HeaderNames() {
		if err := call.WriteResponseHeader(k, call.Header(k)); err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{}, nil
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

func (h *multiHopHandler) Handle(ctx context.Context, body interface{}) (interface{}, error) {
	if h.phoneClient == nil {
		panic("call SetClient() and SetTransport() first")
	}

	assertBaggageMatches(ctx, h.t, h.wantBaggage)

	span := opentracing.SpanFromContext(ctx)
	for key, value := range h.addBaggage {
		span.SetBaggageItem(key, value)
	}
	ctx = opentracing.ContextWithSpan(ctx, span)

	var (
		opts            []yarpc.CallOption
		phoneResHeaders map[string]string
	)

	call := yarpc.CallFromContext(ctx)

	for _, k := range call.HeaderNames() {
		opts = append(opts, yarpc.WithHeader(k, call.Header(k)))
	}
	opts = append(opts, yarpc.ResponseHeaders(&phoneResHeaders))

	var resp js.RawMessage
	err := h.phoneClient.Call(
		ctx,
		"phone",
		&server.PhoneRequest{
			Service:   "ctxclient",
			Procedure: h.phoneCallTo,
			Transport: h.phoneCallTransport,
			Body:      &js.RawMessage{'{', '}'},
		}, &resp, opts...)

	for k, v := range phoneResHeaders {
		if err := call.WriteResponseHeader(k, v); err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{}, err
}

func buildDispatcher(t crossdock.T) (dispatcher *yarpc.Dispatcher, tconfig server.TransportConfig) {
	fatals := crossdock.Fatals(t)

	self := t.Param("ctxclient")
	subject := t.Param("ctxserver")

	fatals.NotEmpty(self, "ctxclient is required")
	fatals.NotEmpty(subject, "ctxserver is required")
	nextHop := nextHopTransport(t)

	httpTransport := http.NewTransport()
	tchannelTransport, err := tch.NewChannelTransport(tch.ListenAddr(":8087"), tch.ServiceName("ctxclient"))
	fatals.NoError(err, "Failed to build ChannelTransport")

	// Outbound to use for this hop.
	var outbound transport.UnaryOutbound

	trans := t.Param(params.Transport)
	switch trans {
	case "http":
		outbound = httpTransport.NewSingleOutbound(fmt.Sprintf("http://%s:8081", subject))
	case "tchannel":
		outbound = tchannelTransport.NewSingleOutbound(fmt.Sprintf("%s:8082", subject))
	default:
		fatals.Fail("", "unknown transport %q", trans)
	}

	nextTrans, ok := nextHop[trans]
	fatals.True(ok, "no transport specified after %q", trans)

	t.Tag("nextTransport", nextTrans)
	switch nextTrans {
	case "http":
		tconfig.HTTP = &server.HTTPTransport{Host: self, Port: 8086}
	case "tchannel":
		tconfig.TChannel = &server.TChannelTransport{Host: self, Port: 8087}
	default:
		fatals.Fail("", "unknown transport %q after transport %q", nextTrans, trans)
	}

	dispatcher = yarpc.NewDispatcher(yarpc.Config{
		Name: "ctxclient",
		Inbounds: yarpc.Inbounds{
			tchannelTransport.NewInbound(),
			httpTransport.NewInbound(":8086"),
		},
		Outbounds: yarpc.Outbounds{
			"yarpc-test": {
				Unary: outbound,
			},
		},
	})

	return dispatcher, tconfig
}

// nextHopTransport returns a map from current transport to next transport
// based on transports defined in ctxavailabletransports.
//
// If ctxavailabletransports was [x, y, z], this returns {x: y, y: z, z: y}.
// So if transport is y, the next hop should use z.
func nextHopTransport(t crossdock.T) map[string]string {
	ts := strings.Split(t.Param("ctxavailabletransports"), ";")
	crossdock.Fatals(t).NotEmpty(ts, "ctxavailabletransports is required")

	m := make(map[string]string, len(ts))
	for i, transport := range ts {
		m[transport] = ts[i%len(ts)]
	}
	return m
}
