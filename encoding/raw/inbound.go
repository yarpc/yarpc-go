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

package raw

import (
	"context"
	"fmt"
	"io/ioutil"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
)

// rawUnaryHandler adapts a Handler into a transport.UnaryHandler
type rawUnaryHandler struct{ UnaryHandler }

// rawOnewayHandler adapts a Handler into a transport.OnewayHandler
type rawOnewayHandler struct{ OnewayHandler }

type reqMeta struct{ *transport.Request }

func (rm reqMeta) Caller() string {
	return rm.Request.Caller
}

func (rm reqMeta) Encoding() transport.Encoding {
	return rm.Request.Encoding
}

func (rm reqMeta) Headers() yarpc.Headers {
	return yarpc.Headers(rm.Request.Headers)
}

func (rm reqMeta) Procedure() string {
	return rm.Request.Procedure
}

func (rm reqMeta) Service() string {
	return rm.Request.Service
}

func checkEncoding(treq *transport.Request) error {
	if treq.Encoding == Encoding {
		return nil
	}
	return transport.BadRequestError(fmt.Errorf(
		"failed to decode request body for procedure %q of service %q from caller %q: "+
			`expected encoding "raw" but got %q`, treq.Procedure, treq.Service, treq.Caller, treq.Encoding))
}

func (r rawUnaryHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	if err := checkEncoding(treq); err != nil {
		return err
	}

	reqBody, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return err
	}

	reqMeta := reqMeta{treq}
	resBody, resMeta, err := r.UnaryHandler(ctx, reqMeta, reqBody)
	if err != nil {
		return err
	}

	if resMeta != nil {
		headers := transport.Headers(resMeta.GetHeaders())
		if headers.Len() > 0 {
			rw.AddHeaders(headers)
		}
	}

	if _, err := rw.Write(resBody); err != nil {
		return err
	}

	return nil
}

func (r rawOnewayHandler) HandleOneway(ctx context.Context, treq *transport.Request) error {
	if err := checkEncoding(treq); err != nil {
		return err
	}

	reqBody, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return err
	}

	return r.OnewayHandler(ctx, reqMeta{treq}, reqBody)
}
