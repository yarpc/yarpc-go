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

package middleware

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// UnaryInboundMiddleware defines a transport-level middleware for
// `UnaryHandler`s.
//
// UnaryInboundMiddleware MAY
//
// - change the context
// - change the request
// - call the ResponseWriter
// - modify the response body by wrapping the ResponseWriter
// - handle the returned error
// - call the given handler zero or more times
//
// UnaryInboundMiddleware MUST be thread-safe.
//
// UnaryInboundMiddleware is re-used across requests and MAY be called multiple times
// for the same request.
type UnaryInboundMiddleware interface {
	Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error
}

// NopUnaryInboundMiddleware is a inbound middleware that does not do anything special. It
// simply calls the underlying Handler.
var NopUnaryInboundMiddleware UnaryInboundMiddleware = nopUnaryInboundMiddleware{}

// ApplyUnaryInboundMiddleware applies the given InboundMiddleware to the given Handler.
func ApplyUnaryInboundMiddleware(h transport.UnaryHandler, i UnaryInboundMiddleware) transport.UnaryHandler {
	if i == nil {
		return h
	}
	return unaryHandlerWithMiddleware{h: h, i: i}
}

// UnaryInboundMiddlewareFunc adapts a function into an InboundMiddleware.
type UnaryInboundMiddlewareFunc func(context.Context, *transport.Request, transport.ResponseWriter, transport.UnaryHandler) error

// Handle for UnaryInboundMiddlewareFunc
func (f UnaryInboundMiddlewareFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	return f(ctx, req, resw, h)
}

type unaryHandlerWithMiddleware struct {
	h transport.UnaryHandler
	i UnaryInboundMiddleware
}

func (h unaryHandlerWithMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return h.i.Handle(ctx, req, resw, h.h)
}

type nopUnaryInboundMiddleware struct{}

func (nopUnaryInboundMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	return handler.Handle(ctx, req, resw)
}

// OnewayInboundMiddleware defines a transport-level middleware for
// `OnewayHandler`s.
//
// OnewayInboundMiddleware MAY
//
// - change the context
// - change the request
// - handle the returned error
// - call the given handler zero or more times
//
// OnewayInboundMiddleware MUST be thread-safe.
//
// OnewayInboundMiddleware is re-used across requests and MAY be called
// multiple times for the same request.
type OnewayInboundMiddleware interface {
	HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error
}

// NopOnewayInboundMiddleware is an inbound middleware that does not do
// anything special. It simply calls the underlying OnewayHandler.
var NopOnewayInboundMiddleware OnewayInboundMiddleware = nopOnewayInboundMiddleware{}

// ApplyOnewayInboundMiddleware applies the given OnewayInboundMiddleware to
// the given OnewayHandler.
func ApplyOnewayInboundMiddleware(h transport.OnewayHandler, i OnewayInboundMiddleware) transport.OnewayHandler {
	if i == nil {
		return h
	}
	return onewayHandlerWithMiddleware{h: h, i: i}
}

// OnewayInboundMiddlewareFunc adapts a function into a OnwayInboundMiddleware.
type OnewayInboundMiddlewareFunc func(context.Context, *transport.Request, transport.OnewayHandler) error

// HandleOneway for OnewayInboundMiddlewareFunc
func (f OnewayInboundMiddlewareFunc) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	return f(ctx, req, h)
}

type onewayHandlerWithMiddleware struct {
	h transport.OnewayHandler
	i OnewayInboundMiddleware
}

func (h onewayHandlerWithMiddleware) HandleOneway(ctx context.Context, req *transport.Request) error {
	return h.i.HandleOneway(ctx, req, h.h)
}

type nopOnewayInboundMiddleware struct{}

func (nopOnewayInboundMiddleware) HandleOneway(ctx context.Context, req *transport.Request, handler transport.OnewayHandler) error {
	return handler.HandleOneway(ctx, req)
}
