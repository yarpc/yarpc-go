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
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// Chain combines a series of filters into a single Filter.
func Chain(filters ...transport.Filter) transport.Filter {
	switch len(filters) {
	case 0:
		return transport.NopFilter
	case 1:
		return filters[0]
	default:
		return chain(filters)
	}
}

// filterChain combines a series of filters into a single Filter.
type chain []transport.Filter

func (c chain) Call(ctx context.Context, request *transport.Request, out transport.Outbound) (*transport.Response, error) {
	return chainExec{
		Chain: []transport.Filter(c),
		Final: out,
	}.Call(ctx, request)
}

// chainExec adapts a series of filters into an Outbound. It is scoped to a
// single call of an Outbound and is not thread-safe.
type chainExec struct {
	Chain []transport.Filter
	Final transport.Outbound
}

func (x chainExec) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if len(x.Chain) == 0 {
		return x.Final.Call(ctx, request)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.Call(ctx, request, x)
}
