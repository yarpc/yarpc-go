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

package filter

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var retryFilter transport.FilterFunc = func(
	ctx context.Context, req *transport.Request, o transport.Outbound) (*transport.Response, error) {
	res, err := o.Call(ctx, req)
	if err != nil {
		res, err = o.Call(ctx, req)
	}
	return res, err
}

type countFilter struct{ Count int }

func (c *countFilter) Call(
	ctx context.Context, req *transport.Request, o transport.Outbound) (*transport.Response, error) {
	c.Count++
	return o.Call(ctx, req)
}

func TestChain(t *testing.T) {
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

	o := transporttest.NewMockOutbound(mockCtrl)
	o.EXPECT().Call(ctx, req).After(
		o.EXPECT().Call(ctx, req).Return(nil, errors.New("great sadness")),
	).Return(res, nil)

	before := &countFilter{}
	after := &countFilter{}
	gotRes, err := transport.ApplyFilter(
		o, Chain(before, retryFilter, after)).Call(ctx, req)

	assert.NoError(t, err, "expected success")
	assert.Equal(t, 1, before.Count, "expected outer filter to be called once")
	assert.Equal(t, 2, after.Count, "expected inner filter to be called twice")
	assert.Equal(t, res, gotRes, "expected response to match")
}
