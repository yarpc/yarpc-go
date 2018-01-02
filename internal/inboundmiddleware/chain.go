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

package inboundmiddleware

import (
	"context"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

// UnaryChain combines a series of `UnaryInbound`s into a single `InboundMiddleware`.
func UnaryChain(mw ...middleware.UnaryInbound) middleware.UnaryInbound {
	unchained := make([]middleware.UnaryInbound, 0, len(mw))
	for _, m := range mw {
		if m == nil {
			continue
		}
		if c, ok := m.(unaryChain); ok {
			unchained = append(unchained, c...)
			continue
		}
		unchained = append(unchained, m)
	}

	switch len(unchained) {
	case 0:
		return middleware.NopUnaryInbound
	case 1:
		return unchained[0]
	default:
		return unaryChain(unchained)
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
	unchained := make([]middleware.OnewayInbound, 0, len(mw))
	for _, m := range mw {
		if m == nil {
			continue
		}
		if c, ok := m.(onewayChain); ok {
			unchained = append(unchained, c...)
			continue
		}
		unchained = append(unchained, m)
	}

	switch len(unchained) {
	case 0:
		return middleware.NopOnewayInbound
	case 1:
		return unchained[0]
	default:
		return onewayChain(unchained)
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

// StreamChain combines a series of `StreamInbound`s into a single `InboundMiddleware`.
func StreamChain(mw ...middleware.StreamInbound) middleware.StreamInbound {
	unchained := make([]middleware.StreamInbound, 0, len(mw))
	for _, m := range mw {
		if m == nil {
			continue
		}
		if c, ok := m.(streamChain); ok {
			unchained = append(unchained, c...)
			continue
		}
		unchained = append(unchained, m)
	}

	switch len(unchained) {
	case 0:
		return middleware.NopStreamInbound
	case 1:
		return unchained[0]
	default:
		return streamChain(unchained)
	}
}

type streamChain []middleware.StreamInbound

func (c streamChain) HandleStream(s *transport.ServerStream, h transport.StreamHandler) error {
	return streamChainExec{
		Chain: []middleware.StreamInbound(c),
		Final: h,
	}.HandleStream(s)
}

// streamChainExec adapts a series of `StreamInbound`s into a StreamHandler.
// It is scoped to a single request to the `Handler` and is not thread-safe.
type streamChainExec struct {
	Chain []middleware.StreamInbound
	Final transport.StreamHandler
}

func (x streamChainExec) HandleStream(s *transport.ServerStream) error {
	if len(x.Chain) == 0 {
		return x.Final.HandleStream(s)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.HandleStream(s, x)
}
