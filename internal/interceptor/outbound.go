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

	"go.uber.org/yarpc/api/middleware"
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
	UnaryOutbound = middleware.UnaryOutbound

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
	OnewayOutbound = middleware.OnewayOutbound

	// StreamOutbound defines transport interceptor for `StreamOutbound`s.
	//
	// StreamOutbound interceptor MAY do zero or more of the following: change the
	// context, change the requestMeta, change the returned Stream, handle the
	// returned error, call the given outbound zero or more times.
	//
	// StreamOutbound interceptor MUST always return a non-nil Stream or error,
	// and they MUST be thread-safe.
	//
	// StreamOutbound interceptors are re-used across requests and MAY be called
	// multiple times on the same request.
	StreamOutbound = middleware.StreamOutbound
)

var (
	_ transport.UnaryOutbound  = UnaryOutboundFunc(nil)
	_ transport.OnewayOutbound = OnewayOutboundFunc(nil)
	_ transport.StreamOutbound = StreamOutboundFunc(nil)
)

// UnaryOutboundFunc defines a function type that implements the UnaryOutbound interface.
type UnaryOutboundFunc func(context.Context, *transport.Request) (*transport.Response, error)

// Call calls the UnaryOutbound function.
func (f UnaryOutboundFunc) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	return f(ctx, req)
}

// Start starts the UnaryOutbound function. This is a no-op in this implementation.
func (f UnaryOutboundFunc) Start() error {
	return nil
}

// Stop stops the UnaryOutbound function. This is a no-op in this implementation.
func (f UnaryOutboundFunc) Stop() error {
	return nil
}

// IsRunning returns whether the UnaryOutbound function is running. This is a no-op in this implementation.
func (f UnaryOutboundFunc) IsRunning() bool {
	return false
}

// Transports returns the transports used by the UnaryOutbound function. This is a no-op in this implementation.
func (f UnaryOutboundFunc) Transports() []transport.Transport {
	return nil
}

// OnewayOutboundFunc defines a function type that implements the OnewayOutbound interface.
type OnewayOutboundFunc func(context.Context, *transport.Request) (transport.Ack, error)

// CallOneway calls the OnewayOutbound function.
func (f OnewayOutboundFunc) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	return f(ctx, req)
}

// Start starts the OnewayOutbound function. This is a no-op in this implementation.
func (f OnewayOutboundFunc) Start() error {
	return nil
}

// Stop stops the OnewayOutbound function. This is a no-op in this implementation.
func (f OnewayOutboundFunc) Stop() error {
	return nil
}

// IsRunning returns whether the OnewayOutbound function is running. This is a no-op in this implementation.
func (f OnewayOutboundFunc) IsRunning() bool {
	return false
}

// Transports returns the transports used by the OnewayOutbound function. This is a no-op in this implementation.
func (f OnewayOutboundFunc) Transports() []transport.Transport {
	return nil
}

// StreamOutboundFunc defines a function type that implements the StreamOutbound interface.
type StreamOutboundFunc func(context.Context, *transport.StreamRequest) (*transport.ClientStream, error)

// CallStream calls the StreamOutbound function.
func (f StreamOutboundFunc) CallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	return f(ctx, req)
}

// Start starts the StreamOutbound function. This is a no-op in this implementation.
func (f StreamOutboundFunc) Start() error {
	return nil
}

// Stop stops the StreamOutbound function. This is a no-op in this implementation.
func (f StreamOutboundFunc) Stop() error {
	return nil
}

// IsRunning returns whether the StreamOutbound function is running. This is a no-op in this implementation.
func (f StreamOutboundFunc) IsRunning() bool {
	return false
}

// Transports returns the transports used by the StreamOutbound function. This is a no-op in this implementation.
func (f StreamOutboundFunc) Transports() []transport.Transport {
	return nil
}
