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

package types

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/atomic"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/inboundmiddleware"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// UnaryHandler is a struct that implements the ProcOptions and HandlerOption
// interfaces (so it can be used directly as a procedure, or as a single use
// handler (depending on the use case)).
type UnaryHandler struct {
	Handler    api.UnaryHandler
	Middleware []api.UnaryInboundMiddleware
}

// Start implements Lifecycle.
func (h *UnaryHandler) Start(t testing.TB) error {
	var err error
	err = multierr.Append(err, h.Handler.Start(t))
	for _, mw := range h.Middleware {
		err = multierr.Append(err, mw.Start(t))
	}
	return err
}

// Stop implements Lifecycle.
func (h *UnaryHandler) Stop(t testing.TB) error {
	var err error
	err = multierr.Append(err, h.Handler.Stop(t))
	for _, mw := range h.Middleware {
		err = multierr.Append(err, mw.Stop(t))
	}
	return err
}

// ApplyProc implements ProcOption.
func (h *UnaryHandler) ApplyProc(opts *api.ProcOpts) {
	opts.HandlerSpec = transport.NewUnaryHandlerSpec(h)
}

// ApplyHandler implements HandlerOption.
func (h *UnaryHandler) ApplyHandler(opts *api.HandlerOpts) {
	opts.Handlers = append(opts.Handlers, h)
}

// Handle implements transport.UnaryHandler.
func (h *UnaryHandler) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	mws := make([]middleware.UnaryInbound, 0, len(h.Middleware))
	for _, mw := range h.Middleware {
		mws = append(mws, mw)
	}
	handler := middleware.ApplyUnaryInbound(h.Handler, inboundmiddleware.UnaryChain(mws...))
	return handler.Handle(ctx, req, resw)
}

// OrderedHandler implements the transport.UnaryHandler and ProcOption
// interfaces.
type OrderedHandler struct {
	attempt  atomic.Int32
	Handlers []api.UnaryHandler
}

// Start implements Lifecycle.
func (h *OrderedHandler) Start(t testing.TB) error {
	var err error
	for _, handler := range h.Handlers {
		err = multierr.Append(err, handler.Start(t))
	}
	return err
}

// Stop implements Lifecycle.
func (h *OrderedHandler) Stop(t testing.TB) error {
	var err error
	for _, handler := range h.Handlers {
		err = multierr.Append(err, handler.Stop(t))
	}
	return err
}

// ApplyProc implements ProcOption.
func (h *OrderedHandler) ApplyProc(opts *api.ProcOpts) {
	opts.HandlerSpec = transport.NewUnaryHandlerSpec(h)
}

// Handle implements transport.UnaryHandler#Handle.
func (h *OrderedHandler) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	if len(h.Handlers) <= 0 {
		return fmt.Errorf("no handlers for the request")
	}
	n := h.attempt.Inc()
	if int(n) > len(h.Handlers) {
		return fmt.Errorf("too many requests, expected %d, got %d", len(h.Handlers), n)
	}
	return h.Handlers[n-1].Handle(ctx, req, resw)
}
