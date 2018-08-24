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

package yarpcraw

import (
	"context"
	"io/ioutil"

	yarpc "go.uber.org/yarpc/v2"
)

// rawUnaryHandler adapts a Handler into a yarpc.UnaryHandler
type rawUnaryHandler struct{ UnaryHandler }

func (r rawUnaryHandler) Handle(ctx context.Context, treq *yarpc.Request, rw yarpc.ResponseWriter) error {
	if err := yarpc.ExpectEncodings(treq, Encoding); err != nil {
		return err
	}

	ctx, call := yarpc.NewInboundCall(ctx)
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

	// we want to return the appErr if it exists as this is what
	// the previous behavior was so we deprioritize this error
	var writeErr error
	if len(resBody) > 0 {
		_, writeErr = rw.Write(resBody)
	}
	if appErr != nil {
		rw.SetApplicationError()
		return appErr
	}
	return writeErr
}
