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

package inboundmiddleware

import (
	"context"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

// UnaryChain combines a series of `UnaryInbound`s into a single `InboundMiddleware`.
func UnaryChain(mw ...middleware.UnaryInbound) middleware.UnaryInbound {
	switch len(mw) {
	case 0:
		return middleware.NopUnaryInbound
	case 1:
		return mw[0]
	default:
		return unaryChain(mw)
	}
}

type unaryChain []middleware.UnaryInbound

func (c unaryChain) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	return unaryChainExec{
		Chain: []middleware.UnaryInbound(c),
		Final: h,
	}.Handle(ctx, req, resw)
}

// unaryChainExec adapts a series of `UnaryInbound`s into a UnaryHandler.
// It is scoped to a single request to the `Handler` and is not thread-safe.
type unaryChainExec struct {
	Chain []middleware.UnaryInbound
	Final transport.UnaryHandler
}

func (x unaryChainExec) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	if len(x.Chain) == 0 {
		return x.Final.Handle(ctx, req, resw)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.Handle(ctx, req, resw, x)
}

// OnewayChain combines a series of `OnewayInbound`s into a single `InboundMiddleware`.
func OnewayChain(mw ...middleware.OnewayInbound) middleware.OnewayInbound {
	switch len(mw) {
	case 0:
		return middleware.NopOnewayInbound
	case 1:
		return mw[0]
	default:
		return onewayChain(mw)
	}
}

type onewayChain []middleware.OnewayInbound

func (c onewayChain) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	return onewayChainExec{
		Chain: []middleware.OnewayInbound(c),
		Final: h,
	}.HandleOneway(ctx, req)
}

// onewayChainExec adapts a series of `OnewayInbound`s into a OnewayHandler.
// It is scoped to a single request to the `Handler` and is not thread-safe.
type onewayChainExec struct {
	Chain []middleware.OnewayInbound
	Final transport.OnewayHandler
}

func (x onewayChainExec) HandleOneway(ctx context.Context, req *transport.Request) error {
	if len(x.Chain) == 0 {
		return x.Final.HandleOneway(ctx, req)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.HandleOneway(ctx, req, x)
}
