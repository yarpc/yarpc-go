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

import "context"

// UnaryFilter defines transport-level middleware for Outbounds.
//
// Filters MAY
//
// - change the context
// - change the request
// - change the returned response
// - handle the returned error
// - call the given outbound zero or more times
//
// Filters MUST
//
// - always return a non-nil Response or error.
// - be thread-safe
//
// Filters are re-used across requests and MAY be called multiple times on the
// same request.
type UnaryFilter interface {
	CallUnary(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error)
}

// NopUnaryFilter is a filter that does not do anything special. It simply calls
// the underlying Outbound.
var NopUnaryFilter UnaryFilter = nopFilter{}

// ApplyFilter applies the given Filter to the given Outbound.
func ApplyFilter(o UnaryOutbound, f UnaryFilter) UnaryOutbound {
	if f == nil {
		return o
	}
	return filteredOutbound{o: o, f: f}
}

// FilterFunc adapts a function into a Filter.
type FilterFunc func(context.Context, *Request, UnaryOutbound) (*Response, error)

// CallUnary for FilterFunc.
func (f FilterFunc) CallUnary(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error) {
	return f(ctx, request, out)
}

type filteredOutbound struct {
	o UnaryOutbound
	f UnaryFilter
}

func (fo filteredOutbound) Start(d Deps) error {
	return fo.o.Start(d)
}

func (fo filteredOutbound) Stop() error {
	return fo.o.Stop()
}

func (fo filteredOutbound) CallUnary(ctx context.Context, request *Request) (*Response, error) {
	return fo.f.CallUnary(ctx, request, fo.o)
}

type nopFilter struct{}

func (nopFilter) CallUnary(ctx context.Context, request *Request, out UnaryOutbound) (*Response, error) {
	return out.CallUnary(ctx, request)
}
