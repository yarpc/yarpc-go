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

// isolating panic tests in separate package to avoid cyclic imports
package transport_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/zap"
)

// TODO: need to adjust the Dispatch* panic tests

func TestDispatchUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *transport.Request, transport.ResponseWriter) error {
		panic(msg)
	}

	err := transport.DispatchUnaryHandler(
		context.Background(),
		transport.UnaryHandlerFunc(handler),
		time.Now(),
		&transport.Request{},
		nil,
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}

func TestDispatchOnewayHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a oneway handler!"
	handler := func(context.Context, *transport.Request) error {
		panic(msg)
	}

	err := transport.DispatchOnewayHandler(
		context.Background(),
		transport.OnewayHandlerFunc(handler),
		&transport.Request{},
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}

func TestDispatchStreamHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a stream handler!"

	handler := func(_ *transport.ServerStream) error {
		panic(msg)
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := transporttest.NewMockStream(mockCtrl)
	mockStream.EXPECT().Request().Return(
		&transport.StreamRequest{
			Meta: &transport.RequestMeta{},
		}).Times(1)
	mockServerStream, _ := transport.NewServerStream(mockStream)
	err := transport.DispatchStreamHandler(
		transport.StreamHandlerFunc(handler),
		mockServerStream,
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}

func TestInvokeUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *transport.Request, transport.ResponseWriter) error {
		panic(msg)
	}

	err := transport.InvokeUnaryHandler(
		context.Background(),
		transport.UnaryHandlerFunc(handler),
		time.Now(),
		&transport.Request{},
		nil,
		zap.NewNop(),
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}

func TestInvokeOnewayHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a oneway handler!"
	handler := func(context.Context, *transport.Request) error {
		panic(msg)
	}

	err := transport.InvokeOnewayHandler(
		context.Background(),
		transport.OnewayHandlerFunc(handler),
		&transport.Request{},
		zap.NewNop(),
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}

func TestInvokeStreamHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a stream handler!"

	handler := func(_ *transport.ServerStream) error {
		panic(msg)
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := transporttest.NewMockStream(mockCtrl)
	mockStream.EXPECT().Request().Return(
		&transport.StreamRequest{
			Meta: &transport.RequestMeta{},
		}).Times(1)
	mockServerStream, _ := transport.NewServerStream(mockStream)
	err := transport.InvokeStreamHandler(
		transport.StreamHandlerFunc(handler),
		mockServerStream,
		zap.NewNop(),
	)
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, err.Error(), expectMsg)
}
