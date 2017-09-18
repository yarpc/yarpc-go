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

package raw

import (
	"context"
	"io/ioutil"

	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/errors"
)

// rawUnaryHandler adapts a Handler into a transport.UnaryHandler
type rawUnaryHandler struct{ UnaryHandler }

// rawOnewayHandler adapts a Handler into a transport.OnewayHandler
type rawOnewayHandler struct{ OnewayHandler }

// rawStreamHandler adapts a Handler into a transport.OnewayHandler
type rawStreamHandler struct{ StreamHandler }

func (r rawUnaryHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return err
	}

	ctx, call := encodingapi.NewInboundCall(ctx)
	if err := call.ReadFromRequest(treq); err != nil {
		return err
	}

	reqBody, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return err
	}

	resBody, appErr := r.UnaryHandler(ctx, reqBody)
	if err := call.WriteToResponse(rw); err != nil {
		return err
	}
	if appErr != nil {
		rw.SetApplicationError()
		return appErr
	}

	if _, err := rw.Write(resBody); err != nil {
		return err
	}

	return nil
}

func (r rawOnewayHandler) HandleOneway(ctx context.Context, treq *transport.Request) error {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return err
	}

	ctx, call := encodingapi.NewInboundCall(ctx)
	if err := call.ReadFromRequest(treq); err != nil {
		return err
	}

	reqBody, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return err
	}

	return r.OnewayHandler(ctx, reqBody)
}

func (r rawStreamHandler) HandleStream(stream transport.ServerStream) error {
	return r.StreamHandler(newServerStream(stream))
}
