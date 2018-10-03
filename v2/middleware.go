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
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE	 OR OTHER DEALINGS IN
// THE SOFTWARE.

package yarpc

import (
	"context"
)

// UnaryInboundEncodingMiddleware defines an encoding-level middleware for `UnaryTransportHandler`s.
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
	Handle(ctx context.Context, req *Request, reqBody interface{}, h UnaryEncodingHandler) (*Response, interface{}, error)
}

type UnaryEncodingHandler interface {
	Handle(ctx context.Context, req *Request, reqBody interface{}) (*Response, interface{}, error)
}

// NopUnaryInboundEncodingMiddleware is a inbound middleware that does not do anything special. It
// simply calls the underlying UnaryEncodingHandler.
var NopUnaryInboundEncodingMiddleware UnaryInboundEncodingMiddleware = nopUnaryInboundEncodingMiddleware{}

// ApplyUnaryInboundEncodingMiddleware applies the given UnaryInboundEncodingMiddleware to the given UnaryEncodingHandler.
func ApplyUnaryInboundEncodingMiddleware(h UnaryEncodingHandler, m UnaryInboundEncodingMiddleware) UnaryEncodingHandler {
	if m == nil {
		return h
	}
	return unaryEncodingHandlerWithMiddleware{h: h, m: m}
}

type unaryEncodingHandlerWithMiddleware struct {
	h UnaryEncodingHandler
	m UnaryInboundEncodingMiddleware
}

func (h unaryEncodingHandlerWithMiddleware) Handle(ctx context.Context, req *Request, reqBody interface{}) (*Response, interface{}, error) {
	return h.m.Handle(ctx, req, reqBody, h.h)
}

type nopUnaryInboundEncodingMiddleware struct{}

func (nopUnaryInboundEncodingMiddleware) Handle(ctx context.Context, req *Request, reqBody interface{}, h UnaryEncodingHandler) (*Response, interface{}, error) {
	return h.Handle(ctx, req, reqBody)
}
