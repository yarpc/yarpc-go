// Copyright (c) 2024 Uber Technologies, Inc.
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

package interceptor

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

type (
	// UnaryOutbound defines transport interceptor for `UnaryOutbound`s.
	//
	// UnaryOutbound interceptor MAY do zero or more of the following: change the
	// context, change the request, change the returned response, handle the
	// returned error, call the given outbound zero or more times.
	//
	// UnaryOutbound interceptor MUST always return a non-nil Response or error,
	// and they MUST be thread-safe.
	//
	// UnaryOutbound interceptor is re-used across requests and MAY be called
	// multiple times on the same request.
	UnaryOutbound interface {
		Call(ctx context.Context, request *transport.Request, out transport.UnchainedUnaryOutbound) (*transport.Response, error)
	}

	// OnewayOutbound defines transport interceptor for `OnewayOutbound`s.
	//
	// OnewayOutbound interceptor MAY do zero or more of the following: change the
	// context, change the request, change the returned ack, handle the returned
	// error, call the given outbound zero or more times.
	//
	// OnewayOutbound interceptor MUST always return an Ack (nil or not) or an
	// error, and they MUST be thread-safe.
	//
	// OnewayOutbound interceptor is re-used across requests and MAY be called
	// multiple times on the same request.
	OnewayOutbound interface {
		CallOneway(ctx context.Context, request *transport.Request, out transport.UnchainedOnewayOutbound) (transport.Ack, error)
	}

	// StreamOutbound defines transport interceptor for `StreamOutbound`s.
	//
	// StreamOutbound interceptor MAY do zero or more of the following: change the
	// context, change the requestMeta, change the returned Stream, handle the
	// returned error, call the given outbound zero or more times.
	//
	// StreamOutbound interceptor MUST always return a non-nil Stream or error,
	// and they MUST be thread-safe.
	//
	// StreamOutbound interceptors is re-used across requests and MAY be called
	// multiple times on the same request.
	StreamOutbound interface {
		CallStream(ctx context.Context, req *transport.StreamRequest, out transport.UnchainedStreamOutbound) (*transport.ClientStream, error)
	}
)

type nopUnaryOutbound struct{}

func (nopUnaryOutbound) Call(ctx context.Context, request *transport.Request, out transport.UnchainedUnaryOutbound) (*transport.Response, error) {
	return out.UnchainedCall(ctx, request)
}

// NopUnaryOutbound is a unary outbound middleware that does not do
// anything special. It simply calls the underlying UnaryOutbound.
var NopUnaryOutbound UnaryOutbound = nopUnaryOutbound{}

type nopOnewayOutbound struct{}

func (nopOnewayOutbound) CallOneway(ctx context.Context, request *transport.Request, out transport.UnchainedOnewayOutbound) (transport.Ack, error) {
	return out.UnchainedOnewayCall(ctx, request)
}

// NopOnewayOutbound is an oneway outbound middleware that does not do
// anything special. It simply calls the underlying OnewayOutbound.
var NopOnewayOutbound OnewayOutbound = nopOnewayOutbound{}

type nopStreamOutbound struct{}

func (nopStreamOutbound) CallStream(ctx context.Context, requestMeta *transport.StreamRequest, out transport.UnchainedStreamOutbound) (*transport.ClientStream, error) {
	return out.UnchainedStreamCall(ctx, requestMeta)
}

// NopStreamOutbound is a stream outbound middleware that does not do
// anything special. It simply calls the underlying StreamOutbound.
var NopStreamOutbound StreamOutbound = nopStreamOutbound{}

// ApplyUnaryOutbound applies the given UnaryOutbound interceptor to the given UnchainedUnaryOutbound transport.
func ApplyUnaryOutbound(uo transport.UnchainedUnaryOutbound, i UnaryOutbound) transport.UnchainedUnaryOutbound {
	return unchainedUnaryOutboundWithInterceptor{uo: uo, i: i}
}

// ApplyOnewayOutbound applies the given OnewayOutbound interceptor to the given UnchainedOnewayOutbound transport.
func ApplyOnewayOutbound(oo transport.UnchainedOnewayOutbound, i OnewayOutbound) transport.UnchainedOnewayOutbound {
	return unchainedOnewayOutboundWithInterceptor{oo: oo, i: i}
}

// ApplyStreamOutbound applies the given StreamOutbound interceptor to the given UnchainedStreamOutbound transport.
func ApplyStreamOutbound(so transport.UnchainedStreamOutbound, i StreamOutbound) transport.UnchainedStreamOutbound {
	return unchainedStreamOutboundWithInterceptor{so: so, i: i}
}

type unchainedUnaryOutboundWithInterceptor struct {
	transport.Outbound
	uo transport.UnchainedUnaryOutbound
	i  UnaryOutbound
}

func (uoc unchainedUnaryOutboundWithInterceptor) UnchainedCall(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	return uoc.i.Call(ctx, request, uoc.uo)
}

type unchainedOnewayOutboundWithInterceptor struct {
	transport.Outbound
	oo transport.UnchainedOnewayOutbound
	i  OnewayOutbound
}

func (ooc unchainedOnewayOutboundWithInterceptor) UnchainedOnewayCall(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	return ooc.i.CallOneway(ctx, request, ooc.oo)
}

type unchainedStreamOutboundWithInterceptor struct {
	transport.Outbound
	so transport.UnchainedStreamOutbound
	i  StreamOutbound
}

func (soc unchainedStreamOutboundWithInterceptor) UnchainedStreamCall(ctx context.Context, requestMeta *transport.StreamRequest) (*transport.ClientStream, error) {
	return soc.i.CallStream(ctx, requestMeta, soc.so)
}
