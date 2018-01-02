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

package middleware_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/middleware/middlewaretest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnaryNopOutboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := transporttest.NewMockUnaryOutbound(mockCtrl)
	wrappedO := middleware.ApplyUnaryOutbound(o, middleware.NopUnaryOutbound)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	req := &transport.Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  raw.Encoding,
		Procedure: "hello",
		Body:      bytes.NewReader([]byte{1, 2, 3}),
	}

	res := &transport.Response{Body: ioutil.NopCloser(bytes.NewReader([]byte{4, 5, 6}))}
	o.EXPECT().Call(ctx, req).Return(res, nil)

	got, err := wrappedO.Call(ctx, req)
	if assert.NoError(t, err) {
		assert.Equal(t, res, got)
	}
}

func TestOnewayNopOutboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := transporttest.NewMockOnewayOutbound(mockCtrl)
	wrappedO := middleware.ApplyOnewayOutbound(o, middleware.NopOnewayOutbound)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	req := &transport.Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  raw.Encoding,
		Procedure: "hello",
		Body:      bytes.NewReader([]byte{1, 2, 3}),
	}

	o.EXPECT().CallOneway(ctx, req).Return(nil, nil)

	got, err := wrappedO.CallOneway(ctx, req)
	if assert.NoError(t, err) {
		assert.Equal(t, nil, got)
	}
}

func TestNilOutboundMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("unary", func(t *testing.T) {
		out := transporttest.NewMockUnaryOutbound(ctrl)
		out.EXPECT().Start()

		mw := middleware.ApplyUnaryOutbound(out, nil)
		require.NoError(t, mw.Start())
	})

	t.Run("oneway", func(t *testing.T) {
		out := transporttest.NewMockOnewayOutbound(ctrl)
		out.EXPECT().Start()

		mw := middleware.ApplyOnewayOutbound(out, nil)
		require.NoError(t, mw.Start())
	})
}

func TestOutboundMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("unary", func(t *testing.T) {
		out := transporttest.NewMockUnaryOutbound(ctrl)
		mw := middlewaretest.NewMockUnaryOutbound(ctrl)
		outWithMW := middleware.ApplyUnaryOutbound(out, mw)

		// start
		out.EXPECT().Start().Return(nil)
		assert.NoError(t, outWithMW.Start(), "could not start outbound")

		// transports
		out.EXPECT().Transports()
		outWithMW.Transports()

		// is running
		out.EXPECT().IsRunning().Return(true)
		assert.True(t, outWithMW.IsRunning(), "expected outbound to be running")

		// stop
		out.EXPECT().Stop().Return(nil)
		assert.NoError(t, outWithMW.Stop(), "unexpected error stopping outbound")
	})

	t.Run("oneway", func(t *testing.T) {
		out := transporttest.NewMockOnewayOutbound(ctrl)
		mw := middlewaretest.NewMockOnewayOutbound(ctrl)
		outWithMW := middleware.ApplyOnewayOutbound(out, mw)

		// start
		out.EXPECT().Start().Return(nil)
		assert.NoError(t, outWithMW.Start(), "could not start outbound")

		// transports
		out.EXPECT().Transports()
		outWithMW.Transports()

		// is running
		out.EXPECT().IsRunning().Return(true)
		assert.True(t, outWithMW.IsRunning(), "expected outbound to be running")

		// stop
		out.EXPECT().Stop().Return(nil)
		assert.NoError(t, outWithMW.Stop(), "unexpected error stopping outbound")
	})
}

func TestStreamNopOutboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := transporttest.NewMockStreamOutbound(mockCtrl)
	wrappedO := middleware.ApplyStreamOutbound(o, middleware.NopStreamOutbound)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	req := &transport.StreamRequest{
		&transport.RequestMeta{
			Caller:    "somecaller",
			Service:   "someservice",
			Encoding:  raw.Encoding,
			Procedure: "hello",
		},
	}

	o.EXPECT().CallStream(ctx, req).Return(nil, nil)

	got, err := wrappedO.CallStream(ctx, req)
	if assert.NoError(t, err) {
		assert.Nil(t, got)
	}
}

func TestStreamDefaultsToOutboundWhenNil(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := transporttest.NewMockStreamOutbound(mockCtrl)
	wrappedO := middleware.ApplyStreamOutbound(o, nil)
	assert.Equal(t, wrappedO, o)
}

func TestStreamMiddlewareCallsUnderlyingFunctions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := transporttest.NewMockStreamOutbound(mockCtrl)
	o.EXPECT().Start().Times(1)
	o.EXPECT().Stop().Times(1)
	o.EXPECT().Transports().Times(1)
	o.EXPECT().IsRunning().Times(1)
	wrappedO := middleware.ApplyStreamOutbound(o, middleware.NopStreamOutbound)

	wrappedO.IsRunning()
	wrappedO.Transports()
	wrappedO.Start()
	wrappedO.Stop()
}
