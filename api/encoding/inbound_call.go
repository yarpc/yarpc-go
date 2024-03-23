// Copyright (c) 2022 Uber Technologies, Inc.
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
	"net/netip"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/inboundcall"
	"go.uber.org/yarpc/yarpcerrors"
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
	return inboundcall.WithMetadata(ctx, (*inboundCallMetadata)(call)), call
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

// inboundCallMetadata wraps an InboundCall to implement inboundcall.Metadata.
type inboundCallMetadata InboundCall

var _ inboundcall.Metadata = (*inboundCallMetadata)(nil)

func (ic *inboundCallMetadata) Caller() string {
	return ic.req.Caller
}

func (ic *inboundCallMetadata) Service() string {
	return ic.req.Service
}

func (ic *inboundCallMetadata) Transport() string {
	return ic.req.Transport
}

func (ic *inboundCallMetadata) Procedure() string {
	return ic.req.Procedure
}

func (ic *inboundCallMetadata) Encoding() transport.Encoding {
	return ic.req.Encoding
}

func (ic *inboundCallMetadata) Headers() transport.Headers {
	return ic.req.Headers
}

func (ic *inboundCallMetadata) ShardKey() string {
	return ic.req.ShardKey
}

func (ic *inboundCallMetadata) RoutingKey() string {
	return ic.req.RoutingKey
}

func (ic *inboundCallMetadata) RoutingDelegate() string {
	return ic.req.RoutingDelegate
}

func (ic *inboundCallMetadata) CallerProcedure() string {
	return ic.req.CallerProcedure
}

func (ic *inboundCallMetadata) CallerPeerAddrPort() netip.AddrPort {
	return ic.req.CallerPeerAddrPort
}

func (ic *inboundCallMetadata) WriteResponseHeader(k, v string) error {
	if ic.disableResponseHeaders {
		return yarpcerrors.InvalidArgumentErrorf("call does not support setting response headers")
	}
	ic.resHeaders = append(ic.resHeaders, keyValuePair{k: k, v: v})
	return nil
}
