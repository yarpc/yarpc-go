// Copyright (c) 2025 Uber Technologies, Inc.
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

// Package firstoutboundmiddleware annotates every outbound request with
// metadata like the request transport protocol.
// These metadata must be avilable to all subsequent middleware.
package firstoutboundmiddleware

import (
	"context"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

var (
	_ middleware.UnaryOutbound  = (*Middleware)(nil)
	_ middleware.StreamOutbound = (*Middleware)(nil)
	_ middleware.OnewayOutbound = (*Middleware)(nil)
)

// Middleware is the first middleware that MUST be executed in the chain of
// TransportOutboundMiddleware.
type Middleware struct{}

// New returns middleware to begin any YARPC outbound middleware chain.
func New() *Middleware {
	return &Middleware{}
}

// Call implements middleware.UnaryOutbound.
func (m *Middleware) Call(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {
	update(ctx, req, next)
	return next.Call(ctx, req)
}

// CallOneway implements middleware.OnewayOutbound.
func (m *Middleware) CallOneway(ctx context.Context, req *transport.Request, next transport.OnewayOutbound) (transport.Ack, error) {
	update(ctx, req, next)
	return next.CallOneway(ctx, req)
}

// CallStream implements middleware.StreamOutbound.
func (m *Middleware) CallStream(ctx context.Context, req *transport.StreamRequest, next transport.StreamOutbound) (*transport.ClientStream, error) {
	updateStream(ctx, req, next)
	return next.CallStream(ctx, req)
}

func update(ctx context.Context, req *transport.Request, out transport.Outbound) {
	// TODO(apeatsbond): Setting environment headers and unique IDs should live
	// here too (T1860945).

	// Reset the transport field to the current outbound transport.
	// Request forwarding in transport layer proxies needs this when copying
	// requests to a different outbound type.
	if namer, ok := out.(transport.Namer); ok {
		req.Transport = namer.TransportName()
	}
}

func updateStream(ctx context.Context, req *transport.StreamRequest, out transport.Outbound) {
	// TODO(apeatsbond): Setting environment headers and unique IDs should live
	// here too (T1860945).

	// Reset the transport field to the current outbound transport.
	// Request forwarding in transport layer proxies needs this when copying
	// requests to a different outbound type.
	if namer, ok := out.(transport.Namer); ok {
		req.Meta.Transport = namer.TransportName()
	}
}
