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

package outboundinterceptor

import (
	"context"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
)

// UnaryChain combines a series of `UnaryInbound`s into a single `InboundMiddleware`.
func UnaryChain(mw ...interceptor.DirectUnaryOutbound) interceptor.DirectUnaryOutbound {
	unchained := make([]interceptor.DirectUnaryOutbound, 0, len(mw))
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
		return interceptor.NopUnaryOutbound
	case 1:
		return unchained[0]
	default:
		return unaryChain(unchained)
	}
}

type unaryChain []interceptor.DirectUnaryOutbound

func (x unaryChainExec) TransportName() string {
	var name string
	if namer, ok := x.Final.(transport.Namer); ok {
		name = namer.TransportName()
	}
	return name
}

func (x unaryChainExec) Transports() []transport.Transport {
	return x.Final.Transports()
}

func (x unaryChainExec) Start() error {
	return x.Final.Start()
}

func (x unaryChainExec) Stop() error {
	return x.Final.Stop()
}

func (x unaryChainExec) IsRunning() bool {
	return x.Final.IsRunning()
}

func (c unaryChain) Call(ctx context.Context, request *transport.Request, out interceptor.UnchainedUnaryOutbound) (*transport.Response, error) {
	return unaryChainExec{
		Chain: c,
		Final: out,
	}.UnchainedCall(ctx, request)
}

func (x unaryChainExec) UnchainedCall(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if len(x.Chain) == 0 {
		return x.Final.UnchainedCall(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.Call(ctx, request, x)
}

// unaryChainExec adapts a series of `UnaryOutbound`s into a `UnaryOutbound`. It
// is scoped to a single call of a UnaryOutbound and is not thread-safe.
type unaryChainExec struct {
	Chain []interceptor.DirectUnaryOutbound
	Final interceptor.UnchainedUnaryOutbound
}

// OnewayChain combines a series of `OnewayOutbound`s into a single `OnewayOutbound`.
func OnewayChain(mw ...interceptor.DirectOnewayOutbound) interceptor.DirectOnewayOutbound {
	unchained := make([]interceptor.DirectOnewayOutbound, 0, len(mw))
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
		return interceptor.NopOnewayOutbound
	case 1:
		return unchained[0]
	default:
		return onewayChain(unchained)
	}
}

type onewayChain []interceptor.DirectOnewayOutbound

func (c onewayChain) CallOneway(ctx context.Context, request *transport.Request, out interceptor.UnchainedOnewayOutbound) (transport.Ack, error) {
	return onewayChainExec{
		Chain: c,
		Final: out,
	}.UnchainedCallOneway(ctx, request)
}

// onewayChainExec adapts a series of `OnewayOutbound`s into a `OnewayOutbound`. It
// is scoped to a single call of a OnewayOutbound and is not thread-safe.
type onewayChainExec struct {
	Chain []interceptor.DirectOnewayOutbound
	Final interceptor.UnchainedOnewayOutbound
}

func (x onewayChainExec) TransportName() string {
	var name string
	if namer, ok := x.Final.(transport.Namer); ok {
		name = namer.TransportName()
	}
	return name
}

func (x onewayChainExec) Transports() []transport.Transport {
	return x.Final.Transports()
}

func (x onewayChainExec) Start() error {
	return x.Final.Start()
}

func (x onewayChainExec) Stop() error {
	return x.Final.Stop()
}

func (x onewayChainExec) IsRunning() bool {
	return x.Final.IsRunning()
}

func (x onewayChainExec) UnchainedCallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	if len(x.Chain) == 0 {
		return x.Final.UnchainedCallOneway(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallOneway(ctx, request, x)
}

// StreamChain combines a series of `StreamOutbound`s into a single `StreamOutbound`.
func StreamChain(mw ...interceptor.DirectStreamOutbound) interceptor.DirectStreamOutbound {
	unchained := make([]interceptor.DirectStreamOutbound, 0, len(mw))
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
		return interceptor.NopStreamOutbound
	case 1:
		return unchained[0]
	default:
		return streamChain(unchained)
	}
}

type streamChain []interceptor.DirectStreamOutbound

func (c streamChain) CallStream(ctx context.Context, request *transport.StreamRequest, out interceptor.UnchainedStreamOutbound) (*transport.ClientStream, error) {
	return streamChainExec{
		Chain: c,
		Final: out,
	}.UnchainedCallStream(ctx, request)
}

// streamChainExec adapts a series of `StreamOutbound`s into a `StreamOutbound`. It
// is scoped to a single call of a StreamOutbound and is not thread-safe.
type streamChainExec struct {
	Chain []interceptor.DirectStreamOutbound
	Final interceptor.UnchainedStreamOutbound
}

func (x streamChainExec) TransportName() string {
	var name string
	if namer, ok := x.Final.(transport.Namer); ok {
		name = namer.TransportName()
	}
	return name
}

func (x streamChainExec) Transports() []transport.Transport {
	return x.Final.Transports()
}

func (x streamChainExec) Start() error {
	return x.Final.Start()
}

func (x streamChainExec) Stop() error {
	return x.Final.Stop()
}

func (x streamChainExec) IsRunning() bool {
	return x.Final.IsRunning()
}

func (x streamChainExec) UnchainedCallStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	if len(x.Chain) == 0 {
		return x.Final.UnchainedCallStream(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallStream(ctx, request, x)
}
