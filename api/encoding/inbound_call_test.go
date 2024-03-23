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

package encoding

import (
	"context"
	"net/netip"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
)

func TestInboundCallReadFromRequest(t *testing.T) {
	ctx, inboundCall := NewInboundCall(context.Background())
	err := inboundCall.ReadFromRequest(&transport.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  transport.Encoding("raw"),
		Procedure: "hello",
		Headers: transport.HeadersFromMap(map[string]string{
			"hello":   "World",
			"Foo":     "bar",
			"success": "true",
		}),
		ShardKey:           "shardKey",
		RoutingKey:         "routingKey",
		RoutingDelegate:    "routingDelegate",
		CallerProcedure:    "callerProcedure",
		CallerPeerAddrPort: netip.MustParseAddrPort("1.2.3.4:1234"),
	})
	require.NoError(t, err)

	call := CallFromContext(ctx)
	assert.Equal(t, "caller", call.Caller())
	assert.Equal(t, "service", call.Service())
	assert.Equal(t, "raw", string(call.Encoding()))
	assert.Equal(t, "hello", call.Procedure())
	assert.Equal(t, "shardKey", call.ShardKey())
	assert.Equal(t, "routingKey", call.RoutingKey())
	assert.Equal(t, "routingDelegate", call.RoutingDelegate())
	assert.Equal(t, "callerProcedure", call.CallerProcedure())
	assert.Zero(t, netip.MustParseAddrPort("1.2.3.4:1234").Compare(call.CallerPeerAddrPort()))
	assert.Equal(t, "World", call.Header("Hello"))
	assert.Equal(t, "bar", call.Header("FOO"))
	assert.Equal(t, "true", call.Header("success"))
	assert.Equal(t, "", call.Header("does-not-exist"))

	headerNames := call.HeaderNames()
	sort.Strings(headerNames)
	assert.Equal(t, []string{"foo", "hello", "success"}, headerNames)
}

func TestInboundCallWriteToResponse(t *testing.T) {
	tests := []struct {
		desc        string
		sendHeaders map[string]string
		wantHeaders transport.Headers
	}{
		{
			desc: "no headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx, inboundCall := NewInboundCall(context.Background())
			call := CallFromContext(ctx)
			for k, v := range tt.sendHeaders {
				call.WriteResponseHeader(k, v)
			}

			var resw transporttest.FakeResponseWriter
			assert.NoError(t, inboundCall.WriteToResponse(&resw))
			assert.Equal(t, tt.wantHeaders, resw.Headers)
		})
	}
}
