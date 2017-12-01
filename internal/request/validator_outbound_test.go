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
	"go.uber.org/yarpc/internal/introspection"
)

func TestCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := newValidTestRequest()

	t.Run("unary", func(t *testing.T) {
		out := transporttest.NewMockUnaryOutbound(ctrl)
		validatorOut := UnaryValidatorOutbound{out}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		out.EXPECT().Call(ctx, req).Return(nil, nil)

		_, err := validatorOut.Call(ctx, req)
		require.NoError(t, err)
	})

	t.Run("oneway", func(t *testing.T) {
		out := transporttest.NewMockOnewayOutbound(ctrl)
		validatorOut := OnewayValidatorOutbound{out}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		out.EXPECT().CallOneway(ctx, req).Return(nil, nil)

		_, err := validatorOut.CallOneway(ctx, req)
		require.NoError(t, err)
	})
}

func TestCallErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	validatorOut := UnaryValidatorOutbound{transporttest.NewMockUnaryOutbound(ctrl)}
	validatorOutOneway := OnewayValidatorOutbound{transporttest.NewMockOnewayOutbound(ctrl)}

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
	}
}

func TestIntrospect(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("unary", func(t *testing.T) {
		validatorOut := UnaryValidatorOutbound{transporttest.NewMockUnaryOutbound(ctrl)}
		assert.Equal(t, introspection.OutboundStatusNotSupported, validatorOut.Introspect())
	})

	t.Run("unary", func(t *testing.T) {
		validatorOut := OnewayValidatorOutbound{transporttest.NewMockOnewayOutbound(ctrl)}
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
