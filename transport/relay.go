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

import (
	"context"
	"io"
	"log"
)

// NewUnaryRelay creates an inbound unary request handler that relays a request
// to the given unary outbound caller. A relay is a utility for forwarding
// requests from a proxy or handle-or-forward registry.
func NewUnaryRelay(outbound UnaryOutbound) UnaryHandler {
	return &unaryRelay{outbound: outbound}
}

type unaryRelay struct {
	outbound UnaryOutbound
}

func (r *unaryRelay) Handle(ctx context.Context, request *Request, resw ResponseWriter) error {
	log.Printf("forwarding request from %v to %v to %v", request.Caller, request.Service, request.Procedure)

	res, err := r.outbound.Call(ctx, request)
	if err != nil {
		return err
	}

	resw.AddHeaders(res.Headers)

	_, err = io.Copy(resw, res.Body)
	if err != nil {
		return err
	}

	return nil
}
