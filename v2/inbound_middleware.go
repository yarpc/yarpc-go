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

// UnaryInboundTransportMiddleware defines a transport-level middleware for `UnaryTransportHandler`s.
//
// UnaryInboundTransportMiddleware MAY do zero or more of the following: change the
// context, change the request, modify the response body, handle the returned
// error, call the given handler zero or more times.
//
// UnaryInboundTransportMiddleware MUST be thread-safe.
//
// UnaryInboundTransportMiddleware is re-used across requests and MAY be called multiple
// times for the same request.
type UnaryInboundTransportMiddleware interface {
	Handle(ctx context.Context, req *Request, buf *Buffer, h UnaryTransportHandler) (*Response, *Buffer, error)
}

// NopUnaryInboundTransportMiddleware is an inbound middleware that does not do anything special. It
// simply calls the underlying UnaryEncodingHandler.
var NopUnaryInboundTransportMiddleware UnaryInboundTransportMiddleware = nopUnaryInboundTransportMiddleware{}

// ApplyUnaryInboundTransportMiddleware applies the given InboundMiddleware to the given UnaryEncodingHandler.
func ApplyUnaryInboundTransportMiddleware(h UnaryTransportHandler, i UnaryInboundTransportMiddleware) UnaryTransportHandler {
	if i == nil {
		return h
	}
	return unaryTransportHandlerWithMiddleware{h: h, i: i}
}

// UnaryInboundTransportMiddlewareFunc adapts a function into an InboundMiddleware.
type UnaryInboundTransportMiddlewareFunc func(context.Context, *Request, *Buffer, UnaryTransportHandler) (*Response, *Buffer, error)

// Handle for UnaryInboundTransportMiddlewareFunc
func (f UnaryInboundTransportMiddlewareFunc) Handle(ctx context.Context, req *Request, buf *Buffer, h UnaryTransportHandler) (*Response, *Buffer, error) {
	return f(ctx, req, buf, h)
}

type unaryTransportHandlerWithMiddleware struct {
	h UnaryTransportHandler
	i UnaryInboundTransportMiddleware
}

func (h unaryTransportHandlerWithMiddleware) Handle(ctx context.Context, req *Request, buf *Buffer) (*Response, *Buffer, error) {
	return h.i.Handle(ctx, req, buf, h.h)
}

type nopUnaryInboundTransportMiddleware struct{}

func (nopUnaryInboundTransportMiddleware) Handle(ctx context.Context, req *Request, buf *Buffer, handler UnaryTransportHandler) (*Response, *Buffer, error) {
	return handler.Handle(ctx, req, buf)
}

// StreamInboundTransportMiddleware defines a transport-level middleware for
// `StreamTransportHandler`s.
//
// StreamInboundTransportMiddleware MAY do zero or more of the following: change the
// stream, handle the returned error, call the given handler zero or more times.
//
// StreamInboundTransportMiddleware MUST be thread-safe.
//
// StreamInboundTransportMiddleware is re-used across requests and MAY be called
// multiple times for the same request.
type StreamInboundTransportMiddleware interface {
	HandleStream(s *ServerStream, h StreamTransportHandler) error
}

// NopStreamInboundTransportMiddleware is an inbound middleware that does not do
// anything special. It simply calls the underlying StreamTransportHandler.
var NopStreamInboundTransportMiddleware StreamInboundTransportMiddleware = nopStreamInboundTransportMiddleware{}

// ApplyStreamInboundTransportMiddleware applies the given StreamInboundTransportMiddleware middleware to
// the given StreamTransportHandler.
func ApplyStreamInboundTransportMiddleware(h StreamTransportHandler, i StreamInboundTransportMiddleware) StreamTransportHandler {
	if i == nil {
		return h
	}
	return streamTransportHandlerWithMiddleware{h: h, i: i}
}

// StreamInboundTransportMiddlewareFunc adapts a function into a StreamInboundTransportMiddleware UnaryInboundEncodingMiddleware.
type StreamInboundTransportMiddlewareFunc func(*ServerStream, StreamTransportHandler) error

// HandleStream for StreamInboundTransportMiddlewareFunc
func (f StreamInboundTransportMiddlewareFunc) HandleStream(s *ServerStream, h StreamTransportHandler) error {
	return f(s, h)
}

type streamTransportHandlerWithMiddleware struct {
	h StreamTransportHandler
	i StreamInboundTransportMiddleware
}

func (h streamTransportHandlerWithMiddleware) HandleStream(s *ServerStream) error {
	return h.i.HandleStream(s, h.h)
}

type nopStreamInboundTransportMiddleware struct{}

func (nopStreamInboundTransportMiddleware) HandleStream(s *ServerStream, handler StreamTransportHandler) error {
	return handler.HandleStream(s)
}
