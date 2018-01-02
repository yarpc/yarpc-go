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

package protobuf

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestInvalidOutboundEncoding(t *testing.T) {
	client := newClient("foo", &transport.OutboundConfig{CallerName: "foo", Outbounds: transport.Outbounds{ServiceName: "bar"}})
	_, _, _, _, err := client.buildTransportRequest(context.Background(), "hello", nil, nil)
	assert.NoError(t, err)
	client.encoding = "bat"
	_, _, _, _, err = client.buildTransportRequest(context.Background(), "hello", nil, nil)
	assert.Equal(t, yarpcerrors.CodeInternal, yarpcerrors.FromError(err).Code())
}

func TestNonOutboundConfigWithUnaryClient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cc := transporttest.NewMockClientConfig(mockCtrl)
	cc.EXPECT().Caller().Return("caller")
	cc.EXPECT().Service().Return("service")
	cc.EXPECT().GetUnaryOutbound().Return(transporttest.NewMockUnaryOutbound(mockCtrl))

	assert.NotPanics(t, func() {
		newClient("test", cc)
	})
}

func TestNonOutboundConfigClient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cc := transporttest.NewMockClientConfig(mockCtrl)
	cc.EXPECT().Caller().Return("caller")
	cc.EXPECT().Service().Return("service")
	cc.EXPECT().GetUnaryOutbound().Do(func() { panic("bad times") })

	assert.Panics(t, func() {
		newClient("test", cc)
	})
}

func TestInvalidStreamClientEncoding(t *testing.T) {
	client := &client{
		serviceName: "test",
		outboundConfig: &transport.OutboundConfig{
			Outbounds: transport.Outbounds{},
		},
		encoding: transport.Encoding("raw"),
	}

	_, err := client.CallStream(context.Background(), "somemethod")

	assert.Contains(t, err.Error(), "code:internal")
	assert.Contains(t, err.Error(), "can only use encodings")
}

func TestNoStreamOutbound(t *testing.T) {
	client := &client{
		serviceName: "test",
		outboundConfig: &transport.OutboundConfig{
			Outbounds: transport.Outbounds{},
		},
		encoding: Encoding,
	}

	_, err := client.CallStream(context.Background(), "somemethod")

	assert.Contains(t, err.Error(), "code:internal")
	assert.Contains(t, err.Error(), "no stream outbounds for OutboundConfig")
}

func TestNoResponseHeaders(t *testing.T) {
	client := &client{
		serviceName: "test",
		outboundConfig: &transport.OutboundConfig{
			Outbounds: transport.Outbounds{},
		},
		encoding: Encoding,
	}

	headers := make(map[string]string)

	_, err := client.CallStream(context.Background(), "somemethod", yarpc.ResponseHeaders(&headers))

	assert.Contains(t, err.Error(), "code:invalid-argument")
	assert.Contains(t, err.Error(), "response headers are not supported for streams")
}
