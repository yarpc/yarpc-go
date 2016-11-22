// Copyright (c) 2016 Uber Technologies, Inc.
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

package filter

import (
	"context"

	"go.uber.org/yarpc/transport"
)

// UnaryChain combines a series of `UnaryFilter`s into a single `UnaryFilter`.
func UnaryChain(filters ...transport.UnaryOutboundMiddleware) transport.UnaryOutboundMiddleware {
	switch len(filters) {
	case 0:
		return transport.NopUnaryOutboundMiddleware
	case 1:
		return filters[0]
	default:
		return unaryChain(filters)
	}
}

type unaryChain []transport.UnaryOutboundMiddleware

func (c unaryChain) Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	return unaryChainExec{
		Chain: []transport.UnaryOutboundMiddleware(c),
		Final: out,
	}.Call(ctx, request)
}

// unaryChainExec adapts a series of `UnaryFilter`s into a `UnaryOutbound`. It
// is scoped to a single call of a UnaryOutbound and is not thread-safe.
type unaryChainExec struct {
	Chain []transport.UnaryOutboundMiddleware
	Final transport.UnaryOutbound
}

func (x unaryChainExec) Start(d transport.Deps) error {
	return x.Final.Start(d)
}

func (x unaryChainExec) Stop() error {
	return x.Final.Stop()
}

func (x unaryChainExec) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if len(x.Chain) == 0 {
		return x.Final.Call(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.Call(ctx, request, x)
}

// OnewayChain combines a series of `OnewayFilter`s into a single `OnewayFilter`.
func OnewayChain(filters ...transport.OnewayOutboundMiddleware) transport.OnewayOutboundMiddleware {
	switch len(filters) {
	case 0:
		return transport.NopOnewayOutboundMiddleware
	case 1:
		return filters[0]
	default:
		return onewayChain(filters)
	}
}

type onewayChain []transport.OnewayOutboundMiddleware

func (c onewayChain) CallOneway(ctx context.Context, request *transport.Request, out transport.OnewayOutbound) (transport.Ack, error) {
	return onewayChainExec{
		Chain: []transport.OnewayOutboundMiddleware(c),
		Final: out,
	}.CallOneway(ctx, request)
}

// onewayChainExec adapts a series of `OnewayFilter`s into a `OnewayOutbound`. It
// is scoped to a single call of a OnewayOutbound and is not thread-safe.
type onewayChainExec struct {
	Chain []transport.OnewayOutboundMiddleware
	Final transport.OnewayOutbound
}

func (x onewayChainExec) Start(d transport.Deps) error {
	return x.Final.Start(d)
}

func (x onewayChainExec) Stop() error {
	return x.Final.Stop()
}

func (x onewayChainExec) CallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	if len(x.Chain) == 0 {
		return x.Final.CallOneway(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallOneway(ctx, request, x)
}
