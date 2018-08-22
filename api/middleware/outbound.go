// Copyright (c) 2018 Uber Technologies, Inc.
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

package middleware

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// UnaryOutbound defines transport-level middleware for
// `UnaryOutbound`s.
//
// UnaryOutbound middleware MAY do zero or more of the following: change the
// context, change the request, change the returned response, handle the
// returned error, call the given outbound zero or more times.
//
// UnaryOutbound middleware MUST always return a non-nil Response or error,
// and they MUST be thread-safe
//
// UnaryOutbound middleware is re-used across requests and MAY be called
// multiple times on the same request.
type UnaryOutbound interface {
	Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (*transport.Response, error)
}

// NopUnaryOutbound is a unary outbound middleware that does not do
// anything special. It simply calls the underlying UnaryOutbound.
var NopUnaryOutbound UnaryOutbound = nopUnaryOutbound{}

// ApplyUnaryOutbound applies the given UnaryOutbound middleware to
// the given UnaryOutbound transport.
func ApplyUnaryOutbound(o transport.UnaryOutbound, f UnaryOutbound) transport.UnaryOutbound {
	if f == nil {
		return o
	}
	return unaryOutboundWithMiddleware{o: o, f: f}
}

// UnaryOutboundFunc adapts a function into a UnaryOutbound middleware.
type UnaryOutboundFunc func(context.Context, *transport.Request, transport.UnaryOutbound) (*transport.Response, error)

// Call for UnaryOutboundFunc.
func (f UnaryOutboundFunc) Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	return f(ctx, request, out)
}

type unaryOutboundWithMiddleware struct {
	o transport.UnaryOutbound
	f UnaryOutbound
}

func (fo unaryOutboundWithMiddleware) Transports() []transport.Transport {
	return fo.o.Transports()
}

func (fo unaryOutboundWithMiddleware) Start() error {
	return fo.o.Start()
}

func (fo unaryOutboundWithMiddleware) Stop() error {
	return fo.o.Stop()
}

func (fo unaryOutboundWithMiddleware) IsRunning() bool {
	return fo.o.IsRunning()
}

func (fo unaryOutboundWithMiddleware) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	return fo.f.Call(ctx, request, fo.o)
}

type nopUnaryOutbound struct{}

func (nopUnaryOutbound) Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	return out.Call(ctx, request)
}

// StreamOutbound defines transport-level middleware for
// `StreamOutbound`s.
//
// StreamOutbound middleware MAY do zero or more of the following: change the
// context, change the requestMeta, change the returned Stream, handle the
// returned error, call the given outbound zero or more times.
//
// StreamOutbound middleware MUST always return a non-nil Stream or error,
// and they MUST be thread-safe
//
// StreamOutbound middleware is re-used across requests and MAY be called
// multiple times on the same request.
type StreamOutbound interface {
	CallStream(ctx context.Context, request *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error)
}

// NopStreamOutbound is a stream outbound middleware that does not do
// anything special. It simply calls the underlying StreamOutbound.
var NopStreamOutbound StreamOutbound = nopStreamOutbound{}

// ApplyStreamOutbound applies the given StreamOutbound middleware to
// the given StreamOutbound transport.
func ApplyStreamOutbound(o transport.StreamOutbound, f StreamOutbound) transport.StreamOutbound {
	if f == nil {
		return o
	}
	return streamOutboundWithMiddleware{o: o, f: f}
}

// StreamOutboundFunc adapts a function into a StreamOutbound middleware.
type StreamOutboundFunc func(context.Context, *transport.StreamRequest, transport.StreamOutbound) (*transport.ClientStream, error)

// CallStream for StreamOutboundFunc.
func (f StreamOutboundFunc) CallStream(ctx context.Context, request *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	return f(ctx, request, out)
}

type streamOutboundWithMiddleware struct {
	o transport.StreamOutbound
	f StreamOutbound
}

func (fo streamOutboundWithMiddleware) Transports() []transport.Transport {
	return fo.o.Transports()
}

func (fo streamOutboundWithMiddleware) Start() error {
	return fo.o.Start()
}

func (fo streamOutboundWithMiddleware) Stop() error {
	return fo.o.Stop()
}

func (fo streamOutboundWithMiddleware) IsRunning() bool {
	return fo.o.IsRunning()
}

func (fo streamOutboundWithMiddleware) CallStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	return fo.f.CallStream(ctx, request, fo.o)
}

type nopStreamOutbound struct{}

func (nopStreamOutbound) CallStream(ctx context.Context, request *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	return out.CallStream(ctx, request)
}
