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

import "context"

// UnaryInboundEncodingMiddleware defines an encoding-level middleware for handlers.
// These are functionally similar to `UnaryInboundTransportMiddleware`s except encoding middleware may access
// the decoded request/response bodies.
//
// UnaryInboundEncodingMiddleware MAY do zero or more of the following: change the
// context, change the request, modify the response body, handle the returned
// error, call the given handler zero or more times.
//
// UnaryInboundEncodingMiddleware MUST be thread-safe.
//
// UnaryInboundEncodingMiddleware is re-used across requests and MAY be called multiple
// times for the same request.
type UnaryInboundEncodingMiddleware interface {
	Handle(ctx context.Context, reqBuf interface{}, h UnaryEncodingHandler) (interface{}, error)
}

// NopUnaryInboundEncodingMiddleware is an inbound middleware that does not do anything special. It
// simply calls the underlying UnaryEncodingHandler.
var NopUnaryInboundEncodingMiddleware UnaryInboundEncodingMiddleware = nopUnaryInboundEncodingMiddleware{}

type nopUnaryInboundEncodingMiddleware struct{}

func (nopUnaryInboundEncodingMiddleware) Handle(ctx context.Context, reqBuf interface{}, handler UnaryEncodingHandler) (interface{}, error) {
	return handler.Handle(ctx, reqBuf)
}

type unaryEncodingHandlerWithMiddleware struct {
	h UnaryEncodingHandler
	i UnaryInboundEncodingMiddleware
}

func (h unaryEncodingHandlerWithMiddleware) Handle(ctx context.Context, reqBuf interface{}) (interface{}, error) {
	return h.i.Handle(ctx, reqBuf, h.h)
}

// ApplyUnaryInboundEncodingMiddleware applies the given middleware to the given UnaryTransportHandler.
func ApplyUnaryInboundEncodingMiddleware(handler UnaryEncodingHandler, middleware UnaryInboundEncodingMiddleware) UnaryEncodingHandler {
	if middleware == nil {
		return handler
	}
	return unaryEncodingHandlerWithMiddleware{h: handler, i: middleware}
}

// UnaryInboundEncodingMiddlewareFunc adapts a function into an InboundMiddleware.
type UnaryInboundEncodingMiddlewareFunc func(context.Context, *Request, *Buffer, UnaryTransportHandler) (*Response, *Buffer, error)

// Handle for UnaryInboundEncodingMiddlewareFunc
func (f UnaryInboundEncodingMiddlewareFunc) Handle(ctx context.Context, req *Request, reqBuf *Buffer, handler UnaryTransportHandler) (*Response, *Buffer, error) {
	return f(ctx, req, reqBuf, handler)
}
