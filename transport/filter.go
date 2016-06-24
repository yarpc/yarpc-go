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
type Filter interface {
	Call(ctx context.Context, request *Request, out Outbound) (*Response, error)
}

// NopFilter is a filter that does not do anything special. It simply calls
// the underlying Outbound.
var NopFilter Filter = nopFilter{}

// ApplyFilter applies the given Filter to the given Outbound.
func ApplyFilter(o Outbound, f Filter) Outbound {
	if f == nil {
		return o
	}
	return filteredOutbound{o: o, f: f}
}

// FilterFunc adapts a function into a Filter.
type FilterFunc func(context.Context, *Request, Outbound) (*Response, error)

// Call for FilterFunc.
func (f FilterFunc) Call(ctx context.Context, request *Request, out Outbound) (*Response, error) {
	return f(ctx, request, out)
}

type filteredOutbound struct {
	o Outbound
	f Filter
}

func (fo filteredOutbound) Start() error {
	return fo.o.Start()
}

func (fo filteredOutbound) Stop() error {
	return fo.o.Stop()
}

func (fo filteredOutbound) Call(ctx context.Context, request *Request) (*Response, error) {
	return fo.f.Call(ctx, request, fo.o)
}

type nopFilter struct{}

func (nopFilter) Call(ctx context.Context, request *Request, out Outbound) (*Response, error) {
	return out.Call(ctx, request)
}
