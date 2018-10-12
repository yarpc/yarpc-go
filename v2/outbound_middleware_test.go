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

package yarpc_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaltesttime"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestUnaryNopOutboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := yarpctest.NewMockUnaryOutbound(mockCtrl)
	wrappedO := ApplyUnaryOutboundTransportMiddleware(o, NopUnaryOutboundTransportMiddleware)

	ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
	defer cancel()
	req := &Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  Encoding("raw"),
		Procedure: "hello",
	}
	reqBuf := NewBufferBytes([]byte{1, 2, 3})
	resp := &Response{}
	respBuf := NewBufferBytes([]byte{4, 5, 6})
	o.EXPECT().Call(ctx, req, reqBuf).Return(resp, respBuf, nil)

	gotResp, gotRespBuf, err := wrappedO.Call(ctx, req, reqBuf)
	require.NoError(t, err)

	assert.Equal(t, resp, gotResp)
	assert.Equal(t, respBuf, gotRespBuf)
}

func TestNilOutboundMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("unary", func(t *testing.T) {
		out := yarpctest.NewMockUnaryOutbound(ctrl)
		_ = ApplyUnaryOutboundTransportMiddleware(out, nil)
	})
}

func TestOutboundMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("unary", func(t *testing.T) {
		out := yarpctest.NewMockUnaryOutbound(ctrl)
		mw := yarpctest.NewMockUnaryOutboundTransportMiddleware(ctrl)
		_ = ApplyUnaryOutboundTransportMiddleware(out, mw)
	})
}

func TestStreamNopOutboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := yarpctest.NewMockStreamOutbound(mockCtrl)
	wrappedO := ApplyStreamOutboundTransportMiddleware(o, NopStreamOutboundTransportMiddleware)

	ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
	defer cancel()
	req := &Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  Encoding("raw"),
		Procedure: "hello",
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

	o := yarpctest.NewMockStreamOutbound(mockCtrl)
	wrappedO := ApplyStreamOutboundTransportMiddleware(o, nil)
	assert.Equal(t, wrappedO, o)
}

func TestStreamMiddlewareCallsUnderlyingFunctions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := yarpctest.NewMockStreamOutbound(mockCtrl)
	_ = ApplyStreamOutboundTransportMiddleware(o, NopStreamOutboundTransportMiddleware)
}
