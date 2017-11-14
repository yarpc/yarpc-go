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

package yarpctest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
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
					p.Port("1"),
					Proc(Name("echo"), EchoHandler()),
				),
			),
			requests: Actions(
				HTTPRequest(
					p.Port("1"),
					Body("test body"),
					Service("myservice"),
					Procedure("echo"),
					ExpectRespBody("test body"),
				),
			),
		},
		{
			name: "tchannel to tchannel request",
			services: Lifecycles(
				TChannelService(
					Name("myservice"),
					p.Port("2"),
					Proc(Name("echo"), EchoHandler()),
				),
			),
			requests: Actions(
				TChannelRequest(
					p.Port("2"),
					Body("test body"),
					Service("myservice"),
					Procedure("echo"),
					ExpectRespBody("test body"),
				),
			),
		},
		{
			name: "grpc to grpc request",
			services: Lifecycles(
				GRPCService(
					Name("myservice"),
					p.Port("3"),
					Proc(Name("echo"), EchoHandler()),
				),
			),
			requests: Actions(
				GRPCRequest(
					p.Port("3"),
					Body("test body"),
					Service("myservice"),
					Procedure("echo"),
					ExpectRespBody("test body"),
				),
			),
		},
		{
			name: "response errors",
			services: Lifecycles(
				HTTPService(
					Name("myservice"),
					p.Port("4-http"),
					Proc(
						Name("error"),
						ErrorHandler(
							errors.New("error from myservice"),
						),
					),
				),
				TChannelService(
					Name("myotherservice"),
					p.Port("4-tch"),
					Proc(Name("error"), ErrorHandler(errors.New("error from myotherservice"))),
				),
				GRPCService(
					Name("myotherservice2"),
					p.Port("4-grpc"),
					Proc(Name("error"), ErrorHandler(errors.New("error from myotherservice2"))),
				),
			),
			requests: Actions(
				HTTPRequest(
					p.Port("4-http"),
					Service("myservice"),
					Procedure("error"),
					ExpectError("error from myservice"),
				),
				TChannelRequest(
					p.Port("4-tch"),
					Service("myotherservice"),
					Procedure("error"),
					ExpectError("error from myotherservice"),
				),
				GRPCRequest(
					p.Port("4-grpc"),
					Service("myotherservice2"),
					Procedure("error"),
					ExpectError("error from myotherservice2"),
				),
			),
		},
		{
			name: "ordered requests",
			services: Lifecycles(
				HTTPService(
					Name("myservice"),
					p.Port("5"),
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
					p.Port("5"),
					Service("myservice"),
					Procedure("proc"),
					ShardKey("ignoreme"),
					ExpectError(yarpcerrors.InternalErrorf("internal error").Error()),
				),
				HTTPRequest(
					p.Port("5"),
					Service("myservice"),
					Procedure("proc"),
					ExpectRespBody("success"),
				),
				HTTPRequest(
					p.Port("5"),
					Service("myservice"),
					Procedure("proc"),
					Body("hello"),
					ExpectRespBody("echo: hello"),
				),
				HTTPRequest(
					p.Port("5"),
					Service("myservice"),
					Procedure("proc"),
					GiveAndExpectLargeBodyIsEchoed(1<<17),
				),
			),
		},
		{
			name: "ordered request headers",
			services: Lifecycles(
				HTTPService(
					Name("myservice"),
					p.Port("6"),
					Proc(
						Name("proc"),
						OrderedRequestHandler(
							ErrorHandler(
								yarpcerrors.InternalErrorf("internal error"),
								ExpectHeader("key1", "val1"),
								ExpectHeader("key2", "val2"),
								WithHeader("resp_key1", "resp_val1"),
								WithHeader("resp_key2", "resp_val2"),
							),
							StaticHandler(
								"success",
								ExpectHeader("successKey", "successValue"),
								WithHeader("responseKey", "responseValue"),
							),
						),
					),
				),
			),
			requests: Actions(
				HTTPRequest(
					p.Port("6"),
					Service("myservice"),
					Procedure("proc"),
					ShardKey("ignoreme"),
					WithHeader("key1", "val1"),
					WithHeader("key2", "val2"),
					ExpectError(yarpcerrors.InternalErrorf("internal error").Error()),
				),
				HTTPRequest(
					p.Port("6"),
					Service("myservice"),
					Procedure("proc"),
					WithHeader("successKey", "successValue"),
					ExpectRespBody("success"),
					ExpectHeader("responseKey", "responseValue"),
				),
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
