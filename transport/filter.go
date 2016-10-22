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
	Send(ctx context.Context, req *Request, sender RequestSender) (*Response, error)
}

// NopFilter is a filter that does not do anything special. It simply calls
// the underlying Outbound.
var NopFilter Filter = nopFilter{}

// ApplyFilter applies the given Filter to the given Outbound.
func ApplyFilter(o Outbound, f Filter) Outbound {
	if f == nil {
		return o
	}
	return filteredOutbound{Filter: f, Outbound: o}
}

// FilterFunc adapts a function into a Filter.
type FilterFunc func(context.Context, *Request, RequestSender) (*Response, error)

// Send for FilterFunc.
func (f FilterFunc) Send(ctx context.Context, req *Request, sender RequestSender) (*Response, error) {
	return f(ctx, req, sender)
}

type filteredOutbound struct {
	Filter   Filter
	Outbound Outbound
}

func (fo filteredOutbound) Start(d Deps) error {
	return fo.Outbound.Start(d)
}

func (fo filteredOutbound) Stop() error {
	return fo.Outbound.Stop()
}

func (fo filteredOutbound) Call(ctx context.Context, call OutboundCall) (*Response, error) {
	return fo.Outbound.Call(ctx, filteredCall{Filter: fo.Filter, Call: call})
}

type filteredCall struct {
	Filter Filter
	Call   OutboundCall
}

func (fc filteredCall) WithRequest(ctx context.Context, opts Options, sender RequestSender) (*Response, error) {
	return fc.Call.WithRequest(ctx, opts, filteredSender{Filter: fc.Filter, Sender: sender})
}

type filteredSender struct {
	Filter Filter
	Sender RequestSender
}

func (fs filteredSender) Send(ctx context.Context, req *Request) (*Response, error) {
	return fs.Filter.Send(ctx, req, fs.Sender)
}

type nopFilter struct{}

func (nopFilter) Send(ctx context.Context, req *Request, sender RequestSender) (*Response, error) {
	return sender.Send(ctx, req)
}
