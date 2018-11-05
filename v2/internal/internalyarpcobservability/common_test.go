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
)

type fakeHandler struct {
	err            error
	applicationErr error
}

func (h fakeHandler) Handle(context.Context, *yarpc.Request, *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	res := &yarpc.Response{ApplicationError: h.applicationErr}
	if h.applicationErr != nil {
		return res, nil, nil
	}
	return res, nil, h.err
}

func (h fakeHandler) HandleStream(*yarpc.ServerStream) error {
	return h.err
}

type fakeOutbound struct {
	err            error
	applicationErr error
}

func (o fakeOutbound) Call(context.Context, *yarpc.Request, *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	if o.err != nil {
		return nil, nil, o.err
	}
	return &yarpc.Response{ApplicationError: o.applicationErr}, nil, nil
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

func (s *fakeStream) SendMessage(context.Context, *yarpc.StreamMessage) error {
	return nil
}

func (s *fakeStream) ReceiveMessage(context.Context) (*yarpc.StreamMessage, error) {
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
