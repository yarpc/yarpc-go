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
	// DirectUnaryOutbound defines transport interceptor for `UnaryOutbound`s.
	//
	// DirectUnaryOutbound interceptor MAY do zero or more of the following: change the
	// context, change the request, change the returned response, handle the
	// returned error, call the given outbound zero or more times.
	//
	// DirectUnaryOutbound interceptor MUST always return a non-nil Response or error,
	// and they MUST be thread-safe.
	//
	// DirectUnaryOutbound interceptor is re-used across requests and MAY be called
	// multiple times on the same request.
	DirectUnaryOutbound interface {
		Call(ctx context.Context, request *transport.Request, out UnchainedUnaryOutbound) (*transport.Response, error)
	}

	// DirectOnewayOutbound defines transport interceptor for `OnewayOutbound`s.
	//
	// DirectOnewayOutbound interceptor MAY do zero or more of the following: change the
	// context, change the request, change the returned ack, handle the returned
	// error, call the given outbound zero or more times.
	//
	// DirectOnewayOutbound interceptor MUST always return an Ack (nil or not) or an
	// error, and they MUST be thread-safe.
	//
	// DirectOnewayOutbound interceptor is re-used across requests and MAY be called
	// multiple times on the same request.
	DirectOnewayOutbound interface {
		CallOneway(ctx context.Context, request *transport.Request, out UnchainedOnewayOutbound) (transport.Ack, error)
	}

	// DirectStreamOutbound defines transport interceptor for `StreamOutbound`s.
	//
	// DirectStreamOutbound interceptor MAY do zero or more of the following: change the
	// context, change the requestMeta, change the returned Stream, handle the
	// returned error, call the given outbound zero or more times.
	//
	// DirectStreamOutbound interceptor MUST always return a non-nil Stream or error,
	// and they MUST be thread-safe.
	//
	// DirectStreamOutbound interceptors is re-used across requests and MAY be called
	// multiple times on the same request.
	DirectStreamOutbound interface {
		CallStream(ctx context.Context, req *transport.StreamRequest, out UnchainedStreamOutbound) (*transport.ClientStream, error)
	}
)

// Outbound is the common interface for all outbounds.
//
// Outbounds should also implement the Namer interface so that YARPC can
// properly update the Request.Transport field.
type Outbound interface {
	transport.Lifecycle

	// Transports returns the transports that used by this outbound, so they
	// can be collected for lifecycle management, typically by a Dispatcher.
	//
	// Though most outbounds only use a single transport, composite outbounds
	// may use multiple transport protocols, particularly for shadowing traffic
	// across multiple transport protocols during a transport protocol
	// migration.
	Transports() []transport.Transport
}

// UnchainedUnaryOutbound is a transport that knows how to send unary requests for procedure
// calls.
type UnchainedUnaryOutbound interface {
	Outbound

	// UnchainedCall is called without interceptor.
	UnchainedCall(ctx context.Context, request *transport.Request) (*transport.Response, error)
}

// UnchainedOnewayOutbound defines a transport outbound for oneway requests
// that does not involve any interceptors.
type UnchainedOnewayOutbound interface {
	Outbound

	// UnchainedCallOneway is called without interceptor.
	UnchainedCallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error)
}

// UnchainedStreamOutbound defines a transport outbound for streaming requests
// that does not involve any interceptors.
type UnchainedStreamOutbound interface {
	Outbound

	// UnchainedCallStream is called without interceptor.
	UnchainedCallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error)
}

type nopUnaryOutbound struct{}

func (nopUnaryOutbound) Call(ctx context.Context, request *transport.Request, out UnchainedUnaryOutbound) (*transport.Response, error) {
	return out.UnchainedCall(ctx, request)
}

// NopUnaryOutbound is a unary outbound middleware that does not do
// anything special. It simply calls the underlying UnaryOutbound.
var NopUnaryOutbound DirectUnaryOutbound = nopUnaryOutbound{}

type nopOnewayOutbound struct{}

func (nopOnewayOutbound) CallOneway(ctx context.Context, request *transport.Request, out UnchainedOnewayOutbound) (transport.Ack, error) {
	return out.UnchainedCallOneway(ctx, request)
}

// NopOnewayOutbound is an oneway outbound middleware that does not do
// anything special. It simply calls the underlying OnewayOutbound.
var NopOnewayOutbound DirectOnewayOutbound = nopOnewayOutbound{}

type nopStreamOutbound struct{}

func (nopStreamOutbound) CallStream(ctx context.Context, requestMeta *transport.StreamRequest, out UnchainedStreamOutbound) (*transport.ClientStream, error) {
	return out.UnchainedCallStream(ctx, requestMeta)
}

// NopStreamOutbound is a stream outbound middleware that does not do
// anything special. It simply calls the underlying StreamOutbound.
var NopStreamOutbound DirectStreamOutbound = nopStreamOutbound{}

// ApplyUnaryOutbound applies the given UnaryOutbound interceptor to the given UnchainedUnaryOutbound transport.
func ApplyUnaryOutbound(uo UnchainedUnaryOutbound, i DirectUnaryOutbound) UnchainedUnaryOutbound {
	return unchainedUnaryOutboundWithInterceptor{uo: uo, i: i}
}

// ApplyOnewayOutbound applies the given OnewayOutbound interceptor to the given UnchainedOnewayOutbound transport.
func ApplyOnewayOutbound(oo UnchainedOnewayOutbound, i DirectOnewayOutbound) UnchainedOnewayOutbound {
	return unchainedOnewayOutboundWithInterceptor{oo: oo, i: i}
}

// ApplyStreamOutbound applies the given StreamOutbound interceptor to the given UnchainedStreamOutbound transport.
func ApplyStreamOutbound(so UnchainedStreamOutbound, i DirectStreamOutbound) UnchainedStreamOutbound {
	return unchainedStreamOutboundWithInterceptor{so: so, i: i}
}

type unchainedUnaryOutboundWithInterceptor struct {
	transport.Outbound
	uo UnchainedUnaryOutbound
	i  DirectUnaryOutbound
}

func (uoc unchainedUnaryOutboundWithInterceptor) UnchainedCall(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	return uoc.i.Call(ctx, request, uoc.uo)
}

type unchainedOnewayOutboundWithInterceptor struct {
	transport.Outbound
	oo UnchainedOnewayOutbound
	i  DirectOnewayOutbound
}

func (ooc unchainedOnewayOutboundWithInterceptor) UnchainedCallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	return ooc.i.CallOneway(ctx, request, ooc.oo)
}

type unchainedStreamOutboundWithInterceptor struct {
	transport.Outbound
	so UnchainedStreamOutbound
	i  DirectStreamOutbound
}

func (soc unchainedStreamOutboundWithInterceptor) UnchainedCallStream(ctx context.Context, requestMeta *transport.StreamRequest) (*transport.ClientStream, error) {
	return soc.i.CallStream(ctx, requestMeta, soc.so)
}
