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

package request

import (
	"context"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
)

// UnaryValidatorOutbound wraps an Outbound to validate all outgoing unary requests.
type UnaryValidatorOutbound struct{ transport.UnaryOutbound }

// Call performs the given request, failing early if the request is invalid.
func (o UnaryValidatorOutbound) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if err := transport.ValidateRequest(request); err != nil {
		return nil, err
	}

	if err := transport.ValidateRequestContext(ctx); err != nil {
		return nil, err
	}

	return o.UnaryOutbound.Call(ctx, request)
}

// Introspect returns the introspection status of the underlying outbound.
func (o UnaryValidatorOutbound) Introspect() introspection.OutboundStatus {
	if o, ok := o.UnaryOutbound.(introspection.IntrospectableOutbound); ok {
		return o.Introspect()
	}
	return introspection.OutboundStatusNotSupported
}

// OnewayValidatorOutbound wraps an Outbound to validate all outgoing oneway requests.
type OnewayValidatorOutbound struct{ transport.OnewayOutbound }

// CallOneway performs the given request, failing early if the request is invalid.
func (o OnewayValidatorOutbound) CallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	if err := transport.ValidateRequest(request); err != nil {
		return nil, err
	}

	if err := transport.ValidateRequestContext(ctx); err != nil {
		return nil, err
	}

	return o.OnewayOutbound.CallOneway(ctx, request)
}

// Introspect returns the introspection status of the underlying outbound.
func (o OnewayValidatorOutbound) Introspect() introspection.OutboundStatus {
	if o, ok := o.OnewayOutbound.(introspection.IntrospectableOutbound); ok {
		return o.Introspect()
	}
	return introspection.OutboundStatusNotSupported
}

// StreamValidatorOutbound wraps an Outbound to validate all outgoing oneway requests.
type StreamValidatorOutbound struct{ transport.StreamOutbound }

// CallStream performs the given request, failing early if the request is invalid.
func (o StreamValidatorOutbound) CallStream(ctx context.Context, request *transport.StreamRequest) (transport.ClientStream, error) {
	if err := transport.ValidateRequest(request.Meta.ToRequest()); err != nil {
		return nil, err
	}

	return o.StreamOutbound.CallStream(ctx, request)
}
