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
	"io"
	"io/ioutil"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/yarpctest/api"
	"go.uber.org/yarpc/x/yarpctest/types"
)

// EchoHandler is a Unary Handler that will echo the body of the request
// into the response.
func EchoHandler(mw ...api.UnaryInboundMiddleware) *types.UnaryHandler {
	return &types.UnaryHandler{Handler: newEchoHandler(), Middleware: mw}
}

func newEchoHandler() api.UnaryHandler {
	return api.UnaryHandlerFunc(
		func(_ context.Context, req *transport.Request, resw transport.ResponseWriter) error {
			_, err := io.Copy(resw, req.Body)
			return err
		},
	)
}

// StaticHandler will always return the same response.
func StaticHandler(msg string, mw ...api.UnaryInboundMiddleware) *types.UnaryHandler {
	return &types.UnaryHandler{Handler: newStaticHandler(msg), Middleware: mw}
}

func newStaticHandler(msg string) api.UnaryHandler {
	return api.UnaryHandlerFunc(
		func(_ context.Context, _ *transport.Request, resw transport.ResponseWriter) error {
			_, err := io.WriteString(resw, msg)
			return err
		},
	)
}

// ErrorHandler will always return an Error.
func ErrorHandler(err error, mw ...api.UnaryInboundMiddleware) *types.UnaryHandler {
	return &types.UnaryHandler{Handler: newErrorHandler(err), Middleware: mw}
}

func newErrorHandler(err error) api.UnaryHandler {
	return api.UnaryHandlerFunc(
		func(context.Context, *transport.Request, transport.ResponseWriter) error {
			return err
		},
	)
}

// EchoHandlerWithPrefix will echo the request it receives into the
// response, but, it will insert a prefix in front of the message.
func EchoHandlerWithPrefix(prefix string, mw ...api.UnaryInboundMiddleware) *types.UnaryHandler {
	return &types.UnaryHandler{Handler: newEchoHandlerWithPrefix(prefix), Middleware: mw}
}

func newEchoHandlerWithPrefix(prefix string) api.UnaryHandler {
	return api.UnaryHandlerFunc(
		func(_ context.Context, req *transport.Request, resw transport.ResponseWriter) error {
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return err
			}
			newMsg := prefix + string(body)
			_, err = io.WriteString(resw, newMsg)
			return err
		},
	)
}

// OrderedRequestHandler will execute a series of Handlers in the order they
// are passed in.  If the number of requests does not match, it will return an
// unknown error.
func OrderedRequestHandler(options ...api.HandlerOption) *types.OrderedHandler {
	opts := api.HandlerOpts{}
	for _, option := range options {
		option.ApplyHandler(&opts)
	}
	return &types.OrderedHandler{
		Handlers: opts.Handlers,
	}
}
