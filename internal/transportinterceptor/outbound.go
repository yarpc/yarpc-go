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

package transportinterceptor

import (
	"context"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

type (
	// UnaryOutbound represents middleware for unary outbound requests.
	UnaryOutbound = middleware.UnaryOutbound

	// OnewayOutbound represents middleware for oneway outbound requests.
	OnewayOutbound = middleware.OnewayOutbound

	// StreamOutbound represents middleware for stream outbound requests.
	StreamOutbound = middleware.StreamOutbound
)

var (
	// NopUnaryOutbound is a no-operation unary outbound middleware.
	NopUnaryOutbound transport.UnaryOutbound = nopUnaryOutbound{}

	// NopOnewayOutbound is a no-operation oneway outbound middleware.
	NopOnewayOutbound transport.OnewayOutbound = nopOnewayOutbound{}

	// NopStreamOutbound is a no-operation stream outbound middleware.
	NopStreamOutbound transport.StreamOutbound = nopStreamOutbound{}
)

type nopUnaryOutbound struct{}

// Call processes a unary request and returns a nil response and no error.
func (nopUnaryOutbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	return nil, nil
}

// Start starts the outbound middleware. It is a no-op.
func (nopUnaryOutbound) Start() error {
	return nil
}

// Stop stops the outbound middleware. It is a no-op.
func (nopUnaryOutbound) Stop() error {
	return nil
}

// IsRunning checks if the outbound middleware is running. Always returns false.
func (nopUnaryOutbound) IsRunning() bool {
	return false
}

// Transports returns the transports associated with this middleware. Always returns nil.
func (nopUnaryOutbound) Transports() []transport.Transport {
	return nil
}

type nopOnewayOutbound struct{}

// CallOneway processes a oneway request and returns a nil ack and no error.
func (nopOnewayOutbound) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	return nil, nil
}

// Start starts the oneway outbound middleware. It is a no-op.
func (nopOnewayOutbound) Start() error {
	return nil
}

// Stop stops the oneway outbound middleware. It is a no-op.
func (nopOnewayOutbound) Stop() error {
	return nil
}

// IsRunning checks if the oneway outbound middleware is running. Always returns false.
func (nopOnewayOutbound) IsRunning() bool {
	return false
}

// Transports returns the transports associated with this middleware. Always returns nil.
func (nopOnewayOutbound) Transports() []transport.Transport {
	return nil
}

type nopStreamOutbound struct{}

// CallStream processes a stream request and returns a nil client stream and no error.
func (nopStreamOutbound) CallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	return nil, nil
}

// Start starts the stream outbound middleware. It is a no-op.
func (nopStreamOutbound) Start() error {
	return nil
}

// Stop stops the stream outbound middleware. It is a no-op.
func (nopStreamOutbound) Stop() error {
	return nil
}

// IsRunning checks if the stream outbound middleware is running. Always returns false.
func (nopStreamOutbound) IsRunning() bool {
	return false
}

// Transports returns the transports associated with this middleware. Always returns nil.
func (nopStreamOutbound) Transports() []transport.Transport {
	return nil
}

// UnaryOutboundFunc adapts a function into a UnaryOutbound middleware.
type UnaryOutboundFunc func(ctx context.Context, req *transport.Request) (*transport.Response, error)

// Call executes the function as a UnaryOutbound call.
func (f UnaryOutboundFunc) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	return f(ctx, req)
}

// Start starts the UnaryOutboundFunc middleware. It is a no-op.
func (f UnaryOutboundFunc) Start() error {
	return nil
}

// Stop stops the UnaryOutboundFunc middleware. It is a no-op.
func (f UnaryOutboundFunc) Stop() error {
	return nil
}

// IsRunning checks if the UnaryOutboundFunc middleware is running. Always returns false.
func (f UnaryOutboundFunc) IsRunning() bool {
	return false
}

// Transports returns the transports associated with this middleware. Always returns nil.
func (f UnaryOutboundFunc) Transports() []transport.Transport {
	return nil
}

// OnewayOutboundFunc adapts a function into a OnewayOutbound middleware.
type OnewayOutboundFunc func(ctx context.Context, req *transport.Request) (transport.Ack, error)

// CallOneway executes the function as a OnewayOutbound call.
func (f OnewayOutboundFunc) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	return f(ctx, req)
}

// Start starts the OnewayOutboundFunc middleware. It is a no-op.
func (f OnewayOutboundFunc) Start() error {
	return nil
}

// Stop stops the OnewayOutboundFunc middleware. It is a no-op.
func (f OnewayOutboundFunc) Stop() error {
	return nil
}

// IsRunning checks if the OnewayOutboundFunc middleware is running. Always returns false.
func (f OnewayOutboundFunc) IsRunning() bool {
	return false
}

// Transports returns the transports associated with this middleware. Always returns nil.
func (f OnewayOutboundFunc) Transports() []transport.Transport {
	return nil
}

// StreamOutboundFunc adapts a function into a StreamOutbound middleware.
type StreamOutboundFunc func(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error)

// CallStream executes the function as a StreamOutbound call.
func (f StreamOutboundFunc) CallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	return f(ctx, req)
}

// Start starts the StreamOutboundFunc middleware. It is a no-op.
func (f StreamOutboundFunc) Start() error {
	return nil
}

// Stop stops the StreamOutboundFunc middleware. It is a no-op.
func (f StreamOutboundFunc) Stop() error {
	return nil
}

// IsRunning checks if the StreamOutboundFunc middleware is running. Always returns false.
func (f StreamOutboundFunc) IsRunning() bool {
	return false
}

// Transports returns the transports associated with this middleware. Always returns nil.
func (f StreamOutboundFunc) Transports() []transport.Transport {
	return nil
}
