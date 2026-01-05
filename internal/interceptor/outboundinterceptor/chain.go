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

package outboundinterceptor

import (
	"context"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
)

// NewUnaryChain combines a series of `UnaryInbound`s into a single `InboundMiddleware`.
func NewUnaryChain(out interceptor.DirectUnaryOutbound, list []interceptor.UnaryOutbound) interceptor.UnaryOutboundChain {
	return unaryChainExec{
		Chain: list,
		Final: out,
	}
}

func (x unaryChainExec) Next(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if len(x.Chain) == 0 {
		return x.Final.DirectCall(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.Call(ctx, request, x)
}

func (x unaryChainExec) Outbound() transport.Outbound {
	return x.Final
}

// unaryChainExec adapts a series of `UnaryOutbound`s into a `UnaryOutbound`. It
// is scoped to a single call of a UnaryOutbound and is not thread-safe.
type unaryChainExec struct {
	Chain []interceptor.UnaryOutbound
	Final interceptor.DirectUnaryOutbound
}

// NewOnewayChain combines a series of `OnewayInbound`s into a single `InboundMiddleware`.
func NewOnewayChain(out interceptor.DirectOnewayOutbound, list []interceptor.OnewayOutbound) interceptor.OnewayOutboundChain {
	return onewayChainExec{
		Chain: list,
		Final: out,
	}
}

func (x onewayChainExec) Next(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	if len(x.Chain) == 0 {
		return x.Final.DirectCallOneway(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallOneway(ctx, request, x)
}

func (x onewayChainExec) Outbound() transport.Outbound {
	return x.Final
}

// onewayChainExec adapts a series of `OnewayOutbound`s into a `OnewayOutbound`. It
// is scoped to a single call of a OnewayOutbound and is not thread-safe.
type onewayChainExec struct {
	Chain []interceptor.OnewayOutbound
	Final interceptor.DirectOnewayOutbound
}

func (x onewayChainExec) DirectCallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	if len(x.Chain) == 0 {
		return x.Final.DirectCallOneway(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallOneway(ctx, request, x)
}

// NewStreamChain combines a series of `OnewayInbound`s into a single `InboundMiddleware`.
func NewStreamChain(out interceptor.DirectStreamOutbound, list []interceptor.StreamOutbound) interceptor.StreamOutboundChain {
	return streamChainExec{
		Chain: list,
		Final: out,
	}
}

func (x streamChainExec) Next(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	if len(x.Chain) == 0 {
		return x.Final.DirectCallStream(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallStream(ctx, request, x)
}

func (x streamChainExec) Outbound() transport.Outbound {
	return x.Final
}

// streamChainExec adapts a series of `StreamOutbound`s into a `StreamOutbound`. It
// is scoped to a single call of a StreamOutbound and is not thread-safe.
type streamChainExec struct {
	Chain []interceptor.StreamOutbound
	Final interceptor.DirectStreamOutbound
}

func (x streamChainExec) DirectCallStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	if len(x.Chain) == 0 {
		return x.Final.DirectCallStream(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallStream(ctx, request, x)
}
