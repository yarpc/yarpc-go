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
	Name() string
	Handle(ctx context.Context, reqBuf interface{}, h UnaryEncodingHandler) (interface{}, error)
}

// NopUnaryInboundEncodingMiddleware is an inbound middleware that does not do anything special. It
// simply calls the underlying UnaryEncodingHandler.
var NopUnaryInboundEncodingMiddleware UnaryInboundEncodingMiddleware = nopUnaryInboundEncodingMiddleware{}

type nopUnaryInboundEncodingMiddleware struct{}

func (nopUnaryInboundEncodingMiddleware) Name() string { return nopName }

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

// ApplyUnaryInboundEncodingMiddleware applies the middleware to the
// UnaryInboundEncodingHandler.
func ApplyUnaryInboundEncodingMiddleware(h UnaryEncodingHandler, middleware ...UnaryInboundEncodingMiddleware) UnaryEncodingHandler {
	handler := h
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = applyUnaryInboundEncodingMiddleware(handler, middleware[i])
	}
	return handler
}

func applyUnaryInboundEncodingMiddleware(handler UnaryEncodingHandler, middleware UnaryInboundEncodingMiddleware) UnaryEncodingHandler {
	if middleware == nil {
		return handler
	}
	return unaryEncodingHandlerWithMiddleware{h: handler, i: middleware}
}

// NewUnaryInboundEncodingMiddleware is a convenience constructor for creating
// new middleware.
func NewUnaryInboundEncodingMiddleware(
	name string,
	f func(context.Context, interface{}, UnaryEncodingHandler) (interface{}, error),
) UnaryInboundEncodingMiddleware {
	return unaryInboundEncodingMiddleware{
		name: name,
		f:    f,
	}
}

// unaryInboundEncodingMiddleware adapts a function and name into a
// UnaryInboundEncodingMiddleware.
type unaryInboundEncodingMiddleware struct {
	name string
	f    func(context.Context, interface{}, UnaryEncodingHandler) (interface{}, error)
}

// Name for unaryInboundEncodingMiddleware
func (u unaryInboundEncodingMiddleware) Name() string {
	return u.name
}

// Handle for unaryInboundEncodingMiddleware
func (u unaryInboundEncodingMiddleware) Handle(ctx context.Context, reqBuf interface{}, handler UnaryEncodingHandler) (interface{}, error) {
	return u.f(ctx, reqBuf, handler)
}
