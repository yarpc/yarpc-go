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

package yarpctransport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/zap/zapcore"
)

type UnaryHandlerFunc func(context.Context, *yarpc.Request, *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error)
type StreamHandlerFunc func(*yarpc.ServerStream) error

func (f UnaryHandlerFunc) Handle(ctx context.Context, r *yarpc.Request, b *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	return f(ctx, r, b)
}

func (f StreamHandlerFunc) HandleStream(stream *yarpc.ServerStream) error {
	return f(stream)
}

func TestHandlerSpecLogMarshaling(t *testing.T) {
	tests := []struct {
		desc string
		spec yarpc.TransportHandlerSpec
		want map[string]interface{}
	}{
		{
			desc: "unary",
			spec: yarpc.NewUnaryTransportHandlerSpec(UnaryHandlerFunc(func(context.Context, *yarpc.Request, *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
				return nil, nil, nil
			})),
			want: map[string]interface{}{"rpcType": "Unary"},
		},
		{
			desc: "stream",
			spec: yarpc.NewStreamTransportHandlerSpec(StreamHandlerFunc(func(*yarpc.ServerStream) error {
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
