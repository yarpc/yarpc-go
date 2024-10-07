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
	UnaryOutbound  = middleware.UnaryOutbound
	OnewayOutbound = middleware.OnewayOutbound
	StreamOutbound = middleware.StreamOutbound
)

var (
	NopUnaryOutbound  transport.UnaryOutbound  = nopUnaryOutbound{}
	NopOnewayOutbound transport.OnewayOutbound = nopOnewayOutbound{}
	NopStreamOutbound transport.StreamOutbound = nopStreamOutbound{}
)

type nopUnaryOutbound struct{}

func (nopUnaryOutbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	return nil, nil
}

func (nopUnaryOutbound) Start() error {
	return nil
}

func (nopUnaryOutbound) Stop() error {
	return nil
}

func (nopUnaryOutbound) IsRunning() bool {
	return false
}

func (nopUnaryOutbound) Transports() []transport.Transport {
	return nil
}

type nopOnewayOutbound struct{}

func (nopOnewayOutbound) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	return nil, nil
}

func (nopOnewayOutbound) Start() error {
	return nil
}

func (nopOnewayOutbound) Stop() error {
	return nil
}

func (nopOnewayOutbound) IsRunning() bool {
	return false
}

func (nopOnewayOutbound) Transports() []transport.Transport {
	return nil
}

type nopStreamOutbound struct{}

func (nopStreamOutbound) CallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	return nil, nil
}

func (nopStreamOutbound) Start() error {
	return nil
}

func (nopStreamOutbound) Stop() error {
	return nil
}

func (nopStreamOutbound) IsRunning() bool {
	return false
}

func (nopStreamOutbound) Transports() []transport.Transport {
	return nil
}

// -----------------------
// Middleware wrapper implementations
// -----------------------

// UnaryOutboundFunc adapts a function into an UnaryOutbound middleware.
type UnaryOutboundFunc func(ctx context.Context, req *transport.Request) (*transport.Response, error)

func (f UnaryOutboundFunc) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	return f(ctx, req)
}

func (f UnaryOutboundFunc) Start() error {
	return nil
}

func (f UnaryOutboundFunc) Stop() error {
	return nil
}

func (f UnaryOutboundFunc) IsRunning() bool {
	return false
}

func (f UnaryOutboundFunc) Transports() []transport.Transport {
	return nil
}

// unaryOutboundWithMiddleware wraps UnaryOutbound with middleware.
type unaryOutboundWithMiddleware struct {
	name string
	o    transport.UnaryOutbound
	f    UnaryOutbound
}

func (fo unaryOutboundWithMiddleware) TransportName() string {
	return fo.name
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

// OnewayOutboundFunc adapts a function into a OnewayOutbound middleware.
type OnewayOutboundFunc func(ctx context.Context, req *transport.Request) (transport.Ack, error)

func (f OnewayOutboundFunc) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	return f(ctx, req)
}

func (f OnewayOutboundFunc) Start() error {
	return nil
}

func (f OnewayOutboundFunc) Stop() error {
	return nil
}

func (f OnewayOutboundFunc) IsRunning() bool {
	return false
}

func (f OnewayOutboundFunc) Transports() []transport.Transport {
	return nil
}

// onewayOutboundWithMiddleware wraps OnewayOutbound with middleware.
type onewayOutboundWithMiddleware struct {
	name string
	o    transport.OnewayOutbound
	f    OnewayOutbound
}

func (fo onewayOutboundWithMiddleware) TransportName() string {
	return fo.name
}

func (fo onewayOutboundWithMiddleware) Transports() []transport.Transport {
	return fo.o.Transports()
}

func (fo onewayOutboundWithMiddleware) Start() error {
	return fo.o.Start()
}

func (fo onewayOutboundWithMiddleware) Stop() error {
	return fo.o.Stop()
}

func (fo onewayOutboundWithMiddleware) IsRunning() bool {
	return fo.o.IsRunning()
}

func (fo onewayOutboundWithMiddleware) CallOneway(ctx context.Context, req *transport.Request) (transport.Ack, error) {
	return fo.f.CallOneway(ctx, req, fo.o)
}

// StreamOutboundFunc adapts a function into a StreamOutbound middleware.
type StreamOutboundFunc func(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error)

func (f StreamOutboundFunc) CallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	return f(ctx, req)
}

func (f StreamOutboundFunc) Start() error {
	return nil
}

func (f StreamOutboundFunc) Stop() error {
	return nil
}

func (f StreamOutboundFunc) IsRunning() bool {
	return false
}

func (f StreamOutboundFunc) Transports() []transport.Transport {
	return nil
}

// streamOutboundWithMiddleware wraps StreamOutbound with middleware.
type streamOutboundWithMiddleware struct {
	name string
	o    transport.StreamOutbound
	f    StreamOutbound
}

func (fo streamOutboundWithMiddleware) TransportName() string {
	return fo.name
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

func (fo streamOutboundWithMiddleware) CallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	return fo.f.CallStream(ctx, req, fo.o)
}
