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

package api

import (
	"bytes"
	"context"
	"testing"
	"time"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

// RequestOpts are configuration options for a yarpc Request and assertions
// to make on the response.
type RequestOpts struct {
	Port            uint16
	UnaryMiddleware []middleware.UnaryOutbound
	GiveTimeout     time.Duration
	GiveRequest     *transport.Request
	WantResponse    *transport.Response
	WantError       error
}

// NewRequestOpts initializes a RequestOpts struct.
func NewRequestOpts() RequestOpts {
	return RequestOpts{
		GiveTimeout: time.Second * 10,
		GiveRequest: &transport.Request{
			Caller:   "unknown",
			Encoding: transport.Encoding("raw"),
			Headers:  transport.NewHeaders(),
			Body:     bytes.NewBufferString(""),
		},
		WantResponse: &transport.Response{
			Headers: transport.NewHeaders(),
		},
	}
}

// RequestOption can be used to configure a request.
type RequestOption interface {
	ApplyRequest(*RequestOpts)
}

// RequestOptionFunc converts a function into a RequestOption.
type RequestOptionFunc func(*RequestOpts)

// ApplyRequest implements RequestOption.
func (f RequestOptionFunc) ApplyRequest(opts *RequestOpts) { f(opts) }

// UnaryOutboundMiddleware is a wrapper around the middleware.UnaryOutbound and
// Lifecycle interfaces.
type UnaryOutboundMiddleware interface {
	Lifecycle
	middleware.UnaryOutbound
}

var _ UnaryOutboundMiddleware = (UnaryOutboundMiddlewareFunc)(nil)

// UnaryOutboundMiddlewareFunc converts a function into a transport.UnaryOutboundMiddleware.
type UnaryOutboundMiddlewareFunc func(context.Context, *transport.Request, transport.UnaryOutbound) (*transport.Response, error)

// Call implements yarpc/api/transport#UnaryOutboundMiddleware.
func (f UnaryOutboundMiddlewareFunc) Call(ctx context.Context, req *transport.Request, next transport.UnaryOutbound) (*transport.Response, error) {
	return f(ctx, req, next)
}

// Start is a noop for wrapped functions.
func (f UnaryOutboundMiddlewareFunc) Start(testing.TB) error { return nil }

// Stop is a noop for wrapped functions.
func (f UnaryOutboundMiddlewareFunc) Stop(testing.TB) error { return nil }
