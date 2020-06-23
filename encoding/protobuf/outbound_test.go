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

package protobuf_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
	"go.uber.org/yarpc/yarpctest"
)

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

			client := protobuf.NewClient(protobuf.ClientParams{
				ClientConfig: &transport.OutboundConfig{
					Outbounds: transport.Outbounds{
						Unary: out,
					},
				},
				AnyResolver: tt.resolver,
				Options:     []protobuf.ClientOption{protobuf.UseJSON},
			})

			testMessage := &testpb.TestMessage{Value: testValue}

			// convert to an Any so that the marshaller will use the custom resolver
			any, err := ptypes.MarshalAny(testMessage)
			require.NoError(t, err)
			any.TypeUrl = tt.anyURL // update to custom URL

			gotMessageI, err := client.Call(context.Background(), "", any, newReq)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				gotMessage, ok := gotMessageI.(*testpb.TestMessage)
				require.True(t, ok, "unexpected message, got %T", gotMessageI)
				assert.Equal(t, testMessage.Value, gotMessage.Value) // we expect the actual type behind the Any
			}
		})
	}
}
