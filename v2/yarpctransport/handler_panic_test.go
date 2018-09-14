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
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctest"
	"go.uber.org/yarpc/v2/yarpctransport"
	"go.uber.org/zap"
)

func TestInvokeUnaryHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a unary handler!"
	handler := func(context.Context, *yarpc.Request) (*yarpc.Response, error) {
		panic(msg)
	}
	var err error
	require.NotPanics(t, func() {
		_, err = yarpctransport.InvokeUnaryHandler(
			yarpctransport.UnaryInvokeRequest{
				Context:   context.Background(),
				StartTime: time.Now(),
				Request:   &yarpc.Request{},
				Handler:   yarpc.UnaryHandlerFunc(handler),
				Logger:    zap.NewNop(),
			},
		)
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}

func TestInvokeStreamHandlerWithPanic(t *testing.T) {
	msg := "I'm panicking in a stream handler!"
	handler := func(*yarpc.ServerStream) error {
		panic(msg)
	}
	var err error

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := yarpctest.NewMockStream(mockCtrl)
	mockStream.EXPECT().Request().Return(
		&yarpc.StreamRequest{
			Meta: &yarpc.RequestMeta{},
		}).Times(1)
	mockServerStream, err := yarpc.NewServerStream(mockStream)
	require.NoError(t, err, "should create mockServerStream")
	require.NotPanics(t, func() {
		err = yarpctransport.InvokeStreamHandler(yarpctransport.StreamInvokeRequest{
			Stream:  mockServerStream,
			Handler: yarpc.StreamHandlerFunc(handler),
			Logger:  zap.NewNop(),
		})
	}, "Panic not recovered")
	expectMsg := fmt.Sprintf("panic: %s", msg)
	assert.Equal(t, expectMsg, err.Error())
}
