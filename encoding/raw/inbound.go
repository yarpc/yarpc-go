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
	"io/ioutil"

	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/meta"
	"go.uber.org/yarpc/transport"
)

// rawHandler adapts a Handler into a transport.UnaryHandler and transport.OnewayHandler
type rawHandler struct {
	UnaryHandler  UnaryHandler
	OnewayHandler OnewayHandler
}

func (r rawHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	if err := encoding.Expect(treq, Encoding); err != nil {
		return err
	}

	reqBody, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return err
	}

	reqMeta := meta.FromTransportRequest(treq)
	resBody, resMeta, err := r.UnaryHandler(ctx, reqMeta, reqBody)
	if err != nil {
		return err
	}

	if resMeta != nil {
		meta.ToTransportResponseWriter(resMeta, rw)
	}

	if _, err := rw.Write(resBody); err != nil {
		return err
	}

	return nil
}

func (r rawHandler) HandleOneway(ctx context.Context, treq *transport.Request) error {
	if err := encoding.Expect(treq, Encoding); err != nil {
		return err
	}

	reqBody, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return err
	}

	reqMeta := meta.FromTransportRequest(treq)
	err = r.OnewayHandler(ctx, reqMeta, reqBody)
	if err != nil {
		return err
	}

	return nil
}
