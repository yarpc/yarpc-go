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
	js "encoding/json"
	"fmt"
	"time"

	"github.com/crossdock/crossdock-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/crossdock/client/params"
	server "go.uber.org/yarpc/crossdock/server/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/internal/baggage"
	"go.uber.org/yarpc/transport"
	ht "go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"

	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

// Run verifies that context is propagated across multiple hops.
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
					wantBaggage: transport.Headers{},
				},
			},
		},
		{
			desc:    "existing baggage",
			initCtx: yarpc.WithBaggage(context.Background(), "token", "42"),
			handlers: map[string]handler{
				"hello": &singleHopHandler{
					t:           t,
					wantBaggage: transport.NewHeaders().With("token", "42"),
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
					addBaggage:  transport.NewHeaders().With("x", "1"),
					wantBaggage: transport.Headers{},
				},
				"two": &multiHopHandler{
					t:           t,
					phoneCallTo: "three",
					addBaggage:  transport.NewHeaders().With("y", "2"),
					wantBaggage: transport.NewHeaders().With("x", "1"),
				},
				"three": &singleHopHandler{
					t:           t,
					wantBaggage: transport.NewHeaders().With("x", "1").With("y", "2"),
				},
			},
		},
		{
			desc:      "add baggage: existing baggage",
			initCtx:   yarpc.WithBaggage(context.Background(), "token", "123"),
			procedure: "one",
			handlers: map[string]handler{
				"one": &multiHopHandler{
					t:           t,
					phoneCallTo: "two",
					addBaggage:  transport.NewHeaders().With("hello", "world"),
					wantBaggage: transport.NewHeaders().With("token", "123"),
				},
				"two": &singleHopHandler{
					t:           t,
					wantBaggage: transport.NewHeaders().With("token", "123").With("hello", "world"),
				},
			},
		},
		{
			desc:      "overwrite baggage",
			initCtx:   yarpc.WithBaggage(context.Background(), "x", "1"),
			procedure: "one",
			handlers: map[string]handler{
				"one": &multiHopHandler{
					t:           t,
					phoneCallTo: "two",
					addBaggage:  transport.NewHeaders().With("x", "2").With("y", "3"),
					wantBaggage: transport.NewHeaders().With("x", "1"),
				},
				"two": &multiHopHandler{
					t:           t,
					phoneCallTo: "three",
					addBaggage:  transport.NewHeaders().With("y", "4"),
					wantBaggage: transport.NewHeaders().With("x", "2").With("y", "3"),
				},
				"three": &singleHopHandler{
					t:           t,
					wantBaggage: transport.NewHeaders().With("x", "2").With("y", "4"),
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
				dispatcher.Register(json.Procedure(name, handler.Handle))
			}

			ctx := context.Background()
			if tt.initCtx != nil {
				ctx = tt.initCtx
			}
			ctx, _ = context.WithTimeout(ctx, time.Second)

			var resp js.RawMessage
			_, err := jsonClient.Call(
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

func assertBaggageMatches(t crossdock.T, ctx context.Context, want transport.Headers) bool {
	assert := crossdock.Assert(t)
	got := baggage.FromContext(ctx)

	if want.Len() == 0 {
		// len check to handle nil vs empty cases gracefully.
		return assert.Equal(0, got.Len(), "baggage must be empty: %v", got)
	}

	return assert.Equal(want, got, "baggage must match")
}

// singleHopHandler provides a JSON handler which verifies that it receives the
// specified baggage.
type singleHopHandler struct {
	t           crossdock.T
	wantBaggage transport.Headers
}

func (*singleHopHandler) SetClient(json.Client)               {}
func (*singleHopHandler) SetTransport(server.TransportConfig) {}

func (h *singleHopHandler) Handle(ctx context.Context, reqMeta yarpc.ReqMeta, body interface{}) (interface{}, yarpc.ResMeta, error) {
	assertBaggageMatches(h.t, ctx, h.wantBaggage)
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

	addBaggage  transport.Headers
	wantBaggage transport.Headers
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

	assertBaggageMatches(h.t, ctx, h.wantBaggage)
	for key, value := range h.addBaggage.Items() {
		ctx = yarpc.WithBaggage(ctx, key, value)
	}

	var resp js.RawMessage
	phoneResMeta, err := h.phoneClient.Call(
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

	var outbound transport.Outbound
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
		Outbounds: transport.Outbounds{"yarpc-test": outbound},
	})

	return dispatcher, tconfig
}
