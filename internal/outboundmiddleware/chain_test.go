// Copyright (c) 2019 Uber Technologies, Inc.
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

package outboundmiddleware

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
	"go.uber.org/yarpc/api/middleware/middlewaretest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/introspection"
	"go.uber.org/yarpc/internal/testtime"
)

type countOutboundMiddleware struct{ Count int }

func (c *countOutboundMiddleware) Call(
	ctx context.Context, req *transport.Request, o transport.UnaryOutbound) (*transport.Response, error) {
	c.Count++
	return o.Call(ctx, req)
}

func (c *countOutboundMiddleware) CallOneway(ctx context.Context, req *transport.Request, o transport.OnewayOutbound) (transport.Ack, error) {
	c.Count++
	return o.CallOneway(ctx, req)
}

func (c *countOutboundMiddleware) CallStream(ctx context.Context, req *transport.StreamRequest, o transport.StreamOutbound) (*transport.ClientStream, error) {
	c.Count++
	return o.CallStream(ctx, req)
}

var retryUnaryOutbound middleware.UnaryOutboundFunc = func(
	ctx context.Context, req *transport.Request, o transport.UnaryOutbound) (*transport.Response, error) {
	res, err := o.Call(ctx, req)
	if err != nil {
		res, err = o.Call(ctx, req)
	}
	return res, err
}

func TestUnaryChain(t *testing.T) {
	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}

	tests := []struct {
		desc string
		mw   middleware.UnaryOutbound
	}{
		{"flat chain", UnaryChain(before, retryUnaryOutbound, nil, after)},
		{"nested chain", UnaryChain(before, UnaryChain(retryUnaryOutbound, after, nil))},
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

var retryOnewayOutbound middleware.OnewayOutboundFunc = func(
	ctx context.Context, req *transport.Request, o transport.OnewayOutbound) (transport.Ack, error) {
	res, err := o.CallOneway(ctx, req)
	if err != nil {
		res, err = o.CallOneway(ctx, req)
	}
	return res, err
}

func TestOnewayChain(t *testing.T) {
	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}

	tests := []struct {
		desc string
		mw   middleware.OnewayOutbound
	}{
		{"flat chain", OnewayChain(before, retryOnewayOutbound, nil, after)},
		{"flat chain", OnewayChain(before, OnewayChain(retryOnewayOutbound, after, nil))},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			var res transport.Ack
			req := &transport.Request{
				Caller:    "somecaller",
				Service:   "someservice",
				Encoding:  transport.Encoding("raw"),
				Procedure: "hello",
				Body:      bytes.NewReader([]byte{1, 2, 3}),
			}
			o := transporttest.NewMockOnewayOutbound(mockCtrl)
			before.Count, after.Count = 0, 0
			o.EXPECT().CallOneway(ctx, req).After(
				o.EXPECT().CallOneway(ctx, req).Return(nil, errors.New("great sadness")),
			).Return(res, nil)

			gotRes, err := middleware.ApplyOnewayOutbound(o, tt.mw).CallOneway(ctx, req)

			assert.NoError(t, err, "expected success")
			assert.Equal(t, 1, before.Count, "expected outer middleware to be called once")
			assert.Equal(t, 2, after.Count, "expected inner middleware to be called twice")
			assert.Equal(t, res, gotRes, "expected response to match")
		})
	}
}

func TestEmptyChain(t *testing.T) {
	errMsg := "expected nop Outbound"

	t.Run("unary", func(t *testing.T) {
		require.Equal(t, middleware.NopUnaryOutbound, UnaryChain(), errMsg)
	})

	t.Run("oneway", func(t *testing.T) {
		require.Equal(t, middleware.NopOnewayOutbound, OnewayChain(), errMsg)
	})
}

func TestSingleOutboundChain(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("unary", func(t *testing.T) {
		out := middlewaretest.NewMockUnaryOutbound(ctrl)
		require.Equal(t, out, UnaryChain(out))
	})

	t.Run("oneway", func(t *testing.T) {
		out := middlewaretest.NewMockOnewayOutbound(ctrl)
		require.Equal(t, out, OnewayChain(out))
	})
}

