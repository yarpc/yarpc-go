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

package yarpctest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestServiceRouting(t *testing.T) {
	p := NewPortProvider(t)
	tests := []struct {
		name     string
		services Lifecycle
		requests Action
	}{
		{
			name: "http to http request",
			services: Lifecycles(
				HTTPService(
					Name("myservice"),
					p.NamedPort("1"),
					Proc(Name("echo"), EchoHandler()),
				),
			),
			requests: ConcurrentAction(
				RepeatAction(
					HTTPRequest(
						p.NamedPort("1"),
						GiveTimeout(testtime.Second),
						Body("test body"),
						Service("myservice"),
						Procedure("echo"),
						WantRespBody("test body"),
					),
					10,
				),
				3,
			),
		},
		{
			name: "tchannel to tchannel request",
			services: Lifecycles(
				TChannelService(
					Name("myservice"),
					p.NamedPort("2"),
					Proc(Name("echo"), EchoHandler()),
				),
			),
			requests: ConcurrentAction(
				RepeatAction(
					TChannelRequest(
						p.NamedPort("2"),
						GiveTimeout(testtime.Second),
						Body("test body"),
						Service("myservice"),
						Procedure("echo"),
						WantRespBody("test body"),
					),
					10,
				),
				3,
			),
		},
		{
			name: "grpc to grpc request",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.NamedPort("3"),
					Proc(Name("echo"), EchoHandler()),
				),
			),
			requests: ConcurrentAction(
				RepeatAction(
					GRPCRequest(
						p.NamedPort("3"),
						GiveTimeout(testtime.Second),
						Body("test body"),
						Service("myservice"),
						Procedure("echo"),
						WantRespBody("test body"),
					),
					10,
				),
				3,
			),
		},
		{
			name: "response errors",
			services: Lifecycles(
				HTTPService(
					Name("myservice"),
					p.NamedPort("4-http"),
					Proc(
						Name("error"),
						ErrorHandler(
							errors.New("error from myservice"),
						),
					),
				),
				TChannelService(
					Name("myotherservice"),
					p.NamedPort("4-tch"),
					Proc(Name("error"), ErrorHandler(errors.New("error from myotherservice"))),
				),
				GRPCService(
					Name("myotherservice2"),
					p.NamedPort("4-grpc"),
					Proc(Name("error"), ErrorHandler(errors.New("error from myotherservice2"))),
				),
			),
			requests: Actions(
				HTTPRequest(
					p.NamedPort("4-http"),
					Service("myservice"),
					Procedure("error"),
					WantError("error from myservice"),
				),
				TChannelRequest(
					p.NamedPort("4-tch"),
					Service("myotherservice"),
					Procedure("error"),
					WantError("error from myotherservice"),
				),
				GRPCRequest(
					p.NamedPort("4-grpc"),
					Service("myotherservice2"),
					Procedure("error"),
					WantError("error from myotherservice2"),
				),
			),
		},
		{
			name: "ordered requests",
			services: Lifecycles(
				HTTPService(
					Name("myservice"),
					p.NamedPort("5"),
					Proc(
						Name("proc"),
						OrderedRequestHandler(
							ErrorHandler(yarpcerrors.InternalErrorf("internal error")),
							StaticHandler("success"),
							EchoHandlerWithPrefix("echo: "),
							EchoHandler(),
						),
					),
				),
			),
			requests: Actions(
				HTTPRequest(
					p.NamedPort("5"),
					Service("myservice"),
					Procedure("proc"),
					ShardKey("ignoreme"),
					WantError(yarpcerrors.InternalErrorf("internal error").Error()),
				),
				HTTPRequest(
					p.NamedPort("5"),
					Service("myservice"),
					Procedure("proc"),
					WantRespBody("success"),
				),
				HTTPRequest(
					p.NamedPort("5"),
					Service("myservice"),
					Procedure("proc"),
					Body("hello"),
					WantRespBody("echo: hello"),
				),
				HTTPRequest(
					p.NamedPort("5"),
					Service("myservice"),
					Procedure("proc"),
					GiveAndWantLargeBodyIsEchoed(1<<17),
				),
			),
		},
		{
			name: "ordered request headers",
			services: Lifecycles(
				HTTPService(
					Name("myservice"),
					p.NamedPort("6"),
					Proc(
						Name("proc"),
						OrderedRequestHandler(
							ErrorHandler(
								yarpcerrors.InternalErrorf("internal error"),
								WantHeader("key1", "val1"),
								WantHeader("key2", "val2"),
								WithHeader("resp_key1", "resp_val1"),
								WithHeader("resp_key2", "resp_val2"),
							),
							StaticHandler(
								"success",
								WantHeader("successKey", "successValue"),
								WithHeader("responseKey", "responseValue"),
							),
						),
					),
				),
			),
			requests: Actions(
				HTTPRequest(
					p.NamedPort("6"),
					Service("myservice"),
					Procedure("proc"),
					ShardKey("ignoreme"),
					WithHeader("key1", "val1"),
					WithHeader("key2", "val2"),
					WantError(yarpcerrors.InternalErrorf("internal error").Error()),
				),
				HTTPRequest(
					p.NamedPort("6"),
					Service("myservice"),
					Procedure("proc"),
					WithHeader("successKey", "successValue"),
					WantRespBody("success"),
					WantHeader("responseKey", "responseValue"),
				),
			),
		},
		{
			name: "hardcoded peer",
			services: Lifecycles(
				TChannelService(
					Name("myservice"),
					Port(54321),
					Proc(Name("echo"), EchoHandler()),
				),
			),
			requests: ConcurrentAction(
				RepeatAction(
					TChannelRequest(
						Port(54321),
						Body("test body"),
						Service("myservice"),
						Procedure("echo"),
						WantRespBody("test body"),
					),
					10,
				),
				3,
			),
		},
		{
			name: "hardcoded peer (same as above, testing reuse)",
			services: Lifecycles(
				TChannelService(
					Name("myservice"),
					Port(54321),
					Proc(Name("echo"), EchoHandler()),
				),
			),
			requests: ConcurrentAction(
				RepeatAction(
					TChannelRequest(
						Port(54321),
						Body("test body"),
						Service("myservice"),
						Procedure("echo"),
						WantRespBody("test body"),
					),
					10,
				),
				3,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.services.Start(t))
			defer func() { require.NoError(t, tt.services.Stop(t)) }()
			tt.requests.Run(t)
		})
	}
}

