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

// UnaryInbound defines transport-level middleware for `UnaryHandler`s.
type UnaryInbound interface {
	Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error
}

// NopUnaryInbound is an inbound middleware that does not do anything special.
// It simply calls the underlying UnaryHandler.
var NopUnaryInbound UnaryInbound = nopUnaryInbound{}

// ApplyUnaryInbound applies the given UnaryInbound middleware to the given UnaryHandler.
func ApplyUnaryInbound(h transport.UnaryHandler, i UnaryInbound) transport.UnaryHandler {
	if i == nil {
		return h
	}
	return unaryHandlerWithMiddleware{h: h, i: i}
}

// UnaryInboundFunc adapts a function into a UnaryInbound middleware.
type UnaryInboundFunc func(context.Context, *transport.Request, transport.ResponseWriter, transport.UnaryHandler) error

// Handle for UnaryInboundFunc.
func (f UnaryInboundFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	return f(ctx, req, resw, h)
}

type unaryHandlerWithMiddleware struct {
	h transport.UnaryHandler
	i UnaryInbound
}

func (h unaryHandlerWithMiddleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return h.i.Handle(ctx, req, resw, h.h)
}

type nopUnaryInbound struct{}

func (nopUnaryInbound) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, handler transport.UnaryHandler) error {
	return handler.Handle(ctx, req, resw)
}
