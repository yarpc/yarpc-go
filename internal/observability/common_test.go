// Copyright (c) 2026 Uber Technologies, Inc.
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

package observability

import (
	"bytes"
	"context"
	"io"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

type fakeAck struct{}

func (a fakeAck) String() string { return "" }

type fakeHandler struct {
	err                   error
	applicationErr        bool
	applicationErrName    string
	applicationErrDetails string
	applicationErrCode    *yarpcerrors.Code
	applicationPanic      bool
	handleStream          func(*transport.ServerStream)
	responseData          []byte
}

func (h fakeHandler) Handle(_ context.Context, _ *transport.Request, rw transport.ResponseWriter) error {
	if h.applicationPanic {
		panic("application panicked")
	}
	if h.applicationErr {
		rw.SetApplicationError()

		if applicationErrorMetaSetter, ok := rw.(transport.ApplicationErrorMetaSetter); ok {
			applicationErrorMetaSetter.SetApplicationErrorMeta(&transport.ApplicationErrorMeta{
				Details: h.applicationErrDetails,
				Name:    h.applicationErrName,
				Code:    h.applicationErrCode,
			})
		}
	}

	if h.responseData != nil {
		rw.Write(h.responseData)
	}

	return h.err
}

func (h fakeHandler) HandleOneway(context.Context, *transport.Request) error {
	if h.applicationPanic {
		panic("application panicked")
	}
	return h.err
}

func (h fakeHandler) HandleStream(stream *transport.ServerStream) error {
	if h.applicationPanic {
		panic("application panicked")
	}
	if h.handleStream != nil {
		h.handleStream(stream)
	}
	return h.err
}

type fakeOutbound struct {
	transport.Outbound

	err                   error
	applicationErr        bool
	applicationErrName    string
	applicationErrDetails string
	applicationErrCode    *yarpcerrors.Code
	applicationPanic      bool
	stream                fakeStream

	body []byte
}

func (o fakeOutbound) Call(context.Context, *transport.Request) (*transport.Response, error) {
	if o.applicationPanic {
		panic("application panicked")
	}

	return &transport.Response{
		ApplicationError: o.applicationErr,
		ApplicationErrorMeta: &transport.ApplicationErrorMeta{
			Details: o.applicationErrDetails,
			Name:    o.applicationErrName,
			Code:    o.applicationErrCode,
		},
		Body:     io.NopCloser(bytes.NewReader(o.body)),
		BodySize: len(o.body)}, o.err
}

func (o fakeOutbound) CallOneway(context.Context, *transport.Request) (transport.Ack, error) {
	if o.applicationPanic {
		panic("application panicked")
	}

	if o.err != nil {
		return nil, o.err
	}
	return fakeAck{}, nil
}

func (o fakeOutbound) CallStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	if o.applicationPanic {
		panic("application panicked")
	}

	if o.err != nil {
		return nil, o.err
	}

	// always use passed in context and request
	stream := o.stream
	stream.ctx = ctx
	stream.request = request

	return transport.NewClientStream(&stream)
}

type fakeStream struct {
	ctx     context.Context
	request *transport.StreamRequest

	receiveMsg *transport.StreamMessage

	sendErr    error
	receiveErr error
	closeErr   error
}

func (s *fakeStream) Context() context.Context {
	return s.ctx
}

func (s *fakeStream) Request() *transport.StreamRequest {
	return s.request
}

func (s *fakeStream) SendMessage(context.Context, *transport.StreamMessage) error {
	return s.sendErr
}

func (s *fakeStream) ReceiveMessage(context.Context) (*transport.StreamMessage, error) {
	return s.receiveMsg, s.receiveErr
}

func (s *fakeStream) Close(context.Context) error {
	return s.closeErr
}

func stubTime() func() {
	return stubTimeWithTimeVal(time.Time{})
}

func stubTimeWithTimeVal(timeVal time.Time) func() {
	prev := _timeNow
	_timeNow = func() time.Time { return timeVal }
	return func() { _timeNow = prev }
}
