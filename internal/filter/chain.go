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

// Chain combines a series of filters into a single Filter.
func Chain(filters ...transport.UnaryFilter) transport.UnaryFilter {
	switch len(filters) {
	case 0:
		return transport.NopUnaryFilter
	case 1:
		return filters[0]
	default:
		return chain(filters)
	}
}

// filterChain combines a series of filters into a single Filter.
type chain []transport.UnaryFilter

func (c chain) CallUnary(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	return chainExec{
		Chain: []transport.UnaryFilter(c),
		Final: out,
	}.Call(ctx, request)
}

// chainExec adapts a series of filters into an Outbound. It is scoped to a
// single call of an Outbound and is not thread-safe.
type chainExec struct {
	Chain []transport.UnaryFilter
	Final transport.UnaryOutbound
}

func (x chainExec) Start(d transport.Deps) error {
	return x.Final.Start(d)
}

func (x chainExec) Stop() error {
	return x.Final.Stop()
}

func (x chainExec) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if len(x.Chain) == 0 {
		return x.Final.Call(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.CallUnary(ctx, request, x)
}
