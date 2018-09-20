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

	yarpc "go.uber.org/yarpc/v2"
)

// UnaryInbound defines a transport-level middleware for `UnaryHandler`s.
//
// UnaryInbound middleware MAY do zero or more of the following: change the
// context, change the request, modify the response body, handle the returned
// error, call the given handler zero or more times.
//
// UnaryInbound middleware MUST be thread-safe.
//
// UnaryInbound middleware is re-used across requests and MAY be called multiple
// times for the same request.
type UnaryInbound interface {
	Handle(ctx context.Context, req *yarpc.Request, buf *yarpc.Buffer, h yarpc.UnaryHandler) (*yarpc.Response, *yarpc.Buffer, error)
}

// NopUnaryInbound is a inbound middleware that does not do anything special. It
// simply calls the underlying Handler.
var NopUnaryInbound UnaryInbound = nopUnaryInbound{}

// ApplyUnaryInbound applies the given InboundMiddleware to the given Handler.
func ApplyUnaryInbound(h yarpc.UnaryHandler, i UnaryInbound) yarpc.UnaryHandler {
	if i == nil {
		return h
	}
	return unaryHandlerWithMiddleware{h: h, i: i}
}

// UnaryInboundFunc adapts a function into an InboundMiddleware.
type UnaryInboundFunc func(context.Context, *yarpc.Request, *yarpc.Buffer, yarpc.UnaryHandler) (*yarpc.Response, *yarpc.Buffer, error)

// Handle for UnaryInboundFunc
func (f UnaryInboundFunc) Handle(ctx context.Context, req *yarpc.Request, buf *yarpc.Buffer, h yarpc.UnaryHandler) (*yarpc.Response, *yarpc.Buffer, error) {
	return f(ctx, req, buf, h)
}

type unaryHandlerWithMiddleware struct {
	h yarpc.UnaryHandler
	i UnaryInbound
}

func (h unaryHandlerWithMiddleware) Handle(ctx context.Context, req *yarpc.Request, buf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	return h.i.Handle(ctx, req, buf, h.h)
}

type nopUnaryInbound struct{}

func (nopUnaryInbound) Handle(ctx context.Context, req *yarpc.Request, buf *yarpc.Buffer, handler yarpc.UnaryHandler) (*yarpc.Response, *yarpc.Buffer, error) {
	return handler.Handle(ctx, req, buf)
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
	HandleStream(s *yarpc.ServerStream, h yarpc.StreamHandler) error
}

// NopStreamInbound is an inbound middleware that does not do
// anything special. It simply calls the underlying StreamHandler.
var NopStreamInbound StreamInbound = nopStreamInbound{}

// ApplyStreamInbound applies the given StreamInbound middleware to
// the given StreamHandler.
func ApplyStreamInbound(h yarpc.StreamHandler, i StreamInbound) yarpc.StreamHandler {
	if i == nil {
		return h
	}
	return streamHandlerWithMiddleware{h: h, i: i}
}

// StreamInboundFunc adapts a function into a StreamInbound Middleware.
type StreamInboundFunc func(*yarpc.ServerStream, yarpc.StreamHandler) error

// HandleStream for StreamInboundFunc
func (f StreamInboundFunc) HandleStream(s *yarpc.ServerStream, h yarpc.StreamHandler) error {
	return f(s, h)
}

type streamHandlerWithMiddleware struct {
	h yarpc.StreamHandler
	i StreamInbound
}

func (h streamHandlerWithMiddleware) HandleStream(s *yarpc.ServerStream) error {
	return h.i.HandleStream(s, h.h)
}

type nopStreamInbound struct{}

func (nopStreamInbound) HandleStream(s *yarpc.ServerStream, handler yarpc.StreamHandler) error {
	return handler.HandleStream(s)
}
