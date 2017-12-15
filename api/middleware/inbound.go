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

package middleware

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// UnaryInbound defines a transport-level middleware for
// `UnaryHandler`s.
//
// UnaryInbound middleware MAY do zero or more of the following: change the
// context, change the request, call the ResponseWriter, modify the response
// body by wrapping the ResponseWriter, handle the returned error, call the
// given handler zero or more times.
//
// UnaryInbound middleware MUST be thread-safe.
//
// UnaryInbound middleware is re-used across requests and MAY be called multiple times
// for the same request.
type UnaryInbound interface {
	Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error
}

// NopUnaryInbound is a inbound middleware that does not do anything special. It
// simply calls the underlying Handler.
var NopUnaryInbound UnaryInbound = nopUnaryInbound{}

// ApplyUnaryInbound applies the given InboundMiddleware to the given Handler.
func ApplyUnaryInbound(h transport.UnaryHandler, i UnaryInbound) transport.UnaryHandler {
	if i == nil {
		return h
	}
	return unaryHandlerWithMiddleware{h: h, i: i}
}

// UnaryInboundFunc adapts a function into an InboundMiddleware.
type UnaryInboundFunc func(context.Context, *transport.Request, transport.ResponseWriter, transport.UnaryHandler) error

// Handle for UnaryInboundFunc
func (f UnaryInboundFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	return f(ctx, req, resw, h)
}

type unaryHandlerWithMiddleware struct {
	h transport.UnaryHandler
	i UnaryInbound
}

func (h unaryHandlerWithMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return h.i.Handle(ctx, req, resw, h.h)
}

type nopUnaryInbound struct{}

func (nopUnaryInbound) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	return handler.Handle(ctx, req, resw)
}

// OnewayInbound defines a transport-level middleware for
// `OnewayHandler`s.
//
// OnewayInbound middleware MAY do zero or more of the following: change the
// context, change the request, handle the returned error, call the given
// handler zero or more times.
//
// OnewayInbound middleware MUST be thread-safe.
//
// OnewayInbound middleware is re-used across requests and MAY be called
// multiple times for the same request.
type OnewayInbound interface {
	HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error
}

// NopOnewayInbound is an inbound middleware that does not do
// anything special. It simply calls the underlying OnewayHandler.
var NopOnewayInbound OnewayInbound = nopOnewayInbound{}

// ApplyOnewayInbound applies the given OnewayInbound middleware to
// the given OnewayHandler.
func ApplyOnewayInbound(h transport.OnewayHandler, i OnewayInbound) transport.OnewayHandler {
	if i == nil {
		return h
	}
	return onewayHandlerWithMiddleware{h: h, i: i}
}

// OnewayInboundFunc adapts a function into a OnewayInbound Middleware.
type OnewayInboundFunc func(context.Context, *transport.Request, transport.OnewayHandler) error

// HandleOneway for OnewayInboundFunc
func (f OnewayInboundFunc) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	return f(ctx, req, h)
}

type onewayHandlerWithMiddleware struct {
	h transport.OnewayHandler
	i OnewayInbound
}

func (h onewayHandlerWithMiddleware) HandleOneway(ctx context.Context, req *transport.Request) error {
	return h.i.HandleOneway(ctx, req, h.h)
}

type nopOnewayInbound struct{}

func (nopOnewayInbound) HandleOneway(ctx context.Context, req *transport.Request, handler transport.OnewayHandler) error {
	return handler.HandleOneway(ctx, req)
}

// StreamInbound defines a transport-level middleware for
// `StreamHandler`s.
//
// StreamInbound middleware MAY do zero or more of the following: change the
// stream, handle the returned error, call the given handler zero or more times.
//
// StreamInbound middleware MUST be thread-safe.
//
// StreamInbound middleware is re-used across requests and MAY be called
// multiple times for the same request.
type StreamInbound interface {
	HandleStream(s *transport.ServerStream, h transport.StreamHandler) error
}

// NopStreamInbound is an inbound middleware that does not do
// anything special. It simply calls the underlying StreamHandler.
var NopStreamInbound StreamInbound = nopStreamInbound{}

// ApplyStreamInbound applies the given StreamInbound middleware to
// the given StreamHandler.
func ApplyStreamInbound(h transport.StreamHandler, i StreamInbound) transport.StreamHandler {
	if i == nil {
		return h
	}
	return streamHandlerWithMiddleware{h: h, i: i}
}

// StreamInboundFunc adapts a function into a StreamInbound Middleware.
type StreamInboundFunc func(*transport.ServerStream, transport.StreamHandler) error

// HandleStream for StreamInboundFunc
func (f StreamInboundFunc) HandleStream(s *transport.ServerStream, h transport.StreamHandler) error {
	return f(s, h)
}

type streamHandlerWithMiddleware struct {
	h transport.StreamHandler
	i StreamInbound
}

func (h streamHandlerWithMiddleware) HandleStream(s *transport.ServerStream) error {
	return h.i.HandleStream(s, h.h)
}

type nopStreamInbound struct{}

func (nopStreamInbound) HandleStream(s *transport.ServerStream, handler transport.StreamHandler) error {
	return handler.HandleStream(s)
}
