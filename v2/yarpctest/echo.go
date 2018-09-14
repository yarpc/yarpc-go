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

package yarpctest

import (
	"context"
	"io"

	"go.uber.org/yarpc/api/transport"
	yarpc "go.uber.org/yarpc/v2"
)

// EchoRouter is a router that echoes all unary inbound requests, with no
// explicit procedures.
type EchoRouter struct{}

// Procedures returns no explicitly supported procedures.
func (EchoRouter) Procedures() []transport.Procedure {
	return nil
}

// Choose always returns a unary echo handler.
func (EchoRouter) Choose(ctx context.Context, req *yarpc.Request) (transport.HandlerSpec, error) {
	return echoHandlerSpec, nil
}

// EchoHandler is a unary handler that echoes the request body on the response
// body for any inbound request.
type EchoHandler struct{}

// Handle handles an inbound request by copying the request body to the
// response body.
func (EchoHandler) Handle(ctx context.Context, req *yarpc.Request) (*yarpc.Response, error) {
	_, err := io.Copy(resw, req.Body)
	return nil, err
}

var echoHandlerSpec = transport.NewUnaryHandlerSpec(EchoHandler{})
