// Copyright (c) 2020 Uber Technologies, Inc.
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
	"context"
	"time"

	"go.uber.org/yarpc/api/transport"
)

type fakeAck struct{}

func (a fakeAck) String() string { return "" }

type fakeHandler struct {
	err              error
	applicationErr   bool
	applicationPanic bool
	handleStream     func(*transport.ServerStream)
}

func (h fakeHandler) Handle(_ context.Context, _ *transport.Request, rw transport.ResponseWriter) error {
	if h.applicationErr {
		rw.SetApplicationError()
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

	err            error
	applicationErr bool
	stream         fakeStream
}

func (o fakeOutbound) Call(context.Context, *transport.Request) (*transport.Response, error) {
	if o.err != nil {
		return nil, o.err
	}
	return &transport.Response{ApplicationError: o.applicationErr}, nil
}

func (o fakeOutbound) CallOneway(context.Context, *transport.Request) (transport.Ack, error) {
	if o.err != nil {
		return nil, o.err
	}
	return fakeAck{}, nil
}

func (o fakeOutbound) CallStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
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
	return nil, s.receiveErr
}

func (s *fakeStream) Close(context.Context) error {
	return s.closeErr
}

func stubTime() func() {
	prev := _timeNow
	_timeNow = func() time.Time { return time.Time{} }
	return func() { _timeNow = prev }
}
