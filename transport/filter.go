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

package transport

import "golang.org/x/net/context"

// Filter defines transport-level middleware for Outbounds.
type Filter interface {
	Apply(ctx context.Context, request *Request, out Outbound) (*Response, error)
}

// NopFilter is a filter that does not do anything special. It simply calls
// the underlying Outbound.
var NopFilter Filter = nopFilter{}

// ApplyFilter wraps the given Outbound with the given Filters in-order.
func ApplyFilter(o Outbound, f Filter) Outbound {
	return filteredOutbound{o: o, f: f}
}

// ChainFilters combines the given filters in-order into a single Filter.
func ChainFilters(filters ...Filter) Filter {
	if len(filters) == 0 {
		return NopFilter
	}
	if len(filters) == 1 {
		return filters[0]
	}
	return filterChain(filters)
}

type filteredOutbound struct {
	o Outbound
	f Filter
}

func (fo filteredOutbound) Call(ctx context.Context, request *Request) (*Response, error) {
	return fo.f.Apply(ctx, request, fo.o)
}

type nopFilter struct{}

func (nopFilter) Apply(ctx context.Context, request *Request, out Outbound) (*Response, error) {
	return out.Call(ctx, request)
}

// filterChain combines a series of filters into a single Filter.
type filterChain []Filter

func (fc filterChain) Apply(ctx context.Context, request *Request, out Outbound) (*Response, error) {
	cx := filterChainExec{
		Chain: []Filter(fc),
		Index: 0,
		Final: out,
	}
	return cx.Call(ctx, request)
}

// filterChainExec adapts a series of filters into an Outbound. It is scoped to
// a single call of an Outbound and is not thread-safe.
type filterChainExec struct {
	Chain []Filter
	Index int
	Final Outbound
}

func (cx *filterChainExec) Call(ctx context.Context, request *Request) (*Response, error) {
	i := cx.Index
	if i == len(cx.Chain) {
		return cx.Final.Call(ctx, request)
	}

	cx.Index++
	return cx.Chain[i].Apply(ctx, request, cx)
}
