// Copyright (c) 2024 Uber Technologies, Inc.
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

package transport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

type UnaryHandlerFunc func(context.Context, *Request, ResponseWriter) error
type OnewayHandlerFunc func(context.Context, *Request) error
type StreamHandlerFunc func(*ServerStream) error

func (f UnaryHandlerFunc) Handle(ctx context.Context, r *Request, w ResponseWriter) error {
	return f(ctx, r, w)
}

func (f OnewayHandlerFunc) HandleOneway(ctx context.Context, r *Request) error {
	return f(ctx, r)
}

func (f StreamHandlerFunc) HandleStream(stream *ServerStream) error {
	return f(stream)
}

func TestHandlerSpecLogMarshaling(t *testing.T) {
	tests := []struct {
		desc string
		spec HandlerSpec
		want map[string]interface{}
	}{
		{
			desc: "unary",
			spec: NewUnaryHandlerSpec(UnaryHandlerFunc(func(context.Context, *Request, ResponseWriter) error {
				return nil
			})),
			want: map[string]interface{}{"rpcType": "Unary"},
		},
		{
			desc: "oneway",
			spec: NewOnewayHandlerSpec(OnewayHandlerFunc(func(context.Context, *Request) error {
				return nil
			})),
			want: map[string]interface{}{"rpcType": "Oneway"},
		},
		{
			desc: "stream",
			spec: NewStreamHandlerSpec(StreamHandlerFunc(func(*ServerStream) error {
				return nil
			})),
			want: map[string]interface{}{"rpcType": "Streaming"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			enc := zapcore.NewMapObjectEncoder()
			assert.NoError(t, tt.spec.MarshalLogObject(enc), "Unexpected error marshaling spec.")
			assert.Equal(t, tt.want, enc.Fields, "Unexpected output from marshaling spec.")
		})
	}
}
