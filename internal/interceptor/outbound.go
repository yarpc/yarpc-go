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
	// UnaryOutboundChain defines the interface for a chain of unary outbound requests.
	// It provides methods to invoke the next outbound in the chain with the given context
	// and request, and to retrieve the outbound component of the chain.
	//
	// Next: Executes the next outbound request in the chain with the provided context and request,
	//       returning the response and any error encountered during the process.
	// Outbound: Retrieves the outbound component of the chain, allowing for further inspection or manipulation.
	UnaryOutboundChain interface {
		Next(ctx context.Context, request *transport.Request) (*transport.Response, error)
		Outbound() Outbound
	}

	// OnewayOutboundChain defines the interface for a chain of one-way outbound requests.
	// It provides methods to invoke the next outbound in the chain with the given context
	// and request, and to retrieve the outbound component of the chain.
	//
	// Next: Executes the next one-way outbound request in the chain with the provided context and request,
	//       returning an acknowledgment and any error encountered during the process.
	// Outbound: Retrieves the outbound component of the chain, allowing for further inspection or manipulation.
	OnewayOutboundChain interface {
		Next(ctx context.Context, request *transport.Request) (transport.Ack, error)
		Outbound() Outbound
	}

	// StreamOutboundChain defines the interface for a chain of streaming outbound requests.
	// It provides methods to invoke the next outbound in the chain with the given context
	// and request, and to retrieve the outbound component of the chain.
	//
	// Next: Executes the next streaming outbound request in the chain with the provided context and request,
	//       returning a client stream and any error encountered during the process.
	// Outbound: Retrieves the outbound component of the chain, allowing for further inspection or manipulation.
	StreamOutboundChain interface {
		Next(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error)
		Outbound() Outbound
	}

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
		Call(ctx context.Context, request *transport.Request, out UnaryOutboundChain) (*transport.Response, error)
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
		CallOneway(ctx context.Context, request *transport.Request, out OnewayOutboundChain) (transport.Ack, error)
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
		CallStream(ctx context.Context, req *transport.StreamRequest, out StreamOutboundChain) (*transport.ClientStream, error)
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

// DirectUnaryOutbound is a transport that knows how to send unary requests for procedure
// calls.
type DirectUnaryOutbound interface {
	Outbound

	// DirectCall is called without interceptor.
	DirectCall(ctx context.Context, request *transport.Request) (*transport.Response, error)
}

// DirectOnewayOutbound defines a transport outbound for oneway requests
// that does not involve any interceptors.
type DirectOnewayOutbound interface {
	Outbound

	// DirectCallOneway is called without interceptor.
	DirectCallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error)
}

// DirectStreamOutbound defines a transport outbound for streaming requests
// that does not involve any interceptors.
type DirectStreamOutbound interface {
	Outbound

	// DirectCallStream is called without interceptor.
	DirectCallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error)
}

type nopUnaryOutbound struct{}

func (nopUnaryOutbound) Call(ctx context.Context, request *transport.Request, out UnaryOutboundChain) (*transport.Response, error) {
	return out.Next(ctx, request)
}

// NopUnaryOutbound is a unary outbound middleware that does not do
// anything special. It simply calls the underlying UnaryOutbound.
var NopUnaryOutbound UnaryOutbound = nopUnaryOutbound{}

type nopOnewayOutbound struct{}

func (nopOnewayOutbound) CallOneway(ctx context.Context, request *transport.Request, out OnewayOutboundChain) (transport.Ack, error) {
	return out.Next(ctx, request)
}

// NopOnewayOutbound is an oneway outbound middleware that does not do
// anything special. It simply calls the underlying OnewayOutbound.
var NopOnewayOutbound OnewayOutbound = nopOnewayOutbound{}

type nopStreamOutbound struct{}

func (nopStreamOutbound) CallStream(ctx context.Context, requestMeta *transport.StreamRequest, out StreamOutboundChain) (*transport.ClientStream, error) {
	return out.Next(ctx, requestMeta)
}

// NopStreamOutbound is a stream outbound middleware that does not do
// anything special. It simply calls the underlying StreamOutbound.
var NopStreamOutbound StreamOutbound = nopStreamOutbound{}

// ApplyUnaryOutbound applies the given UnaryOutbound interceptor to the given DirectUnaryOutbound transport.
func ApplyUnaryOutbound(uo UnaryOutboundChain, i UnaryOutbound) transport.UnaryOutbound {
	return unaryOutboundWithInterceptor{uo: uo, i: i}
}

// ApplyOnewayOutbound applies the given OnewayOutbound interceptor to the given DirectOnewayOutbound transport.
func ApplyOnewayOutbound(oo OnewayOutboundChain, i OnewayOutbound) transport.OnewayOutbound {
	return onewayOutboundWithInterceptor{oo: oo, i: i}
}

// ApplyStreamOutbound applies the given StreamOutbound interceptor to the given DirectStreamOutbound transport.
func ApplyStreamOutbound(so StreamOutboundChain, i StreamOutbound) transport.StreamOutbound {
	return streamOutboundWithInterceptor{so: so, i: i}
}

type unaryOutboundWithInterceptor struct {
	transport.Outbound
	uo UnaryOutboundChain
	i  UnaryOutbound
}

func (uoc unaryOutboundWithInterceptor) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	return uoc.i.Call(ctx, request, uoc.uo)
}

type onewayOutboundWithInterceptor struct {
	transport.Outbound
	oo OnewayOutboundChain
	i  OnewayOutbound
}

func (ooc onewayOutboundWithInterceptor) CallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	return ooc.i.CallOneway(ctx, request, ooc.oo)
}

type streamOutboundWithInterceptor struct {
	transport.Outbound
	so StreamOutboundChain
	i  StreamOutbound
}

func (soc streamOutboundWithInterceptor) CallStream(ctx context.Context, requestMeta *transport.StreamRequest) (*transport.ClientStream, error) {
	return soc.i.CallStream(ctx, requestMeta, soc.so)
}
