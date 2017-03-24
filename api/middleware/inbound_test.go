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
	"errors"
	"testing"
	"time"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestUnaryNopInboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := transporttest.NewMockUnaryHandler(mockCtrl)
	wrappedH := middleware.ApplyUnaryInbound(h, middleware.NopUnaryInbound)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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
