// Copyright (c) 2022 Uber Technologies, Inc.
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

package v2_test

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb/v2"
	"go.uber.org/yarpc/encoding/protobuf/v2"
	"go.uber.org/yarpc/yarpctest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestOutboundAnyResolver(t *testing.T) {
	const testValue = "foo-bar-baz"
	newReq := func() proto.Message { return &testpb.TestMessage{} }
	customAnyResolver := &anyResolver{NewMessage: &testpb.TestMessage{}}
	tests := []struct {
		name     string
		anyURL   string
		resolver v2.AnyResolver
		wantErr  bool
	}{
		{
			name:   "nothing custom",
			anyURL: "uber.yarpc.encoding.protobuf.TestMessage",
		},
		{
			name:     "custom resolver",
			anyURL:   "uber.yarpc.encoding.protobuf.TestMessage",
			resolver: customAnyResolver,
		},
		{
			name:     "custom resolver, custom URL",
			anyURL:   "foo.bar.baz",
			resolver: customAnyResolver,
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

			client := v2.NewClient(v2.ClientParams{
				ClientConfig: &transport.OutboundConfig{
					Outbounds: transport.Outbounds{
						Unary: out,
					},
				},
				AnyResolver: tt.resolver,
				Options:     []v2.ClientOption{v2.UseJSON},
			})

			testMessage := &testpb.TestMessage{Value: testValue}

			// convert to an Any so that the marshaller will use the custom resolver
			anyMsg, err := anypb.New(testMessage)
			require.NoError(t, err)
			anyMsg.TypeUrl = tt.anyURL // update to custom URL

			gotMessage, err := client.Call(context.Background(), "", anyMsg, newReq)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, proto.Equal(testMessage, gotMessage)) // we expect the actual type behind the Any
			}
		})
	}
}

type anyResolver struct {
	NewMessage proto.Message
}

func (r anyResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	return r.FindMessageByURL(string(message))
}

func (r anyResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	if r.NewMessage == nil {
		return nil, errors.New("test resolver is not initialized")
	}
	return protoimpl.X.MessageTypeOf(r.NewMessage), nil
}

func (r anyResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, nil
}

func (r anyResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, nil
}
