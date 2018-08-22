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

package yarpc

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/testtime"
)

var (
	retryUnaryInbound middleware.UnaryInboundFunc = func(
		ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
		if err := h.Handle(ctx, req, resw); err != nil {
			return h.Handle(ctx, req, resw)
		}
		return nil
	}

	retryUnaryOutbound middleware.UnaryOutboundFunc = func(
		ctx context.Context, req *transport.Request, o transport.UnaryOutbound) (*transport.Response, error) {
		res, err := o.Call(ctx, req)
		if err != nil {
			res, err = o.Call(ctx, req)
		}
		return res, err
	}
)

type countInboundMiddleware struct{ Count int }

func (c *countInboundMiddleware) Handle(
	ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	c.Count++
	return h.Handle(ctx, req, resw)
}

type countOutboundMiddleware struct{ Count int }

func (c *countOutboundMiddleware) Call(
	ctx context.Context, req *transport.Request, o transport.UnaryOutbound) (*transport.Response, error) {
	c.Count++
	return o.Call(ctx, req)
}

func TestUnaryInboundMiddleware(t *testing.T) {
	before := &countInboundMiddleware{}
	after := &countInboundMiddleware{}

	tests := []struct {
		desc string
		mw   middleware.UnaryInbound
	}{
		{"flat chain", UnaryInboundMiddleware(before, retryUnaryInbound, after, nil)},
		{"nested chain", UnaryInboundMiddleware(before, UnaryInboundMiddleware(retryUnaryInbound, nil, after))},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			before.Count, after.Count = 0, 0
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
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

			err := middleware.ApplyUnaryInbound(h, tt.mw).Handle(ctx, req, resw)

			assert.NoError(t, err, "expected success")
			assert.Equal(t, 1, before.Count, "expected outer inbound middleware to be called once")
			assert.Equal(t, 2, after.Count, "expected inner inbound middleware to be called twice")
		})
	}
}

func TestUnaryOutboundMiddleware(t *testing.T) {
	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}

	tests := []struct {
		desc string
		mw   middleware.UnaryOutbound
	}{
		{"flat chain", UnaryOutboundMiddleware(before, retryUnaryOutbound, nil, after)},
		{"nested chain", UnaryOutboundMiddleware(before, UnaryOutboundMiddleware(retryUnaryOutbound, after, nil))},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			before.Count, after.Count = 0, 0
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			req := &transport.Request{
				Caller:    "somecaller",
				Service:   "someservice",
				Encoding:  transport.Encoding("raw"),
				Procedure: "hello",
				Body:      bytes.NewReader([]byte{1, 2, 3}),
			}
			res := &transport.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte{4, 5, 6})),
			}
			o := transporttest.NewMockUnaryOutbound(mockCtrl)
			o.EXPECT().Call(ctx, req).After(
				o.EXPECT().Call(ctx, req).Return(nil, errors.New("great sadness")),
			).Return(res, nil)

			gotRes, err := middleware.ApplyUnaryOutbound(o, tt.mw).Call(ctx, req)

			assert.NoError(t, err, "expected success")
			assert.Equal(t, 1, before.Count, "expected outer middleware to be called once")
			assert.Equal(t, 2, after.Count, "expected inner middleware to be called twice")
			assert.Equal(t, res, gotRes, "expected response to match")
		})
	}
}

func TestStreamInboundMiddlewareChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	stream, err := transport.NewServerStream(transporttest.NewMockStream(mockCtrl))
	require.NoError(t, err)
	handler := transporttest.NewMockStreamHandler(mockCtrl)
	handler.EXPECT().HandleStream(stream)

	inboundMW := StreamInboundMiddleware(
		middleware.NopStreamInbound,
		middleware.NopStreamInbound,
		middleware.NopStreamInbound,
		middleware.NopStreamInbound,
	)

	h := middleware.ApplyStreamInbound(handler, inboundMW)

	assert.NoError(t, h.HandleStream(stream))
}

func TestStreamOutboundMiddlewareChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	req := &transport.StreamRequest{}

	stream, err := transport.NewClientStream(transporttest.NewMockStreamCloser(mockCtrl))
	require.NoError(t, err)

	out := transporttest.NewMockStreamOutbound(mockCtrl)
	out.EXPECT().CallStream(ctx, req).Return(stream, nil)

	mw := StreamOutboundMiddleware(
		middleware.NopStreamOutbound,
		middleware.NopStreamOutbound,
		middleware.NopStreamOutbound,
		middleware.NopStreamOutbound,
	)

	o := middleware.ApplyStreamOutbound(out, mw)

	s, err := o.CallStream(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, stream, s)
}
