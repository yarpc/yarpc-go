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

package internalyarpcobservability

import (
	"context"
	"time"

	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
)

type fakeHandler struct {
	err            error
	applicationErr error
}

func (h fakeHandler) Handle(context.Context, *yarpc.Request, *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	if h.applicationErr != nil {
		errorInfo := yarpcerror.ExtractInfo(h.applicationErr)
		res := &yarpc.Response{ApplicationErrorInfo: &errorInfo}
		return res, nil, nil
	} else if h.err != nil {
		return nil, nil, h.err
	}
	return &yarpc.Response{}, &yarpc.Buffer{}, nil
}

func (h fakeHandler) HandleStream(*yarpc.ServerStream) error {
	return h.err
}

type fakeOutbound struct {
	err            error
	applicationErr error
}

func (o fakeOutbound) Call(context.Context, *yarpc.Request, *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	if o.applicationErr != nil {
		errorInfo := yarpcerror.ExtractInfo(o.applicationErr)
		res := &yarpc.Response{ApplicationErrorInfo: &errorInfo}
		return res, nil, nil
	} else if o.err != nil {
		return nil, nil, o.err
	}
	return &yarpc.Response{}, &yarpc.Buffer{}, nil
}

func (o fakeOutbound) CallStream(ctx context.Context, request *yarpc.Request) (*yarpc.ClientStream, error) {
	if o.err != nil {
		return nil, o.err
	}
	return yarpc.NewClientStream(&fakeStream{
		ctx:     ctx,
		request: request,
	})
}

var _ yarpc.Stream = (*fakeStream)(nil)

type fakeStream struct {
	ctx     context.Context
	request *yarpc.Request
}

func (s *fakeStream) Context() context.Context {
	return s.ctx
}

func (s *fakeStream) Request() *yarpc.Request {
	return s.request
}

func (s *fakeStream) SendMessage(context.Context, *yarpc.Buffer) error {
	return nil
}

func (s *fakeStream) ReceiveMessage(context.Context) (*yarpc.Buffer, error) {
	return nil, nil
}

func (s *fakeStream) Close(context.Context) error {
	return nil
}

func stubTime() func() {
	prev := _timeNow
	_timeNow = func() time.Time { return time.Time{} }
	return func() { _timeNow = prev }
}
