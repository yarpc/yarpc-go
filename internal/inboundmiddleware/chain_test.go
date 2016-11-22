// Copyright (c) 2016 Uber Technologies, Inc.
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

package inboundmiddleware

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type countInboundMiddleware struct{ Count int }

func (c *countInboundMiddleware) Handle(
	ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	c.Count++
	return h.Handle(ctx, req, resw)
}

func (c *countInboundMiddleware) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	c.Count++
	return h.HandleOneway(ctx, req)
}

var retryUnaryInboundMiddleware transport.UnaryInboundMiddlewareFunc = func(
	ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	if err := h.Handle(ctx, req, resw); err != nil {
		return h.Handle(ctx, req, resw)
	}
	return nil
}

func TestUnaryChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &transport.Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  transport.Encoding("raw"),
		Procedure: "hello",
		Body:      bytes.NewReader([]byte{1, 2, 3}),
	}
	resw := new(transporttest.FakeResponseWriter)

	h := transporttest.NewMockUnaryHandler(mockCtrl)
	h.EXPECT().Handle(ctx, req, resw).After(
		h.EXPECT().Handle(ctx, req, resw).Return(errors.New("great sadness")),
	).Return(nil)

	before := &countInboundMiddleware{}
	after := &countInboundMiddleware{}
	err := transport.ApplyUnaryInboundMiddleware(
		h, UnaryChain(before, retryUnaryInboundMiddleware, after),
	).Handle(ctx, req, resw)

	assert.NoError(t, err, "expected success")
	assert.Equal(t, 1, before.Count, "expected outer inbound middleware to be called once")
	assert.Equal(t, 2, after.Count, "expected inner inbound middleware to be called twice")
}

var retryOnewayInboundMiddleware transport.OnewayInboundMiddlewareFunc = func(
	ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	if err := h.HandleOneway(ctx, req); err != nil {
		return h.HandleOneway(ctx, req)
	}
	return nil
}

func TestOnewayChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &transport.Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  transport.Encoding("raw"),
		Procedure: "hello",
		Body:      bytes.NewReader([]byte{1, 2, 3}),
	}

	h := transporttest.NewMockOnewayHandler(mockCtrl)
	h.EXPECT().HandleOneway(ctx, req).After(
		h.EXPECT().HandleOneway(ctx, req).Return(errors.New("great sadness")),
	).Return(nil)

	before := &countInboundMiddleware{}
	after := &countInboundMiddleware{}
	err := transport.ApplyOnewayInboundMiddleware(
		h, OnewayChain(before, retryOnewayInboundMiddleware, after),
	).HandleOneway(ctx, req)

	assert.NoError(t, err, "expected success")
	assert.Equal(t, 1, before.Count, "expected outer inbound middleware to be called once")
	assert.Equal(t, 2, after.Count, "expected inner inbound middleware to be called twice")
}
