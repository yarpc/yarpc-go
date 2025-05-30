// Copyright (c) 2025 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software are
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF TORT, TORT OR OTHERWISE, ARISING FROM,
// THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package tracinginterceptor

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

type mockClientStream struct {
	ctx        context.Context
	req        *transport.StreamRequest
	sendMsg    func(context.Context, *transport.StreamMessage) error
	receiveMsg func(context.Context) (*transport.StreamMessage, error)
	close      func(context.Context) error
	headers    func() (transport.Headers, error)
}

func (m *mockClientStream) Context() context.Context {
	return m.ctx
}

func (m *mockClientStream) Request() *transport.StreamRequest {
	return m.req
}

func (m *mockClientStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	return m.sendMsg(ctx, msg)
}

func (m *mockClientStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	return m.receiveMsg(ctx)
}

func (m *mockClientStream) Close(ctx context.Context) error {
	return m.close(ctx)
}

func (m *mockClientStream) Headers() (transport.Headers, error) {
	return m.headers()
}

type mockServerStream struct {
	ctx         context.Context
	req         *transport.StreamRequest
	sendMsg     func(context.Context, *transport.StreamMessage) error
	receiveMsg  func(context.Context) (*transport.StreamMessage, error)
	sendHeaders func(transport.Headers) error
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) Request() *transport.StreamRequest {
	return m.req
}

func (m *mockServerStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	return m.sendMsg(ctx, msg)
}

func (m *mockServerStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	return m.receiveMsg(ctx)
}

func (m *mockServerStream) SendHeaders(h transport.Headers) error {
	return m.sendHeaders(h)
}

func TestTracedClientStream(t *testing.T) {
	tracer := mocktracer.New()
	span := tracer.StartSpan("test")
	ctx := context.Background()

	clientStream := &mockClientStream{
		ctx: ctx,
		req: &transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Caller:    "test-caller",
				Service:   "test-service",
				Procedure: "test-procedure",
			},
		},
		sendMsg: func(ctx context.Context, msg *transport.StreamMessage) error {
			return nil
		},
		receiveMsg: func(ctx context.Context) (*transport.StreamMessage, error) {
			return &transport.StreamMessage{}, nil
		},
		close: func(ctx context.Context) error {
			return nil
		},
		headers: func() (transport.Headers, error) {
			return transport.Headers{}, nil
		},
	}

	clientStreamWrapper, err := transport.NewClientStream(clientStream)
	assert.NoError(t, err)

	tracedStream := &tracedClientStream{
		clientStream: clientStreamWrapper,
		span:         span,
	}

	// Test successful SendMessage
	err = tracedStream.SendMessage(ctx, &transport.StreamMessage{})
	assert.NoError(t, err)

	// Test successful ReceiveMessage
	msg, err := tracedStream.ReceiveMessage(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, msg)

	// Test successful Close
	err = tracedStream.Close(ctx)
	assert.NoError(t, err)

	// Test successful Headers
	headers, err := tracedStream.Headers()
	assert.NoError(t, err)
	assert.NotNil(t, headers)

	// Test error cases
	clientStream.sendMsg = func(ctx context.Context, msg *transport.StreamMessage) error {
		return errors.New("send error")
	}
	err = tracedStream.SendMessage(ctx, &transport.StreamMessage{})
	assert.Error(t, err)

	clientStream.receiveMsg = func(ctx context.Context) (*transport.StreamMessage, error) {
		return nil, errors.New("receive error")
	}
	msg, err = tracedStream.ReceiveMessage(ctx)
	assert.Error(t, err)
	assert.Nil(t, msg)

	clientStream.close = func(ctx context.Context) error {
		return errors.New("close error")
	}
	err = tracedStream.Close(ctx)
	assert.Error(t, err)

	clientStream.headers = func() (transport.Headers, error) {
		return transport.Headers{}, errors.New("headers error")
	}
	headers, err = tracedStream.Headers()
	assert.Error(t, err)
	assert.NotNil(t, headers)

	// Test EOF handling
	clientStream.receiveMsg = func(ctx context.Context) (*transport.StreamMessage, error) {
		return nil, io.EOF
	}
	msg, err = tracedStream.ReceiveMessage(ctx)
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, msg)
}

func TestTracedServerStream(t *testing.T) {
	tracer := mocktracer.New()
	span := tracer.StartSpan("test")
	ctx := context.Background()

	serverStream := &mockServerStream{
		ctx: ctx,
		req: &transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Caller:    "test-caller",
				Service:   "test-service",
				Procedure: "test-procedure",
			},
		},
		sendMsg: func(ctx context.Context, msg *transport.StreamMessage) error {
			return nil
		},
		receiveMsg: func(ctx context.Context) (*transport.StreamMessage, error) {
			return &transport.StreamMessage{}, nil
		},
		sendHeaders: func(h transport.Headers) error {
			return nil
		},
	}

	serverStreamWrapper, err := transport.NewServerStream(serverStream)
	assert.NoError(t, err)

	tracedStream := &tracedServerStream{
		serverStream: serverStreamWrapper,
		span:         span,
	}

	// Test successful SendMessage
	err = tracedStream.SendMessage(ctx, &transport.StreamMessage{})
	assert.NoError(t, err)

	// Test successful ReceiveMessage
	msg, err := tracedStream.ReceiveMessage(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, msg)

	// Test successful SendHeaders
	err = tracedStream.SendHeaders(transport.Headers{})
	assert.NoError(t, err)

	// Test error cases
	serverStream.sendMsg = func(ctx context.Context, msg *transport.StreamMessage) error {
		return errors.New("send error")
	}
	err = tracedStream.SendMessage(ctx, &transport.StreamMessage{})
	assert.Error(t, err)

	serverStream.receiveMsg = func(ctx context.Context) (*transport.StreamMessage, error) {
		return nil, errors.New("receive error")
	}
	msg, err = tracedStream.ReceiveMessage(ctx)
	assert.Error(t, err)
	assert.Nil(t, msg)

	serverStream.sendHeaders = func(h transport.Headers) error {
		return errors.New("headers error")
	}
	err = tracedStream.SendHeaders(transport.Headers{})
	assert.Error(t, err)

	// Test EOF handling
	serverStream.receiveMsg = func(ctx context.Context) (*transport.StreamMessage, error) {
		return nil, io.EOF
	}
	msg, err = tracedStream.ReceiveMessage(ctx)
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, msg)
}
