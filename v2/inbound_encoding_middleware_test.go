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
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaltesttime"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestUnaryNopInboundEncodingMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := yarpctest.NewMockUnaryEncodingHandler(mockCtrl)
	wrappedH := yarpc.ApplyUnaryInboundEncodingMiddleware(h, yarpc.NopUnaryInboundEncodingMiddleware)

	ctx, cancel := context.WithTimeout(context.Background(), internaltesttime.Second)
	defer cancel()
	reqBuf := yarpc.NewBufferBytes([]byte{1, 2, 3})

	err := errors.New("great sadness")
	h.EXPECT().Handle(ctx, reqBuf).Return(nil, err)

	_, handleErr := wrappedH.Handle(ctx, reqBuf)
	assert.Equal(t, err, handleErr)
}

func TestNilInboundEncodingMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	handler := yarpctest.NewMockUnaryEncodingHandler(ctrl)
	mw := yarpc.ApplyUnaryInboundEncodingMiddleware(handler, nil)

	handler.EXPECT().Handle(ctx, "test")
	_, err := mw.Handle(ctx, "test")
	require.NoError(t, err, "unexpected error calling handler")
}

func TestOrderedInboundEncodingMiddlewareAppply(t *testing.T) {
	var gotOrder []string

	var newMiddleware = func(name string) yarpc.UnaryInboundEncodingMiddleware {
		return yarpc.NewUnaryInboundEncodingMiddleware(name,
			func(ctx context.Context, _ interface{}, h yarpc.UnaryEncodingHandler) (interface{}, error) {
				gotOrder = append(gotOrder, name)
				return h.Handle(ctx, nil)
			})
	}

	mw1 := newMiddleware("mw1")
	mw2 := newMiddleware("mw2")
	mw3 := newMiddleware("mw3")

	handler := yarpc.UnaryEncodingHandlerFunc(func(context.Context, interface{}) (interface{}, error) {
		gotOrder = append(gotOrder, "handler")
		return nil, nil
	})

	handlerWithMW := yarpc.ApplyUnaryInboundEncodingMiddleware(handler, mw1, mw2, mw3)
	_, _ = handlerWithMW.Handle(context.Background(), "nothing")

	wantOrder := []string{"mw1", "mw2", "mw3", "handler"}
	assert.Equal(t, wantOrder, gotOrder, "unexpected middleware ordering")
}
