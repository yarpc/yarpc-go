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

package yarpc

import (
	"context"
)

// UnaryInboundMiddleware defines a transport-level middleware for `UnaryHandler`s.
//
// UnaryInboundMiddleware MAY do zero or more of the following: change the
// context, change the request, modify the response body, handle the returned
// error, call the given handler zero or more times.
//
// UnaryInboundMiddleware MUST be thread-safe.
//
// UnaryInboundMiddleware is re-used across requests and MAY be called multiple
// times for the same request.
type UnaryInboundMiddleware interface {
	Handle(ctx context.Context, req *Request, buf *Buffer, h UnaryHandler) (*Response, *Buffer, error)
}

// NopUnaryInboundMiddleware is an inbound middleware that does not do anything special. It
// simply calls the underlying Handler.
var NopUnaryInboundMiddleware UnaryInboundMiddleware = nopUnaryInboundMiddleware{}

// ApplyUnaryInboundMiddleware applies the given InboundMiddleware to the given Handler.
func ApplyUnaryInboundMiddleware(h UnaryHandler, i UnaryInboundMiddleware) UnaryHandler {
	if i == nil {
		return h
	}
	return unaryHandlerWithMiddleware{h: h, i: i}
}

// UnaryInboundMiddlewareFunc adapts a function into an InboundMiddleware.
type UnaryInboundMiddlewareFunc func(context.Context, *Request, *Buffer, UnaryHandler) (*Response, *Buffer, error)

// Handle for UnaryInboundMiddlewareFunc
func (f UnaryInboundMiddlewareFunc) Handle(ctx context.Context, req *Request, buf *Buffer, h UnaryHandler) (*Response, *Buffer, error) {
	return f(ctx, req, buf, h)
}

type unaryHandlerWithMiddleware struct {
	h UnaryHandler
	i UnaryInboundMiddleware
}

func (h unaryHandlerWithMiddleware) Handle(ctx context.Context, req *Request, buf *Buffer) (*Response, *Buffer, error) {
	return h.i.Handle(ctx, req, buf, h.h)
}

type nopUnaryInboundMiddleware struct{}

func (nopUnaryInboundMiddleware) Handle(ctx context.Context, req *Request, buf *Buffer, handler UnaryHandler) (*Response, *Buffer, error) {
	return handler.Handle(ctx, req, buf)
}

// StreamInboundMiddleware defines a transport-level middleware for
// `StreamHandler`s.
//
// StreamInboundMiddleware MAY do zero or more of the following: change the
// stream, handle the returned error, call the given handler zero or more times.
//
// StreamInboundMiddleware MUST be thread-safe.
//
// StreamInboundMiddleware is re-used across requests and MAY be called
// multiple times for the same request.
type StreamInboundMiddleware interface {
	HandleStream(s *ServerStream, h StreamHandler) error
}

// NopStreamInboundMiddleware is an inbound middleware that does not do
// anything special. It simply calls the underlying StreamHandler.
var NopStreamInboundMiddleware StreamInboundMiddleware = nopStreamInboundMiddleware{}

// ApplyStreamInboundMiddleware applies the given StreamInboundMiddleware middleware to
// the given StreamHandler.
func ApplyStreamInboundMiddleware(h StreamHandler, i StreamInboundMiddleware) StreamHandler {
	if i == nil {
		return h
	}
	return streamHandlerWithMiddleware{h: h, i: i}
}

// StreamInboundMiddlewareFunc adapts a function into a StreamInboundMiddleware Middleware.
type StreamInboundMiddlewareFunc func(*ServerStream, StreamHandler) error

// HandleStream for StreamInboundMiddlewareFunc
func (f StreamInboundMiddlewareFunc) HandleStream(s *ServerStream, h StreamHandler) error {
	return f(s, h)
}

type streamHandlerWithMiddleware struct {
	h StreamHandler
	i StreamInboundMiddleware
}

func (h streamHandlerWithMiddleware) HandleStream(s *ServerStream) error {
	return h.i.HandleStream(s, h.h)
}

type nopStreamInboundMiddleware struct{}

func (nopStreamInboundMiddleware) HandleStream(s *ServerStream, handler StreamHandler) error {
	return handler.HandleStream(s)
}
