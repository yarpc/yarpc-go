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

const nopName = "nop"

// UnaryOutboundTransportMiddleware defines transport-level middleware for
// `UnaryOutbound`s.
//
// UnaryOutboundTransportMiddleware MAY do zero or more of the following: change the
// context, change the request, change the returned response, handle the
// returned error, call the given outbound zero or more times.
//
// UnaryOutboundTransportMiddleware MUST always return a non-nil Response or error,
// and they MUST be thread-safe
//
// UnaryOutboundTransportMiddleware is re-used across requests and MAY be called
// multiple times on the same request.
type UnaryOutboundTransportMiddleware interface {
	Name() string
	Call(
		ctx context.Context,
		request *Request,
		buf *Buffer,
		out UnaryOutbound,
	) (*Response, *Buffer, error)
}

// NopUnaryOutboundTransportMiddleware is a unary outbound middleware that does not do
// anything special. It simply calls the underlying UnaryOutboundTransportMiddleware.
var NopUnaryOutboundTransportMiddleware UnaryOutboundTransportMiddleware = nopUnaryOutboundTransportMiddleware{}

// ApplyUnaryOutboundTransportMiddleware applies the given UnaryOutboundTransportMiddleware to
// the given UnaryOutbound transport.
func ApplyUnaryOutboundTransportMiddleware(o UnaryOutbound, f ...UnaryOutboundTransportMiddleware) UnaryOutbound {
	if f == nil {
		return o
	}
	outbound := o
	for i := len(f) - 1; i >= 0; i-- {
		outbound = applyUnaryOutboundTransportMiddleware(outbound, f[i])
	}
	return outbound
}

func applyUnaryOutboundTransportMiddleware(o UnaryOutbound, f UnaryOutboundTransportMiddleware) UnaryOutbound {
	if f == nil {
		return o
	}
	return unaryOutboundWithMiddleware{o: o, f: f}
}

// UnaryOutboundTransportMiddlewareFunc adapts a function into a UnaryOutboundTransportMiddleware.
type UnaryOutboundTransportMiddlewareFunc func(context.Context, *Request, *Buffer, UnaryOutbound) (*Response, *Buffer, error)

// Call for UnaryOutboundTransportMiddlewareFunc.
func (f UnaryOutboundTransportMiddlewareFunc) Call(
	ctx context.Context,
	request *Request,
	buf *Buffer,
	out UnaryOutbound,
) (*Response, *Buffer, error) {
	return f(ctx, request, buf, out)
}

type unaryOutboundWithMiddleware struct {
	o UnaryOutbound
	f UnaryOutboundTransportMiddleware
}

func (fo unaryOutboundWithMiddleware) Call(
	ctx context.Context,
	request *Request,
	buf *Buffer,
) (*Response, *Buffer, error) {
	return fo.f.Call(ctx, request, buf, fo.o)
}

type nopUnaryOutboundTransportMiddleware struct{}

func (nopUnaryOutboundTransportMiddleware) Name() string {
	return nopName
}

func (nopUnaryOutboundTransportMiddleware) Call(
	ctx context.Context,
	request *Request,
	buf *Buffer,
	out UnaryOutbound,
) (*Response, *Buffer, error) {
	return out.Call(ctx, request, buf)
}

// StreamOutboundTransportMiddleware defines transport-level middleware for `StreamOutbound`s.
//
// StreamOutboundTransportMiddleware MAY do zero or more of the following: change the
// context, change the request, change the returned Stream, handle the returned
// error, call the given outbound zero or more times.
//
// StreamOutboundTransportMiddleware MUST always return a non-nil Stream or error, and
// they MUST be thread-safe
//
// StreamOutboundTransportMiddleware is re-used across requests and MAY be called
// multiple times on the same request.
type StreamOutboundTransportMiddleware interface {
	CallStream(ctx context.Context, request *Request, out StreamOutbound) (*ClientStream, error)
}

// NopStreamOutboundTransportMiddleware is a stream outbound middleware that does not do
// anything special. It simply calls the underlying StreamOutboundTransportMiddleware.
var NopStreamOutboundTransportMiddleware StreamOutboundTransportMiddleware = nopStreamOutboundTransportMiddleware{}

// ApplyStreamOutboundTransportMiddleware applies the given StreamOutboundTransportMiddleware to
// the given StreamOutboundTransportMiddleware transport.
func ApplyStreamOutboundTransportMiddleware(o StreamOutbound, f StreamOutboundTransportMiddleware) StreamOutbound {
	if f == nil {
		return o
	}
	return streamOutboundWithMiddleware{o: o, f: f}
}

// StreamOutboundTransportMiddlewareFunc adapts a function into a StreamOutboundTransportMiddleware.
type StreamOutboundTransportMiddlewareFunc func(context.Context, *Request, StreamOutbound) (*ClientStream, error)

// CallStream for StreamOutboundTransportMiddlewareFunc.
func (f StreamOutboundTransportMiddlewareFunc) CallStream(ctx context.Context, request *Request, out StreamOutbound) (*ClientStream, error) {
	return f(ctx, request, out)
}

type streamOutboundWithMiddleware struct {
	o StreamOutbound
	f StreamOutboundTransportMiddleware
}

func (fo streamOutboundWithMiddleware) CallStream(ctx context.Context, request *Request) (*ClientStream, error) {
	return fo.f.CallStream(ctx, request, fo.o)
}

type nopStreamOutboundTransportMiddleware struct{}

func (nopStreamOutboundTransportMiddleware) CallStream(ctx context.Context, request *Request, out StreamOutbound) (*ClientStream, error) {
	return out.CallStream(ctx, request)
}
