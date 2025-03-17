// Copyright (c) 2025 Uber Technologies, Inc.
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

package outboundinterceptor

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
	"go.uber.org/yarpc/internal/interceptor/interceptortest"
	"go.uber.org/yarpc/internal/testtime"
)

type countOutboundMiddleware struct {
	Count int
}

func (c *countOutboundMiddleware) Call(ctx context.Context, req *transport.Request, next interceptor.UnaryOutboundChain) (*transport.Response, error) {
	c.Count++
	return next.Next(ctx, req)
}

type mockAck struct{}

func (m *mockAck) String() string {
	return "mockAck"
}

func (c *countOutboundMiddleware) CallOneway(ctx context.Context, req *transport.Request, next interceptor.OnewayOutboundChain) (transport.Ack, error) {
	c.Count++
	return next.Next(ctx, req)
}

func (c *countOutboundMiddleware) CallStream(ctx context.Context, req *transport.StreamRequest, next interceptor.DirectStreamOutbound) (*transport.ClientStream, error) {
	c.Count++
	return next.DirectCallStream(ctx, req)
}

func TestUnaryChain(t *testing.T) {
	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}
	nopOutbound := interceptortest.NewMockDirectUnaryOutbound(gomock.NewController(t))

	nopOutbound.EXPECT().DirectCall(gomock.Any(), gomock.Any()).AnyTimes().Return(&transport.Response{}, nil)

	tests := []struct {
		desc string
		mw   interceptor.UnaryOutbound
	}{
		{
			desc: "flat chain",
			mw: unaryOutboundAdapter{
				chain: NewUnaryChain(nopOutbound, []interceptor.UnaryOutbound{before, after}),
			},
		},
		{
			desc: "nested chain",
			mw: unaryOutboundAdapter{
				chain: NewUnaryChain(nopOutbound, []interceptor.UnaryOutbound{
					before,
					unaryOutboundAdapter{
						chain: NewUnaryChain(nopOutbound, []interceptor.UnaryOutbound{after}),
					},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			before.Count, after.Count = 0, 0
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{}, // Ensure exact match
			}
			res := &transport.Response{}

			mockOutbound := interceptortest.NewMockUnaryOutboundChain(mockCtrl)

			gotRes, err := tt.mw.Call(ctx, req, mockOutbound)

			assert.NoError(t, err)
			assert.Equal(t, 1, before.Count)
			assert.Equal(t, 1, after.Count)
			assert.Equal(t, res, gotRes)
		})
	}
}

type unaryOutboundAdapter struct {
	chain interceptor.UnaryOutboundChain
}

func (u unaryOutboundAdapter) Call(ctx context.Context, req *transport.Request, out interceptor.UnaryOutboundChain) (*transport.Response, error) {
	return u.chain.Next(ctx, req)
}

func TestOnewayChain(t *testing.T) {
	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}
	nopOutbound := interceptortest.NewMockDirectOnewayOutbound(gomock.NewController(t))

	nopOutbound.EXPECT().DirectCallOneway(gomock.Any(), gomock.Any()).AnyTimes().Return(&mockAck{}, nil)

	tests := []struct {
		desc string
		mw   interceptor.OnewayOutbound
	}{
		{
			desc: "flat chain",
			mw: onewayOutboundAdapter{
				chain: NewOnewayChain(nopOutbound, []interceptor.OnewayOutbound{before, after}),
			},
		},
		{
			desc: "nested chain",
			mw: onewayOutboundAdapter{
				chain: NewOnewayChain(nopOutbound, []interceptor.OnewayOutbound{
					before,
					onewayOutboundAdapter{
						chain: NewOnewayChain(nopOutbound, []interceptor.OnewayOutbound{after}),
					},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			before.Count, after.Count = 0, 0
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			mockOutbound := interceptortest.NewMockOnewayOutboundChain(mockCtrl)

			gotAck, err := tt.mw.CallOneway(ctx, req, mockOutbound)

			assert.NoError(t, err)
			assert.Equal(t, 1, before.Count)
			assert.Equal(t, 1, after.Count)
			assert.NotNil(t, gotAck)
		})
	}
}

type onewayOutboundAdapter struct {
	chain interceptor.OnewayOutboundChain
}

func (u onewayOutboundAdapter) CallOneway(ctx context.Context, req *transport.Request, out interceptor.OnewayOutboundChain) (transport.Ack, error) {
	return u.chain.Next(ctx, req)
}

func TestStreamChain(t *testing.T) {
	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}
	nopOutbound := interceptortest.NewMockDirectUnaryOutbound(gomock.NewController(t))

	nopOutbound.EXPECT().DirectCall(gomock.Any(), gomock.Any()).AnyTimes().Return(&transport.Response{}, nil)

	tests := []struct {
		desc string
		mw   interceptor.UnaryOutbound
	}{
		{
			desc: "flat chain",
			mw: unaryOutboundAdapter{
				chain: NewUnaryChain(nopOutbound, []interceptor.UnaryOutbound{before, after}),
			},
		},
		{
			desc: "nested chain",
			mw: unaryOutboundAdapter{
				chain: NewUnaryChain(nopOutbound, []interceptor.UnaryOutbound{
					before,
					unaryOutboundAdapter{
						chain: NewUnaryChain(nopOutbound, []interceptor.UnaryOutbound{after}),
					},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			before.Count, after.Count = 0, 0
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{}, // Ensure exact match
			}
			res := &transport.Response{}

			mockOutbound := interceptortest.NewMockUnaryOutboundChain(mockCtrl)

			gotRes, err := tt.mw.Call(ctx, req, mockOutbound)

			assert.NoError(t, err)
			assert.Equal(t, 1, before.Count)
			assert.Equal(t, 1, after.Count)
			assert.Equal(t, res, gotRes)
		})
	}
}
