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

package outboundmiddleware

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

var retryUnaryOutbound middleware.UnaryOutboundFunc = func(
	ctx context.Context, req *transport.Request, o transport.UnaryOutbound) (*transport.Response, error) {
	res, err := o.Call(ctx, req)
	if err != nil {
		res, err = o.Call(ctx, req)
	}
	return res, err
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
	res := &transport.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte{4, 5, 6})),
	}

	o := transporttest.NewMockUnaryOutbound(mockCtrl)
	o.EXPECT().Call(ctx, req).After(
		o.EXPECT().Call(ctx, req).Return(nil, errors.New("great sadness")),
	).Return(res, nil)

	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}
	gotRes, err := middleware.ApplyUnaryOutbound(
		o, UnaryChain(before, retryUnaryOutbound, after)).Call(ctx, req)

	assert.NoError(t, err, "expected success")
	assert.Equal(t, 1, before.Count, "expected outer middleware to be called once")
	assert.Equal(t, 2, after.Count, "expected inner middleware to be called twice")
	assert.Equal(t, res, gotRes, "expected response to match")
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
	var res transport.Ack

	o := transporttest.NewMockOnewayOutbound(mockCtrl)
	o.EXPECT().CallOneway(ctx, req).After(
		o.EXPECT().CallOneway(ctx, req).Return(nil, errors.New("great sadness")),
	).Return(res, nil)

	before := &countOutboundMiddleware{}
	after := &countOutboundMiddleware{}
	gotRes, err := middleware.ApplyOnewayOutbound(
		o, OnewayChain(before, retryOnewayOutbound, after)).CallOneway(ctx, req)

	assert.NoError(t, err, "expected success")
	assert.Equal(t, 1, before.Count, "expected outer middleware to be called once")
	assert.Equal(t, 2, after.Count, "expected inner middleware to be called twice")
	assert.Equal(t, res, gotRes, "expected response to match")
}