func TestUnaryOutboundMiddleware(t *testing.T) {
	p := NewPortProvider(t)

	const (
		service            = "service"
		correctProcedure   = "correct-procedure"
		incorrectProcedure = "inccorect"
	)

	mw := middleware.UnaryOutboundFunc(
		func(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {
			// fix procedure name
			req.Procedure = correctProcedure
			return next.Call(ctx, req)
		})

	tests := []struct {
		name    string
		service Lifecycle
		request Action
	}{
		{
			name: "HTTP",
			service: HTTPService(
				Name(service),
				Proc(Name(correctProcedure), EchoHandler()),
				p.NamedPort("http"),
			),
			request: HTTPRequest(
				Service(service),
				Procedure(incorrectProcedure),
				UnaryOutboundMiddleware(mw),
				p.NamedPort("http"),
			),
		},
		{
			name: "TChannel",
			service: TChannelService(
				Name(service),
				Proc(Name(correctProcedure), EchoHandler()),
				p.NamedPort("TChannel"),
			),
			request: TChannelRequest(
				Service(service),
				Procedure(incorrectProcedure),
				UnaryOutboundMiddleware(mw),
				p.NamedPort("TChannel"),
			),
		},
		{
			name: "gRPC",
			service: GRPCService(
				Name(service),
				Proc(Name(correctProcedure), EchoHandler()),
				p.NamedPort("grpc"),
			),
			request: GRPCRequest(
				Service(service),
				Procedure(incorrectProcedure),
				UnaryOutboundMiddleware(mw),
				p.NamedPort("grpc"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.service.Start(t))
			defer func() { assert.NoError(t, tt.service.Stop(t)) }()
			tt.request.Run(t)
		})
	}
}
