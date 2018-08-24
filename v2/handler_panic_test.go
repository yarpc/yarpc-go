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

package yarpctransport_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2/yarpctransport"
	"go.uber.org/yarpc/v2/yarpctransporttest"
	"go.uber.org/zap"
)

func TestDispatchUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *yarpctransport.Request, yarpctransport.ResponseWriter) error {
		panic(msg)
	}
	var err error
	require.NotPanics(t, func() {
		err = yarpctransport.DispatchUnaryHandler(
			context.Background(),
			yarpctransport.UnaryHandlerFunc(handler),
			time.Now(),
			&yarpctransport.Request{},
			nil,
		)
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestDispatchStreamHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a stream handler!"
	handler := func(*yarpctransport.ServerStream) error {
		panic(msg)
	}
	var err error

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := yarpctransporttest.NewMockStream(mockCtrl)
	mockStream.EXPECT().Request().Return(
		&yarpctransport.StreamRequest{
			Meta: &yarpctransport.RequestMeta{},
		}).Times(1)
	mockServerStream, err := yarpctransport.NewServerStream(mockStream)
	require.NoError(t, err, "Should create mockServerStream")
	require.NotPanics(t, func() {
		err = yarpctransport.DispatchStreamHandler(
			yarpctransport.StreamHandlerFunc(handler),
			mockServerStream,
		)
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestInvokeUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *yarpctransport.Request, yarpctransport.ResponseWriter) error {
		panic(msg)
	}
	var err error
	require.NotPanics(t, func() {
		err = yarpctransport.InvokeUnaryHandler(
			yarpctransport.UnaryInvokeRequest{
				Context:   context.Background(),
				StartTime: time.Now(),
				Request:   &yarpctransport.Request{},
				Handler:   yarpctransport.UnaryHandlerFunc(handler),
				Logger:    zap.NewNop(),
			},
		)
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestInvokeStreamHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a stream handler!"
	handler := func(*yarpctransport.ServerStream) error {
		panic(msg)
	}
	var err error

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := yarpctransporttest.NewMockStream(mockCtrl)
	mockStream.EXPECT().Request().Return(
		&yarpctransport.StreamRequest{
			Meta: &yarpctransport.RequestMeta{},
		}).Times(1)
	mockServerStream, err := yarpctransport.NewServerStream(mockStream)
	require.NoError(t, err, "should create mockServerStream")
	require.NotPanics(t, func() {
		err = yarpctransport.InvokeStreamHandler(yarpctransport.StreamInvokeRequest{
			Stream:  mockServerStream,
			Handler: yarpctransport.StreamHandlerFunc(handler),
			Logger:  zap.NewNop(),
		})
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}
