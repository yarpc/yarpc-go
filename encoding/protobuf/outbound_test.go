// Copyright (c) 2020 Uber Technologies, Inc.
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
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/yarpc/yarpctest"
)

func TestInvalidOutboundEncoding(t *testing.T) {
	client := newClient("foo", &transport.OutboundConfig{CallerName: "foo", Outbounds: transport.Outbounds{ServiceName: "bar"}}, nil /*AnyResolver*/)
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
		newClient("test", cc, nil /*AnyResolver*/)
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
		newClient("test", cc, nil /*AnyResolver*/)
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

var _ jsonpb.AnyResolver = (*testResolver)(nil)

func TestOutboundAnyResolver(t *testing.T) {
	const testValue = "foo-bar-baz"
	newReq := func() proto.Message { return &testpb.TestMessage{} }
	customResolver := &testResolver{NewMessage: newReq}

	tests := []struct {
		name     string
		anyURL   string
		resolver jsonpb.AnyResolver
		wantErr  bool
	}{
		{
			name:   "nothing custom",
			anyURL: "uber.yarpc.encoding.protobuf.TestMessage",
		},
		{
			name:     "custom resolver",
			anyURL:   "uber.yarpc.encoding.protobuf.TestMessage",
			resolver: customResolver,
		},
		{
			name:     "custom resolver, custom URL",
			anyURL:   "foo.bar.baz",
			resolver: customResolver,
		},
		{
			name:    "custom URL, no resolver",
			anyURL:  "foo.bar.baz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans := yarpctest.NewFakeTransport()
			// outbound that echos the body back
			out := trans.NewOutbound(nil, yarpctest.OutboundCallOverride(
				yarpctest.OutboundCallable(func(ctx context.Context, req *transport.Request) (*transport.Response, error) {
					return &transport.Response{Body: ioutil.NopCloser(req.Body)}, nil
				}),
			))

			client := NewClient(ClientParams{
				ClientConfig: &transport.OutboundConfig{
					Outbounds: transport.Outbounds{
						Unary: out,
					},
				},
				AnyResolver: tt.resolver,
				Options:     []ClientOption{UseJSON},
			})

			testMessage := &testpb.TestMessage{Value: testValue}

			// convert to an Any so that the marshaller will use the custom resolver
			any, err := types.MarshalAny(testMessage)
			require.NoError(t, err)
			any.TypeUrl = tt.anyURL // update to custom URL

			gotMessage, err := client.Call(context.Background(), "", any, newReq)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testMessage, gotMessage) // we expect the actual type behind the Any
			}
		})
	}
}
