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
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaltesttime"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestUnaryNopInboundTransportMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
	wrappedH := yarpc.ApplyUnaryInboundTransportMiddleware(h, yarpc.NopUnaryInboundTransportMiddleware)

	ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
	defer cancel()
	req := &yarpc.Request{
		Caller:    "somecaller",
		Service:   "someservice",
		Encoding:  yarpc.Encoding("raw"),
		Procedure: "hello",
	}
	reqBuf := yarpc.NewBufferBytes([]byte{1, 2, 3})

	err := errors.New("great sadness")
	h.EXPECT().Handle(ctx, req, reqBuf).Return(nil, nil, err)

	_, _, handleErr := wrappedH.Handle(ctx, req, reqBuf)
	assert.Equal(t, err, handleErr)
}

func TestNilInboundMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := &yarpc.Request{}

	t.Run("unary", func(t *testing.T) {
		handler := yarpctest.NewMockUnaryTransportHandler(ctrl)
		mw := yarpc.ApplyUnaryInboundTransportMiddleware(handler, nil)

		handler.EXPECT().Handle(ctx, req, &yarpc.Buffer{})
		_, _, err := mw.Handle(ctx, req, &yarpc.Buffer{})
		require.NoError(t, err, "unexpected error calling handler")
	})
}

func TestStreamNopInboundMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := yarpctest.NewMockStreamTransportHandler(mockCtrl)
	wrappedH := yarpc.ApplyStreamInboundTransportMiddleware(h, yarpc.NopStreamInboundTransportMiddleware)
	s, err := yarpc.NewServerStream(yarpctest.NewMockStream(mockCtrl))
	require.NoError(t, err)

	err = errors.New("great sadness")
	h.EXPECT().HandleStream(s).Return(err)

	assert.Equal(t, err, wrappedH.HandleStream(s))
}

func TestStreamDefaultsToHandlerWhenNil(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := yarpctest.NewMockStreamTransportHandler(mockCtrl)
	wrappedH := yarpc.ApplyStreamInboundTransportMiddleware(h, nil)
	assert.Equal(t, wrappedH, h)
}
