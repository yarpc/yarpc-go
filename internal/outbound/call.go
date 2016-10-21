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

package outbound

import (
	"go.uber.org/yarpc/transport"

	"golang.org/x/net/context"
)

// OutboundCallFunc is an OutboundCall implemented as a single function.
type OutboundCallFunc func(
	context.Context, transport.Options, transport.RequestSender) (*transport.Response, error)

func (f OutboundCallFunc) WithRequest(ctx context.Context, opts transport.Options, req transport.RequestSender) (*transport.Response, error) {
	return f(ctx, opts, req)
}

// RequestSenderFunc is a RequestSender implemented as a single function.
type RequestSenderFunc func(context.Context, *transport.Request) (*transport.Response, error)

func (f RequestSenderFunc) Send(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	return f(ctx, req)
}

// verify that we don't break the interfaces
var (
	_ transport.OutboundCall  = (OutboundCallFunc)(nil)
	_ transport.RequestSender = (RequestSenderFunc)(nil)
)

// CallFromRequest builds an OutboundCall from the given request that does not
// vary based on the transport options.
func CallFromRequest(req *transport.Request) transport.OutboundCall {
	return (*callFromRequest)(req)
}

type callFromRequest transport.Request

func (r *callFromRequest) WithRequest(
	ctx context.Context,
	_ transport.Options,
	s transport.RequestSender,
) (*transport.Response, error) {
	return s.Send(ctx, (*transport.Request)(r))
}
