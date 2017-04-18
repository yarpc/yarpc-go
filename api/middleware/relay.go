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

package middleware

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// NewRelayMiddleware returns a middleware that forwards all unrecognized
// procedures through a configured outbound client.
// The relay middleware currently only supports unary RPC.
func NewRelayMiddleware() *RelayMiddleware {
	return &RelayMiddleware{}
}

// RelayMiddleware forwards requests for unrecognized procedures to the
// configured client.
type RelayMiddleware struct {
	spec transport.HandlerSpec
}

// Procedures enumerates the recognized procedures through the middleware.
func (r *RelayMiddleware) Procedures(next transport.Router) []transport.Procedure {
	return next.Procedures()
}

// SetClientConfig configures the middleware to send requests through an
// outbound configured on a dispatcher, including through all of its outbound
// middleware.
func (r *RelayMiddleware) SetClientConfig(cc transport.ClientConfig) {
	// TODO support RPC types generally, not merely Unary
	r.spec = transport.NewUnaryHandlerSpec(transport.NewUnaryRelay(cc.GetUnaryOutbound()))
}

// Choose returns the spec for all recognized procedures or the spec for
// forwarding unrecognized procedures.
func (r *RelayMiddleware) Choose(ctx context.Context, req *transport.Request, next transport.Router) (transport.HandlerSpec, error) {
	handlerSpec, err := next.Choose(ctx, req)
	if transport.IsUnrecognizedProcedureError(err) {
		return r.spec, nil
	}
	return handlerSpec, err
}
