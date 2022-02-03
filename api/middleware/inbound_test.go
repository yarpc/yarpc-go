// Copyright (c) 2022 Uber Technologies, Inc.
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

package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnaryNopInboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := transporttest.NewMockUnaryHandler(mockCtrl)
	wrappedH := middleware.ApplyUnaryInbound(h, middleware.NopUnaryInbound)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	req := &transport.Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  raw.Encoding,
		Procedure: "hello",
		Body:      bytes.NewReader([]byte{1, 2, 3}),
	}
	resw := new(transporttest.FakeResponseWriter)
	err := errors.New("great sadness")
	h.EXPECT().Handle(ctx, req, resw).Return(err)

	assert.Equal(t, err, wrappedH.Handle(ctx, req, resw))
}

func TestOnewayNopInboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := transporttest.NewMockOnewayHandler(mockCtrl)
	wrappedH := middleware.ApplyOnewayInbound(h, middleware.NopOnewayInbound)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	req := &transport.Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  raw.Encoding,
		Procedure: "hello",
		Body:      bytes.NewReader([]byte{1, 2, 3}),
	}
	err := errors.New("great sadness")
	h.EXPECT().HandleOneway(ctx, req).Return(err)

	assert.Equal(t, err, wrappedH.HandleOneway(ctx, req))
}

func TestNilInboundMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := &transport.Request{}

	t.Run("unary", func(t *testing.T) {
		handler := transporttest.NewMockUnaryHandler(ctrl)
		mw := middleware.ApplyUnaryInbound(handler, nil)

		resWriter := &transporttest.FakeResponseWriter{}

		handler.EXPECT().Handle(ctx, req, resWriter)
		err := mw.Handle(ctx, req, resWriter)
		require.NoError(t, err, "unexpected error calling handler")
	})

	t.Run("oneway", func(t *testing.T) {
		handler := transporttest.NewMockOnewayHandler(ctrl)
		mw := middleware.ApplyOnewayInbound(handler, nil)

		handler.EXPECT().HandleOneway(ctx, req)
		err := mw.HandleOneway(ctx, req)
		require.NoError(t, err, "unexpected error calling handler")
	})
}

func TestStreamNopInboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := transporttest.NewMockStreamHandler(mockCtrl)
	wrappedH := middleware.ApplyStreamInbound(h, middleware.NopStreamInbound)
	s, err := transport.NewServerStream(transporttest.NewMockStream(mockCtrl))
	require.NoError(t, err)

	err = errors.New("great sadness")
	h.EXPECT().HandleStream(s).Return(err)

	assert.Equal(t, err, wrappedH.HandleStream(s))
}

func TestStreamDefaultsToHandlerWhenNil(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := transporttest.NewMockStreamHandler(mockCtrl)
	wrappedH := middleware.ApplyStreamInbound(h, nil)
	assert.Equal(t, wrappedH, h)
}
