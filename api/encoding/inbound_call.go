// Copyright (c) 2017 Uber Technologies, Inc.
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

package encoding

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// InboundCall holds information about the inbound call and its response.
//
// Encoding authors may use InboundCall to provide information about the
// incoming request on the Context and send response headers through
// WriteResponseHeader.
type InboundCall struct {
	resHeaders             []keyValuePair
	req                    *transport.Request
	disableResponseHeaders bool
}

type inboundCallKey struct{} // context key for *InboundCall

// NewInboundCall builds a new InboundCall with the given context.
//
// A request context is returned and must be used in place of the original.
func NewInboundCall(ctx context.Context) (context.Context, *InboundCall) {
	return NewInboundCallWithOptions(ctx)
}

// InboundCallOption is an option for configuring an InboundCall.
type InboundCallOption interface {
	apply(call *InboundCall)
}

type inboundCallOptionFunc func(*InboundCall)

func (i inboundCallOptionFunc) apply(call *InboundCall) { i(call) }

// DisableResponseHeaders disables response headers for inbound calls.
func DisableResponseHeaders() InboundCallOption {
	return inboundCallOptionFunc(func(call *InboundCall) {
		call.disableResponseHeaders = true
	})
}

// NewInboundCallWithOptions builds a new InboundCall with the given context and
// options.
//
// A request context is returned and must be used in place of the original.
func NewInboundCallWithOptions(ctx context.Context, opts ...InboundCallOption) (context.Context, *InboundCall) {
	call := &InboundCall{}
	for _, opt := range opts {
		opt.apply(call)
	}
	return context.WithValue(ctx, inboundCallKey{}, call), call
}

// getInboundCall returns the inbound call on this context or nil.
func getInboundCall(ctx context.Context) (*InboundCall, bool) {
	call, ok := ctx.Value(inboundCallKey{}).(*InboundCall)
	return call, ok
}

// ReadFromRequest reads information from the given request.
//
// This information may be queried on the context using functions like Caller,
// Service, Procedure, etc.
func (ic *InboundCall) ReadFromRequest(req *transport.Request) error {
	// TODO(abg): Maybe we should copy attributes over so that changes to the
	// Request don't change the output.
	ic.req = req
	return nil
}

// ReadFromRequestMeta reads information from the given request.
//
// This information may be queried on the context using functions like Caller,
// Service, Procedure, etc.
func (ic *InboundCall) ReadFromRequestMeta(reqMeta *transport.RequestMeta) error {
	ic.req = reqMeta.ToRequest()
	return nil
}

// WriteToResponse writes response information from the InboundCall onto the
// given ResponseWriter.
//
// If used, this must be called before writing the response body to the
// ResponseWriter.
func (ic *InboundCall) WriteToResponse(resw transport.ResponseWriter) error {
	var headers transport.Headers
	for _, h := range ic.resHeaders {
		headers = headers.With(h.k, h.v)
	}

	if headers.Len() > 0 {
		resw.AddHeaders(headers)
	}

	return nil
}
