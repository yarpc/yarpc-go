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

package request

import (
	"go.uber.org/yarpc/transport"

	"golang.org/x/net/context"
)

// ValidatorOutbound wraps an Outbound to validate all outgoing requests.
type ValidatorOutbound struct{ transport.Outbound }

// Call performs the given request, failing early if the request is invalid.
func (o ValidatorOutbound) Call(ctx context.Context, call transport.OutboundCall) (*transport.Response, error) {
	return o.Outbound.Call(ctx, validatorCall{call})
}

type validatorCall struct{ transport.OutboundCall }

func (c validatorCall) WithRequest(ctx context.Context, opts transport.Options, sender transport.RequestSender) (*transport.Response, error) {
	return c.OutboundCall.WithRequest(ctx, opts, validatorSender{sender})
}

type validatorSender struct{ transport.RequestSender }

func (s validatorSender) Send(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	req, err := Validate(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.RequestSender.Send(ctx, req)
}
