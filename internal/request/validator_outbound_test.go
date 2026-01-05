// Copyright (c) 2026 Uber Technologies, Inc.
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

package request

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/api/x/introspection"
)

func TestCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := newValidTestRequest()

	t.Run("unary", func(t *testing.T) {
		out := transporttest.NewMockUnaryOutbound(ctrl)
		validatorOut := UnaryValidatorOutbound{UnaryOutbound: out}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		out.EXPECT().Call(ctx, req).Return(nil, nil)

		_, err := validatorOut.Call(ctx, req)
		require.NoError(t, err)
	})

	t.Run("oneway", func(t *testing.T) {
		out := transporttest.NewMockOnewayOutbound(ctrl)
		validatorOut := OnewayValidatorOutbound{OnewayOutbound: out}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		out.EXPECT().CallOneway(ctx, req).Return(nil, nil)

		_, err := validatorOut.CallOneway(ctx, req)
		require.NoError(t, err)
	})
}

func TestCallErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	validatorOut := UnaryValidatorOutbound{UnaryOutbound: transporttest.NewMockUnaryOutbound(ctrl)}
	validatorOutOneway := OnewayValidatorOutbound{OnewayOutbound: transporttest.NewMockOnewayOutbound(ctrl)}

	tests := []struct {
		name string
		ctx  context.Context
		req  *transport.Request
	}{
		{
			name: "invalid context",
			ctx:  context.Background(), // no deadline
			req:  newValidTestRequest(),
		},
		{
			name: "invalid request",
			req:  &transport.Request{},
		},
	}

	for _, tt := range tests {
		t.Run("unary: "+tt.name, func(t *testing.T) {
			res, err := validatorOut.Call(tt.ctx, tt.req)
			assert.Nil(t, res)
			assert.Error(t, err, "expected error from invalid request")
		})

		t.Run("oneway: "+tt.name, func(t *testing.T) {
			ack, err := validatorOutOneway.CallOneway(tt.ctx, tt.req)
			assert.Nil(t, ack)
			assert.Error(t, err, "expected error from invalid request")
		})

		// Streaming requests without deadlines are valid, so this this test
		// table does not apply.
	}
}

func TestIntrospect(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("unary", func(t *testing.T) {
		validatorOut := UnaryValidatorOutbound{UnaryOutbound: transporttest.NewMockUnaryOutbound(ctrl)}
		assert.Equal(t, introspection.OutboundStatusNotSupported, validatorOut.Introspect())
	})

	t.Run("oneway", func(t *testing.T) {
		validatorOut := OnewayValidatorOutbound{OnewayOutbound: transporttest.NewMockOnewayOutbound(ctrl)}
		assert.Equal(t, introspection.OutboundStatusNotSupported, validatorOut.Introspect())
	})

	t.Run("stream", func(t *testing.T) {
		validatorOut := StreamValidatorOutbound{StreamOutbound: transporttest.NewMockStreamOutbound(ctrl)}
		assert.Equal(t, introspection.OutboundStatusNotSupported, validatorOut.Introspect())
	})
}

func newValidTestRequest() *transport.Request {
	return &transport.Request{
		Service:   "service",
		Procedure: "procedure",
		Caller:    "caller",
		Encoding:  "encoding",
	}
}

func TestStreamValidate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Service:   "service",
			Procedure: "proc",
			Caller:    "caller",
			Encoding:  "raw",
		},
	}
	stream, err := transport.NewClientStream(transporttest.NewMockStreamCloser(mockCtrl))
	require.NoError(t, err)

	out := transporttest.NewMockStreamOutbound(mockCtrl)
	out.EXPECT().CallStream(ctx, req).Times(1).Return(stream, nil)

	validator := StreamValidatorOutbound{StreamOutbound: out}

	gotStream, gotErr := validator.CallStream(ctx, req)

	assert.NoError(t, gotErr)
	assert.Equal(t, stream, gotStream)
}

func TestStreamValidateError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Service:  "service",
			Caller:   "caller",
			Encoding: "raw",
		},
	}

	out := transporttest.NewMockStreamOutbound(mockCtrl)

	validator := StreamValidatorOutbound{StreamOutbound: out}

	_, gotErr := validator.CallStream(ctx, req)

	assert.Error(t, gotErr)
}
