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

package outboundmiddleware

import (
	"context"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

// UnaryChain combines a series of `UnaryOutbound`s into a single `UnaryOutbound`.
func UnaryChain(mw ...middleware.UnaryOutbound) middleware.UnaryOutbound {
	unchained := make([]middleware.UnaryOutbound, 0, len(mw))
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
		return middleware.NopUnaryOutbound
	case 1:
		return unchained[0]
	default:
		return unaryChain(unchained)
	}
}

type unaryChain []middleware.UnaryOutbound

func (c unaryChain) Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	return unaryChainExec{
		Chain: []middleware.UnaryOutbound(c),
		Final: out,
	}.Call(ctx, request)
}

// unaryChainExec adapts a series of `UnaryOutbound`s into a `UnaryOutbound`. It
// is scoped to a single call of a UnaryOutbound and is not thread-safe.
type unaryChainExec struct {
	Chain []middleware.UnaryOutbound
	Final transport.UnaryOutbound
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

func (x unaryChainExec) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if len(x.Chain) == 0 {
		return x.Final.Call(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.Call(ctx, request, x)
}

// StreamChain combines a series of `StreamOutbound`s into a single `StreamOutbound`.
func StreamChain(mw ...middleware.StreamOutbound) middleware.StreamOutbound {
	unchained := make([]middleware.StreamOutbound, 0, len(mw))
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
		return middleware.NopStreamOutbound
	case 1:
		return unchained[0]
	default:
		return streamChain(unchained)
	}
}

type streamChain []middleware.StreamOutbound

func (c streamChain) CallStream(ctx context.Context, request *transport.StreamRequest, out transport.StreamOutbound) (*transport.ClientStream, error) {
	return streamChainExec{
		Chain: []middleware.StreamOutbound(c),
		Final: out,
	}.CallStream(ctx, request)
}

// streamChainExec adapts a series of `StreamOutbound`s into a `StreamOutbound`. It
// is scoped to a single call of a StreamOutbound and is not thread-safe.
type streamChainExec struct {
	Chain []middleware.StreamOutbound
	Final transport.StreamOutbound
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

func (x streamChainExec) CallStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	if len(x.Chain) == 0 {
		return x.Final.CallStream(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallStream(ctx, request, x)
}
