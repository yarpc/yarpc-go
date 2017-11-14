// Copyright (c) 2017 Uber Technologies, Inc.
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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"
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

func TestStreamNopOutboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := transporttest.NewMockStreamOutbound(mockCtrl)
	wrappedO := middleware.ApplyStreamOutbound(o, middleware.NopStreamOutbound)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	req := &transport.RequestMeta{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  raw.Encoding,
		Procedure: "hello",
	}

	o.EXPECT().CallStream(ctx, req).Return(nil, nil)

	got, err := wrappedO.CallStream(ctx, req)
	if assert.NoError(t, err) {
		assert.Equal(t, nil, got)
	}
}
