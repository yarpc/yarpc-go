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

package transport_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/zap"
)

func TestDispatchUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *transport.Request, transport.ResponseWriter) error {
		panic(msg)
	}
	var err error
	require.NotPanics(t, func() {
		err = transport.DispatchUnaryHandler(
			context.Background(),
			transport.UnaryHandlerFunc(handler),
			time.Now(),
			&transport.Request{},
			nil,
		)
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestDispatchOnewayHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a oneway handler!"
	handler := func(context.Context, *transport.Request) error {
		panic(msg)
	}
	var err error
	require.NotPanics(t, func() {
		err = transport.DispatchOnewayHandler(
			context.Background(),
			transport.OnewayHandlerFunc(handler),
			&transport.Request{},
		)
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestDispatchStreamHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a stream handler!"
	handler := func(*transport.ServerStream) error {
		panic(msg)
	}
	var err error

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := transporttest.NewMockStream(mockCtrl)
	mockStream.EXPECT().Request().Return(
		&transport.StreamRequest{
			Meta: &transport.RequestMeta{},
		}).Times(1)
	mockServerStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err, "Should create mockServerStream")
	require.NotPanics(t, func() {
		err = transport.DispatchStreamHandler(
			transport.StreamHandlerFunc(handler),
			mockServerStream,
		)
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestInvokeUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *transport.Request, transport.ResponseWriter) error {
		panic(msg)
	}
	var err error
	require.NotPanics(t, func() {
		err = transport.InvokeUnaryHandler(
			transport.UnaryInvokeRequest{
				Context:   context.Background(),
				StartTime: time.Now(),
				Request:   &transport.Request{},
				Handler:   transport.UnaryHandlerFunc(handler),
				Logger:    zap.NewNop(),
			},
		)
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestInvokeOnewayHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a oneway handler!"
	handler := func(context.Context, *transport.Request) error {
		panic(msg)
	}
	var err error
	require.NotPanics(t, func() {
		err = transport.InvokeOnewayHandler(transport.OnewayInvokeRequest{
			Context: context.Background(),
			Request: &transport.Request{},
			Handler: transport.OnewayHandlerFunc(handler),
			Logger:  zap.NewNop(),
		})
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestInvokeStreamHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a stream handler!"
	handler := func(*transport.ServerStream) error {
		panic(msg)
	}
	var err error

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := transporttest.NewMockStream(mockCtrl)
	mockStream.EXPECT().Request().Return(
		&transport.StreamRequest{
			Meta: &transport.RequestMeta{},
		}).Times(1)
	mockServerStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err, "should create mockServerStream")
	require.NotPanics(t, func() {
		err = transport.InvokeStreamHandler(transport.StreamInvokeRequest{
			Stream:  mockServerStream,
			Handler: transport.StreamHandlerFunc(handler),
			Logger:  zap.NewNop(),
		})
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}
