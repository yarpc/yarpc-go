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

// UnaryOutboundMiddleware defines transport-level middleware for
// `UnaryOutbound`s.
//
// UnaryOutboundMiddleware MAY do zero or more of the following: change the
// context, change the request, change the returned response, handle the
// returned error, call the given outbound zero or more times.
//
// UnaryOutboundMiddleware MUST always return a non-nil Response or error,
// and they MUST be thread-safe
//
// UnaryOutboundMiddleware is re-used across requests and MAY be called
// multiple times on the same request.
type UnaryOutboundMiddleware interface {
	Call(
		ctx context.Context,
		request *Request,
		buf *Buffer,
		out UnaryOutbound,
	) (*Response, *Buffer, error)
}

// NopUnaryOutboundMiddleware is a unary outbound middleware that does not do
// anything special. It simply calls the underlying UnaryOutboundMiddleware.
var NopUnaryOutboundMiddleware UnaryOutboundMiddleware = nopUnaryOutboundMiddleware{}

// ApplyUnaryOutboundMiddleware applies the given UnaryOutboundMiddleware to
// the given UnaryOutbound transport.
func ApplyUnaryOutboundMiddleware(o UnaryOutbound, f UnaryOutboundMiddleware) UnaryOutbound {
	if f == nil {
		return o
	}
	return unaryOutboundWithMiddleware{o: o, f: f}
}

// UnaryOutboundMiddlewareFunc adapts a function into a UnaryOutboundMiddleware.
type UnaryOutboundMiddlewareFunc func(context.Context, *Request, *Buffer, UnaryOutbound) (*Response, *Buffer, error)

// Call for UnaryOutboundMiddlewareFunc.
func (f UnaryOutboundMiddlewareFunc) Call(
	ctx context.Context,
	request *Request,
	buf *Buffer,
	out UnaryOutbound,
) (*Response, *Buffer, error) {
	return f(ctx, request, buf, out)
}

type unaryOutboundWithMiddleware struct {
	o UnaryOutbound
	f UnaryOutboundMiddleware
}

func (fo unaryOutboundWithMiddleware) Call(
	ctx context.Context,
	request *Request,
	buf *Buffer,
) (*Response, *Buffer, error) {
	return fo.f.Call(ctx, request, buf, fo.o)
}

type nopUnaryOutboundMiddleware struct{}

func (nopUnaryOutboundMiddleware) Call(
	ctx context.Context,
	request *Request,
	buf *Buffer,
	out UnaryOutbound,
) (*Response, *Buffer, error) {
	return out.Call(ctx, request, buf)
}

// StreamOutboundMiddleware defines transport-level middleware for `StreamOutbound`s.
//
// StreamOutboundMiddleware MAY do zero or more of the following: change the
// context, change the request, change the returned Stream, handle the returned
// error, call the given outbound zero or more times.
//
// StreamOutboundMiddleware MUST always return a non-nil Stream or error, and
// they MUST be thread-safe
//
// StreamOutboundMiddleware is re-used across requests and MAY be called
// multiple times on the same request.
type StreamOutboundMiddleware interface {
	CallStream(ctx context.Context, request *Request, out StreamOutbound) (*ClientStream, error)
}

// NopStreamOutboundMiddleware is a stream outbound middleware that does not do
// anything special. It simply calls the underlying StreamOutboundMiddleware.
var NopStreamOutboundMiddleware StreamOutboundMiddleware = nopStreamOutboundMiddleware{}

// ApplyStreamOutboundMiddleware applies the given StreamOutboundMiddleware to
// the given StreamOutboundMiddleware transport.
func ApplyStreamOutboundMiddleware(o StreamOutbound, f StreamOutboundMiddleware) StreamOutbound {
	if f == nil {
		return o
	}
	return streamOutboundWithMiddleware{o: o, f: f}
}

// StreamOutboundMiddlewareFunc adapts a function into a StreamOutboundMiddleware.
type StreamOutboundMiddlewareFunc func(context.Context, *Request, StreamOutbound) (*ClientStream, error)

// CallStream for StreamOutboundMiddlewareFunc.
func (f StreamOutboundMiddlewareFunc) CallStream(ctx context.Context, request *Request, out StreamOutbound) (*ClientStream, error) {
	return f(ctx, request, out)
}

type streamOutboundWithMiddleware struct {
	o StreamOutbound
	f StreamOutboundMiddleware
}

func (fo streamOutboundWithMiddleware) CallStream(ctx context.Context, request *Request) (*ClientStream, error) {
	return fo.f.CallStream(ctx, request, fo.o)
}

type nopStreamOutboundMiddleware struct{}

func (nopStreamOutboundMiddleware) CallStream(ctx context.Context, request *Request, out StreamOutbound) (*ClientStream, error) {
	return out.CallStream(ctx, request)
}
