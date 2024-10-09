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

// Package interceptor defines the interceptor interfaces that are used within each transport.
// The package is currently put under internal because we don't allow customized interceptors at this moment.
// Interceptor interfaces are the alias types of middleware interfaces to share the common utility functions.
package interceptor

import (
	"context"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

type (
	// UnaryInbound defines a transport interceptor for `UnaryHandler`s.
	//
	// UnaryInbound interceptor MAY do zero or more of the following: change the
	// context, change the request, call the ResponseWriter, modify the response
	// body by wrapping the ResponseWriter, handle the returned error, call the
	// given handler zero or more times.
	//
	// UnaryInbound interceptor MUST be thread-safe.
	//
	// UnaryInbound interceptor is re-used across requests and MAY be called multiple times
	// for the same request.
	UnaryInbound = middleware.UnaryInbound

	// OnewayInbound defines a transport interceptor for `OnewayHandler`s.
	//
	// OnewayInbound interceptor MAY do zero or more of the following: change the
	// context, change the request, handle the returned error, call the given
	// handler zero or more times.
	//
	// OnewayInbound interceptor MUST be thread-safe.
	//
	// OnewayInbound interceptor is re-used across requests and MAY be called
	// multiple times for the same request.
	OnewayInbound = middleware.OnewayInbound

	// StreamInbound defines a transport interceptor for `StreamHandler`s.
	//
	// StreamInbound interceptor MAY do zero or more of the following: change the
	// stream, handle the returned error, call the given handler zero or more times.
	//
	// StreamInbound interceptor MUST be thread-safe.
	//
	// StreamInbound interceptor is re-used across requests and MAY be called
	// multiple times for the same request.
	StreamInbound = middleware.StreamInbound
)

// UnaryInboundFunc adapts a function into a UnaryInbound middleware.
type UnaryInboundFunc func(context.Context, *transport.Request, transport.ResponseWriter, transport.UnaryHandler) error

// Handle for UnaryInboundFunc.
func (f UnaryInboundFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	return f(ctx, req, resw, h)
}

type unaryHandlerWithMiddleware struct {
	h transport.UnaryHandler
	i UnaryInbound
}

// Handle applies the UnaryInbound middleware to the handler's Handle method.
func (h unaryHandlerWithMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return h.i.Handle(ctx, req, resw, h.h)
}

type nopUnaryInbound struct{}

// Handle simply calls the underlying UnaryHandler without any modifications.
func (nopUnaryInbound) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	return handler.Handle(ctx, req, resw)
}
