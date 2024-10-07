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

// UnaryOutbound represents middleware for unary outbound requests.
type UnaryOutbound = middleware.UnaryOutbound

var (
	// NopUnaryOutbound is a no-operation unary outbound middleware.
	NopUnaryOutbound transport.UnaryOutbound = nopUnaryOutbound{}
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
