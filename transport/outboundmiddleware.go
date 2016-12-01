// Copyright (c) 2016 Uber Technologies, Inc.
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

package transport

import "context"

// UnaryOutboundMiddleware defines transport-level middleware for
// `UnaryOutbound`s.
//
// UnaryOutboundMiddleware MAY
//
// - change the context
// - change the request
// - change the returned response
// - handle the returned error
// - call the given outbound zero or more times
//
// UnaryOutboundMiddleware MUST
//
// - always return a non-nil Response or error.
// - be thread-safe
//
// UnaryOutboundMiddleware is re-used across requests and MAY be called
// multiple times on the same request.
type UnaryOutboundMiddleware interface {
	Call(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error)
}

// NopUnaryOutboundMiddleware is a unary outbound middleware that does not do
// anything special. It simply calls the underlying UnaryOutbound.
var NopUnaryOutboundMiddleware UnaryOutboundMiddleware = nopUnaryOutboundMiddleware{}

// ApplyUnaryOutboundMiddleware applies the given UnaryOutboundMiddleware to
// the given UnaryOutbound.
func ApplyUnaryOutboundMiddleware(o UnaryOutbound, f UnaryOutboundMiddleware) UnaryOutbound {
	if f == nil {
		return o
	}
	return unaryOutboundWithMiddleware{o: o, f: f}
}

// UnaryOutboundMiddlewareFunc adapts a function into a UnaryOutboundMiddleware.
type UnaryOutboundMiddlewareFunc func(context.Context, *Request, UnaryOutbound) (*Response, error)

// Call for UnaryOutboundMiddlewareFunc.
func (f UnaryOutboundMiddlewareFunc) Call(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error) {
	return f(ctx, request, out)
}

type unaryOutboundWithMiddleware struct {
	o UnaryOutbound
	f UnaryOutboundMiddleware
}

func (fo unaryOutboundWithMiddleware) Start() error {
	return fo.o.Start()
}

func (fo unaryOutboundWithMiddleware) Stop() error {
	return fo.o.Stop()
}

func (fo unaryOutboundWithMiddleware) Call(ctx context.Context, request *Request) (*Response, error) {
	return fo.f.Call(ctx, request, fo.o)
}

type nopUnaryOutboundMiddleware struct{}

func (nopUnaryOutboundMiddleware) Call(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error) {
	return out.Call(ctx, request)
}

// OnewayOutboundMiddleware defines transport-level middleware for `OnewayOutbound`s.
//
// OnewayOutboundMiddleware MAY
//
// - change the context
// - change the request
// - change the returned ack
// - handle the returned error
// - call the given outbound zero or more times
//
// OnewayOutboundMiddleware MUST
//
// - always return an Ack (nil or not) or an error.
// - be thread-safe
//
// OnewayOutboundMiddleware is re-used across requests and MAY be called
// multiple times on the same request.
type OnewayOutboundMiddleware interface {
	CallOneway(ctx context.Context, request *Request, out OnewayOutbound) (Ack, error)
}

// NopOnewayOutboundMiddleware is a oneway outbound middleware that does not do
// anything special. It simply calls the underlying OnewayOutbound.
var NopOnewayOutboundMiddleware OnewayOutboundMiddleware = nopOnewayOutboundMiddleware{}

// ApplyOnewayOutboundMiddleware applies the given OnewayOutboundMiddleware to
// the given OnewayOutbound.
func ApplyOnewayOutboundMiddleware(o OnewayOutbound, f OnewayOutboundMiddleware) OnewayOutbound {
	if f == nil {
		return o
	}
	return onewayOutboundWithMiddleware{o: o, f: f}
}

// OnewayOutboundMiddlewareFunc adapts a function into a OnewayOutboundMiddleware.
type OnewayOutboundMiddlewareFunc func(context.Context, *Request, OnewayOutbound) (Ack, error)

// CallOneway for OnewayOutboundMiddlewareFunc.
func (f OnewayOutboundMiddlewareFunc) CallOneway(ctx context.Context, request *Request, out OnewayOutbound) (Ack, error) {
	return f(ctx, request, out)
}

type onewayOutboundWithMiddleware struct {
	o OnewayOutbound
	f OnewayOutboundMiddleware
}

func (fo onewayOutboundWithMiddleware) Start() error {
	return fo.o.Start()
}

func (fo onewayOutboundWithMiddleware) Stop() error {
	return fo.o.Stop()
}

func (fo onewayOutboundWithMiddleware) CallOneway(ctx context.Context, request *Request) (Ack, error) {
	return fo.f.CallOneway(ctx, request, fo.o)
}

type nopOnewayOutboundMiddleware struct{}

func (nopOnewayOutboundMiddleware) CallOneway(ctx context.Context, request *Request, out OnewayOutbound) (Ack, error) {
	return out.CallOneway(ctx, request)
}
