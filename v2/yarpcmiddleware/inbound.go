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

package yarpcmiddleware

import (
	"context"

	"go.uber.org/yarpc/v2/yarpctransport"
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
	Handle(ctx context.Context, req *yarpctransport.Request, resw yarpctransport.ResponseWriter, h yarpctransport.UnaryHandler) error
}

// NopUnaryInbound is a inbound middleware that does not do anything special. It
// simply calls the underlying Handler.
var NopUnaryInbound UnaryInbound = nopUnaryInbound{}

// ApplyUnaryInbound applies the given InboundMiddleware to the given Handler.
func ApplyUnaryInbound(h yarpctransport.UnaryHandler, i UnaryInbound) yarpctransport.UnaryHandler {
	if i == nil {
		return h
	}
	return unaryHandlerWithMiddleware{h: h, i: i}
}

// UnaryInboundFunc adapts a function into an InboundMiddleware.
type UnaryInboundFunc func(context.Context, *yarpctransport.Request, yarpctransport.ResponseWriter, yarpctransport.UnaryHandler) error

// Handle for UnaryInboundFunc
func (f UnaryInboundFunc) Handle(ctx context.Context, req *yarpctransport.Request, resw yarpctransport.ResponseWriter, h yarpctransport.UnaryHandler) error {
	return f(ctx, req, resw, h)
}

type unaryHandlerWithMiddleware struct {
	h yarpctransport.UnaryHandler
	i UnaryInbound
}

func (h unaryHandlerWithMiddleware) Handle(ctx context.Context, req *yarpctransport.Request, resw yarpctransport.ResponseWriter) error {
	return h.i.Handle(ctx, req, resw, h.h)
}

type nopUnaryInbound struct{}

func (nopUnaryInbound) Handle(ctx context.Context, req *yarpctransport.Request, resw yarpctransport.ResponseWriter, handler yarpctransport.UnaryHandler) error {
	return handler.Handle(ctx, req, resw)
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
	HandleStream(s *yarpctransport.ServerStream, h yarpctransport.StreamHandler) error
}

// NopStreamInbound is an inbound middleware that does not do
// anything special. It simply calls the underlying StreamHandler.
var NopStreamInbound StreamInbound = nopStreamInbound{}

// ApplyStreamInbound applies the given StreamInbound middleware to
// the given StreamHandler.
func ApplyStreamInbound(h yarpctransport.StreamHandler, i StreamInbound) yarpctransport.StreamHandler {
	if i == nil {
		return h
	}
	return streamHandlerWithMiddleware{h: h, i: i}
}

// StreamInboundFunc adapts a function into a StreamInbound Middleware.
type StreamInboundFunc func(*yarpctransport.ServerStream, yarpctransport.StreamHandler) error

// HandleStream for StreamInboundFunc
func (f StreamInboundFunc) HandleStream(s *yarpctransport.ServerStream, h yarpctransport.StreamHandler) error {
	return f(s, h)
}

type streamHandlerWithMiddleware struct {
	h yarpctransport.StreamHandler
	i StreamInbound
}

func (h streamHandlerWithMiddleware) HandleStream(s *yarpctransport.ServerStream) error {
	return h.i.HandleStream(s, h.h)
}

type nopStreamInbound struct{}

func (nopStreamInbound) HandleStream(s *yarpctransport.ServerStream, handler yarpctransport.StreamHandler) error {
	return handler.HandleStream(s)
}
