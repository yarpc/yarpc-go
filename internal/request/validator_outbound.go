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
	"context"

	"go.uber.org/yarpc/transport"
)

// UnaryValidatorOutbound wraps an Outbound to validate all outgoing unary requests.
type UnaryValidatorOutbound struct{ transport.UnaryOutbound }

// OnewayValidatorOutbound wraps an Outbound to validate all outgoing oneway requests.
type OnewayValidatorOutbound struct{ transport.OnewayOutbound }

// Call performs the given request, failing early if the request is invalid.
func (o UnaryValidatorOutbound) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	request, err := Validate(ctx, request)
	if err != nil {
		return nil, err
	}

	request, err = ValidateUnary(ctx, request)
	if err != nil {
		return nil, err
	}

	return o.UnaryOutbound.Call(ctx, request)
}

// CallOneway performs the given request, failing early if the request is invalid.
func (o OnewayValidatorOutbound) CallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	request, err := Validate(ctx, request)
	if err != nil {
		return nil, err
	}

	request, err = ValidateOneway(ctx, request)
	if err != nil {
		return nil, err
	}

	return o.OnewayOutbound.CallOneway(ctx, request)
}
