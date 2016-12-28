package encoding

import (
	"context"
	"sort"
	"testing"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		ShardKey:        "shardKey",
		RoutingKey:      "routingKey",
		RoutingDelegate: "routingDelegate",
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
