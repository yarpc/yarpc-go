package outboundinterceptor

import (
	"context"
	"go.uber.org/yarpc/internal/interceptor/interceptortest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/interceptor"
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

func (c *countOutboundMiddleware) CallOneway(ctx context.Context, req *transport.Request, next interceptor.DirectOnewayOutbound) (transport.Ack, error) {
	c.Count++
	return next.DirectCallOneway(ctx, req)
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

	tests := []struct {
		desc string
		mw   interceptor.OnewayOutbound
	}{
		{"flat chain", OnewayChain(before, nil, after)},
		{"nested chain", OnewayChain(before, OnewayChain(after, nil))},
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
			}
			mockOutbound := interceptortest.NewMockDirectOnewayOutbound(mockCtrl)
			mockOutbound.EXPECT().DirectCallOneway(ctx, req).Return(&mockAck{}, nil)

			gotAck, err := tt.mw.CallOneway(ctx, req, mockOutbound)

			assert.NoError(t, err)
			assert.Equal(t, 1, before.Count)
			assert.Equal(t, 1, after.Count)
			assert.NotNil(t, gotAck)
		})
	}
}

func TestStreamChain(t *testing.T) {
	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}

	tests := []struct {
		desc string
		mw   interceptor.StreamOutbound
	}{
		{"flat chain", StreamChain(before, nil, after)},
		{"nested chain", StreamChain(before, StreamChain(after, nil))},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			before.Count, after.Count = 0, 0
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			req := &transport.StreamRequest{
				Meta: &transport.RequestMeta{
					Caller:    "caller",
					Service:   "service",
					Procedure: "procedure",
				},
			}
			mockOutbound := interceptortest.NewMockDirectStreamOutbound(mockCtrl)
			mockOutbound.EXPECT().DirectCallStream(ctx, req).Return(&transport.ClientStream{}, nil)

			gotStream, err := tt.mw.CallStream(ctx, req, mockOutbound)

			assert.NoError(t, err)
			assert.Equal(t, 1, before.Count)
			assert.Equal(t, 1, after.Count)
			assert.NotNil(t, gotStream)
		})
	}
}

//func TestEmptyChains(t *testing.T) {
//	assert.Equal(t, outboundinterceptor.unaryChainExec(DirectUnaryOutbound(nil)}), NewUnaryChain(nil, nil))
//	assert.Equal(t, interceptor.NopOnewayOutbound, OnewayChain())
//	assert.Equal(t, interceptor.NopStreamOutbound, StreamChain())
//}
