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

package yarpc

import (
	"context"

	"go.uber.org/yarpc/v2/yarpcerrors"
)

// OutboundCall is an outgoing call. It holds per-call options for a request.
//
// Encoding authors may use OutboundCall to provide a CallOption-based request
// customization mechanism, including returning response headers through
// ResponseHeaders.
type OutboundCall struct {
	// request attributes to fill if non-nil
	headers         []keyValuePair
	shardKey        *string
	routingKey      *string
	routingDelegate *string

	// If non-nil, response headers should be written here.
	responseHeaders *map[string]string
}

// NewOutboundCall constructs a new OutboundCall with the given options.
func NewOutboundCall(options ...CallOption) *OutboundCall {
	var call OutboundCall
	for _, opt := range options {
		opt.apply(&call)
	}
	return &call
}

// NewStreamOutboundCall constructs a new OutboundCall with the given
// options and enforces the OutboundCall is valid for streams.
func NewStreamOutboundCall(options ...CallOption) (*OutboundCall, error) {
	call := NewOutboundCall(options...)
	if call.responseHeaders != nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("response headers are not supported for streams")
	}
	return call, nil
}

// WriteToRequest fills the given request with request-specific options from
// the call.
//
// The context MAY be replaced by the OutboundCall.
func (c *OutboundCall) WriteToRequest(ctx context.Context, req *Request) (context.Context, error) {
	for _, h := range c.headers {
		req.Headers = req.Headers.With(h.k, h.v)
	}

	if c.shardKey != nil {
		req.ShardKey = *c.shardKey
	}
	if c.routingKey != nil {
		req.RoutingKey = *c.routingKey
	}
	if c.routingDelegate != nil {
		req.RoutingDelegate = *c.routingDelegate
	}

	// NB(abg): context and error are unused for now but we want to leave room
	// for CallOptions which can fail or modify the context.
	return ctx, nil
}

// WriteToRequestMeta fills the given request with request-specific options from
// the call.
//
// The context MAY be replaced by the OutboundCall.
func (c *OutboundCall) WriteToRequestMeta(ctx context.Context, reqMeta *RequestMeta) (context.Context, error) {
	for _, h := range c.headers {
		reqMeta.Headers = reqMeta.Headers.With(h.k, h.v)
	}

	if c.shardKey != nil {
		reqMeta.ShardKey = *c.shardKey
	}
	if c.routingKey != nil {
		reqMeta.RoutingKey = *c.routingKey
	}
	if c.routingDelegate != nil {
		reqMeta.RoutingDelegate = *c.routingDelegate
	}

	// NB(abg): context and error are unused for now but we want to leave room
	// for CallOptions which can fail or modify the context.
	return ctx, nil
}

// ReadFromResponse reads information from the response for this call.
//
// This should be called only if the request is unary.
func (c *OutboundCall) ReadFromResponse(ctx context.Context, res *Response) (context.Context, error) {
	// We're not using ctx right now but we may in the future.
	if c.responseHeaders != nil && res.Headers.Len() > 0 {
		// We make a copy of the response headers because Headers.Items() must
		// never be mutated.
		headers := make(map[string]string, res.Headers.Len())
		for k, v := range res.Headers.Items() {
			headers[k] = v
		}
		*c.responseHeaders = headers
	}

	// NB(abg): context and error are unused for now but we want to leave room
	// for CallOptions which can fail or modify the context.
	return ctx, nil
}
