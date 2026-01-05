// Copyright (c) 2026 Uber Technologies, Inc.
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

package api

import (
	"context"
	"testing"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

// UnaryHandler is a wrapper around the transport.UnaryInbound and Lifecycle
// interfaces.
type UnaryHandler interface {
	Lifecycle

	transport.UnaryHandler
}

// UnaryHandlerFunc converts a function into a transport.UnaryHandler.
type UnaryHandlerFunc func(context.Context, *transport.Request, transport.ResponseWriter) error

// Handle implements yarpc/api/transport#UnaryHandler.
func (f UnaryHandlerFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return f(ctx, req, resw)
}

// Start is a noop for wrapped functions.
func (f UnaryHandlerFunc) Start(testing.TB) error { return nil }

// Stop is a noop for wrapped functions.
func (f UnaryHandlerFunc) Stop(testing.TB) error { return nil }

// UnaryInboundMiddleware is a wrapper around the middleware.UnaryInbound and
// Lifecycle interfaces.
type UnaryInboundMiddleware interface {
	Lifecycle

	middleware.UnaryInbound
}

// UnaryInboundMiddlewareFunc converts a function into a transport.UnaryInboundMiddleware.
type UnaryInboundMiddlewareFunc func(context.Context, *transport.Request, transport.ResponseWriter, transport.UnaryHandler) error

// Handle implements yarpc/api/transport#UnaryInboundMiddleware.
func (f UnaryInboundMiddlewareFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	return f(ctx, req, resw, h)
}

// Start is a noop for wrapped functions.
func (f UnaryInboundMiddlewareFunc) Start(testing.TB) error { return nil }

// Stop is a noop for wrapped functions.
func (f UnaryInboundMiddlewareFunc) Stop(testing.TB) error { return nil }

// HandlerOpts are configuration options for a series of handlers.
type HandlerOpts struct {
	Handlers []UnaryHandler
}

// HandlerOption defines options that can be passed into a handler.
type HandlerOption interface {
	Lifecycle

	ApplyHandler(opts *HandlerOpts)
}

// HandlerOptionFunc converts a function into a HandlerOption.
type HandlerOptionFunc func(opts *HandlerOpts)

// ApplyHandler implements HandlerOption.
func (f HandlerOptionFunc) ApplyHandler(opts *HandlerOpts) { f(opts) }