func TestUnaryChainExec(t *testing.T) {
	ctrl := gomock.NewController(t)
	out := transporttest.NewMockUnaryOutbound(ctrl)

	chain := &unaryChainExec{Final: out}

	// start
	out.EXPECT().Start().Return(nil)
	assert.NoError(t, chain.Start(), "could not start outbound")

	// transports
	out.EXPECT().Transports()
	chain.Transports()

	// is running
	out.EXPECT().IsRunning().Return(true)
	assert.True(t, chain.IsRunning(), "expected outbound to be running")

	// stop
	out.EXPECT().Stop().Return(nil)
	assert.NoError(t, chain.Stop(), "unexpected error stopping outbound")
}

func TestOnewayChainExec(t *testing.T) {
	ctrl := gomock.NewController(t)
	out := transporttest.NewMockOnewayOutbound(ctrl)

	chain := &onewayChainExec{Final: out}

	// start
	out.EXPECT().Start().Return(nil)
	assert.NoError(t, chain.Start(), "could not start outbound")

	// transports
	out.EXPECT().Transports()
	chain.Transports()

	// is running
	out.EXPECT().IsRunning().Return(true)
	assert.True(t, chain.IsRunning(), "expected outbound to be running")

	// stop
	out.EXPECT().Stop().Return(nil)
	assert.NoError(t, chain.Stop(), "unexpected error stopping outbound")
}

func TestIntrospect(t *testing.T) {
	ctrl := gomock.NewController(t)
	expectStatus := introspection.OutboundStatusNotSupported
	errMsg := "expected not supported status"

	t.Run("unary", func(t *testing.T) {
		out := transporttest.NewMockUnaryOutbound(ctrl)
		chain := &unaryChainExec{Final: out}
		assert.Equal(t, expectStatus, chain.Introspect(), errMsg)
	})

	t.Run("oneway", func(t *testing.T) {
		out := transporttest.NewMockOnewayOutbound(ctrl)
		chain := &onewayChainExec{Final: out}
		assert.Equal(t, expectStatus, chain.Introspect(), errMsg)
	})
}

var retryStreamOutbound middleware.StreamOutboundFunc = func(
	ctx context.Context, req *transport.StreamRequest, o transport.StreamOutbound) (*transport.ClientStream, error) {
	res, err := o.CallStream(ctx, req)
	if err != nil {
		res, err = o.CallStream(ctx, req)
	}
	return res, err
}

func TestStreamChain(t *testing.T) {
	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}

	tests := []struct {
		desc string
		mw   middleware.StreamOutbound
	}{
		{"flat chain", StreamChain(before, retryStreamOutbound, nil, after)},
		{"nested chain", StreamChain(before, StreamChain(retryStreamOutbound, after, nil))},
		{"single chain", StreamChain(StreamChain(before), retryStreamOutbound, StreamChain(after), StreamChain())},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			var res *transport.ClientStream
			req := &transport.StreamRequest{
				Meta: &transport.RequestMeta{
					Caller:    "somecaller",
					Service:   "someservice",
					Encoding:  transport.Encoding("raw"),
					Procedure: "hello",
				},
			}
			o := transporttest.NewMockStreamOutbound(mockCtrl)

			before.Count, after.Count = 0, 0
			o.EXPECT().CallStream(ctx, req).After(
				o.EXPECT().CallStream(ctx, req).Return(nil, errors.New("great sadness")),
			).Return(res, nil)

			mw := middleware.ApplyStreamOutbound(o, tt.mw)
			gotRes, err := mw.CallStream(ctx, req)

			assert.NoError(t, err, "expected success")
			assert.Equal(t, 1, before.Count, "expected outer middleware to be called once")
			assert.Equal(t, 2, after.Count, "expected inner middleware to be called twice")
			assert.Equal(t, res, gotRes, "expected response to match")
		})
	}
}

func TestStreamChainExecFuncs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	o := transporttest.NewMockStreamOutbound(mockCtrl)
	o.EXPECT().Stop()
	o.EXPECT().Start()
	o.EXPECT().Transports()
	o.EXPECT().IsRunning().Return(true)

	mw := streamChainExec{Final: o}

	assert.Nil(t, mw.Start())
	assert.True(t, mw.IsRunning())
	assert.Nil(t, mw.Stop())
	assert.Len(t, mw.Transports(), 0)
}
