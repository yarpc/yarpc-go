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

// UnaryFilter defines transport-level middleware for `UnaryOutbound`s.
// Note: this is client side.
//
// UnaryFilter MAY
//
// - change the context
// - change the request
// - change the returned response
// - handle the returned error
// - call the given outbound zero or more times
//
// UnaryFilter MUST
//
// - always return a non-nil Response or error.
// - be thread-safe
//
// UnaryFilter is re-used across requests and MAY be called multiple times on
// the same request.
type UnaryFilter interface {
	Call(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error)
}

// UnaryNopFilter is a filter that does not do anything special. It simply
// calls the underlying UnaryOutbound.
var UnaryNopFilter UnaryFilter = unaryNopFilter{}

// ApplyUnaryFilter applies the given Filter to the given Outbound.
func ApplyUnaryFilter(o UnaryOutbound, f UnaryFilter) UnaryOutbound {
	if f == nil {
		return o
	}
	return unaryFilteredOutbound{o: o, f: f}
}

// UnaryFilterFunc adapts a function into a UnaryFilter.
type UnaryFilterFunc func(context.Context, *Request, UnaryOutbound) (*Response, error)

// Call for UnaryFilterFunc.
func (f UnaryFilterFunc) Call(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error) {
	return f(ctx, request, out)
}

type unaryFilteredOutbound struct {
	o UnaryOutbound
	f UnaryFilter
}

func (fo unaryFilteredOutbound) Start(d Deps) error {
	return fo.o.Start(d)
}

func (fo unaryFilteredOutbound) Stop() error {
	return fo.o.Stop()
}

func (fo unaryFilteredOutbound) Call(ctx context.Context, request *Request) (*Response, error) {
	return fo.f.Call(ctx, request, fo.o)
}

type unaryNopFilter struct{}

func (unaryNopFilter) Call(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error) {
	return out.Call(ctx, request)
}

// OnewayFilter defines transport-level middleware for `OnewayOutbound`s.
// Note: this is client side.
//
// OnewayFilter MAY
//
// - change the context
// - change the request
// - change the returned response
// - handle the returned error
// - call the given outbound zero or more times
//
// OnewayFilter MUST
//
// - always return a non-nil Response or error.
// - be thread-safe
//
// OnewayFilter is re-used across requests and MAY be called multiple times on
// the same request.
type OnewayFilter interface {
	CallOneway(ctx context.Context, request *Request, out OnewayOutbound) (Ack, error)
}

// OnewayNopFilter is a filter that does not do anything special. It simply
// calls the underlying OnewayOutbound.
var OnewayNopFilter OnewayFilter = onewayNopFilter{}

// ApplyOnewayFilter applies the given Filter to the given Outbound.
func ApplyOnewayFilter(o OnewayOutbound, f OnewayFilter) OnewayOutbound {
	if f == nil {
		return o
	}
	return onewayFilteredOutbound{o: o, f: f}
}

// OnewayFilterFunc adapts a function into a OnewayFilter.
type OnewayFilterFunc func(context.Context, *Request, OnewayOutbound) (Ack, error)

// CallOneway for OnewayFilterFunc.
func (f OnewayFilterFunc) CallOneway(ctx context.Context, request *Request, out OnewayOutbound) (Ack, error) {
	return f(ctx, request, out)
}

type onewayFilteredOutbound struct {
	o OnewayOutbound
	f OnewayFilter
}

func (fo onewayFilteredOutbound) Start(d Deps) error {
	return fo.o.Start(d)
}

func (fo onewayFilteredOutbound) Stop() error {
	return fo.o.Stop()
}

func (fo onewayFilteredOutbound) CallOneway(ctx context.Context, request *Request) (Ack, error) {
	return fo.f.CallOneway(ctx, request, fo.o)
}

type onewayNopFilter struct{}

func (onewayNopFilter) CallOneway(ctx context.Context, request *Request, out OnewayOutbound) (Ack, error) {
	return out.CallOneway(ctx, request)
}
