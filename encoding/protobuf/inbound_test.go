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
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb"
)

var _ jsonpb.AnyResolver = (*testResolver)(nil)

type testResolver struct {
	NewMessage func() proto.Message
}

func (t *testResolver) Resolve(url string) (proto.Message, error) {
	if t.NewMessage == nil {
		return nil, errors.New("test resolver is not initialized")
	}
	return t.NewMessage(), nil
}

func TestInboundAnyResolver(t *testing.T) {
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
			handler := protobuf.NewUnaryHandler(protobuf.UnaryHandlerParams{
				Handle: func(context.Context, proto.Message) (proto.Message, error) {
					testMessage := &testpb.TestMessage{Value: "foo-bar-baz"}

					any, err := ptypes.MarshalAny(testMessage)
					assert.NoError(t, err)
					any.TypeUrl = tt.anyURL // update to custom URL
					return any, nil
				},
				NewRequest:  newReq,
				AnyResolver: tt.resolver,
			})

			req := &transport.Request{
				Encoding: protobuf.JSONEncoding,
				Body:     bytes.NewReader(nil),
			}

			var resw transporttest.FakeResponseWriter
			err := handler.Handle(context.Background(), req, &resw)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}