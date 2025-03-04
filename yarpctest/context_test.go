// Copyright (c) 2025 Uber Technologies, Inc.
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

package yarpctest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpctest"
)

func TestContextWithCallNil(t *testing.T) {
	ctx := yarpctest.ContextWithCall(context.Background(), nil)
	assert.Nil(t, yarpc.CallFromContext(ctx))
}

func TestContextWithCall(t *testing.T) {
	tests := []struct {
		desc           string
		resHeaders     map[string]string
		wantResHeaders map[string]string
	}{
		{
			desc:           "with response headers",
			resHeaders:     make(map[string]string),
			wantResHeaders: map[string]string{"baz": "qux"},
		},
		{
			desc:           "without response headers",
			resHeaders:     nil,
			wantResHeaders: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testCall := &yarpctest.Call{
				Caller:          "caller",
				Service:         "service",
				Transport:       "transport",
				Procedure:       "procedure",
				Encoding:        "encoding",
				Headers:         map[string]string{"foo": "bar"},
				ShardKey:        "shardkey",
				RoutingKey:      "routingkey",
				RoutingDelegate: "routingdelegate",
				ResponseHeaders: tt.resHeaders,
				CallerProcedure: "callerProcedure",
			}
			ctx := yarpctest.ContextWithCall(context.Background(), testCall)
			call := yarpc.CallFromContext(ctx)
			clone := call.Clone()
			assert.NotNil(t, clone)

			assert.Equal(t, "caller", call.Caller())
			assert.Equal(t, "service", call.Service())
			assert.Equal(t, "transport", call.Transport())
			assert.Equal(t, "procedure", call.Procedure())
			assert.Equal(t, transport.Encoding("encoding"), call.Encoding())
			assert.Equal(t, []string{"foo"}, call.HeaderNames())
			assert.Equal(t, "bar", call.Header("foo"))
			assert.Equal(t, "shardkey", call.ShardKey())
			assert.Equal(t, "routingkey", call.RoutingKey())
			assert.Equal(t, "routingdelegate", call.RoutingDelegate())
			assert.Equal(t, "callerProcedure", call.CallerProcedure())

			assert.Equal(t, "caller", clone.Caller())
			assert.Equal(t, "service", clone.Service())
			assert.Equal(t, "transport", clone.Transport())
			assert.Equal(t, "procedure", clone.Procedure())
			assert.Equal(t, transport.Encoding("encoding"), clone.Encoding())
			assert.Equal(t, []string{"foo"}, clone.HeaderNames())
			assert.Equal(t, "bar", clone.Header("foo"))
			assert.Equal(t, "shardkey", clone.ShardKey())
			assert.Equal(t, "routingkey", clone.RoutingKey())
			assert.Equal(t, "routingdelegate", clone.RoutingDelegate())
			assert.Equal(t, "callerProcedure", clone.CallerProcedure())

			assert.NoError(t, call.WriteResponseHeader("baz", "qux"))
			assert.Equal(t, tt.wantResHeaders, testCall.ResponseHeaders)

			assert.NoError(t, clone.WriteResponseHeader("baz", "qux"))
			// no way to get to response headers of the clone without breaking encapsulation
		})
	}

}
