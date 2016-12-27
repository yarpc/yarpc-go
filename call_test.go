package yarpc

import (
	"context"
	"sort"
	"testing"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboundCallWriteToRequest(t *testing.T) {
	tests := []struct {
		desc        string
		giveOptions []CallOption
		giveRequest transport.Request
		wantRequest transport.Request
	}{
		{
			desc:        "no options",
			giveOptions: []CallOption{},
			giveRequest: transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  transport.Encoding("raw"),
				Procedure: "hello",
			},
			wantRequest: transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  transport.Encoding("raw"),
				Procedure: "hello",
			},
		},
		{
			desc: "headers",
			giveOptions: []CallOption{
				WithHeader("foo", "bar"),
				WithHeader("baz", "qux"),
			},
			giveRequest: transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  transport.Encoding("raw"),
				Procedure: "hello",
			},
			wantRequest: transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  transport.Encoding("raw"),
				Procedure: "hello",
				Headers:   transport.HeadersFromMap(map[string]string{"foo": "bar", "baz": "qux"}),
			},
		},
		{
			desc: "headers with duplicates",
			giveOptions: []CallOption{
				WithHeader("foo", "bar"),
				WithHeader("baz", "qux"),
				WithHeader("foo", "qux"),
			},
			wantRequest: transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{
					"foo": "qux",
					"baz": "qux",
				}),
			},
		},
		{
			desc: "shard key",
			giveOptions: []CallOption{
				WithHeader("foo", "bar"),
				WithShardKey("derp"),
			},
			wantRequest: transport.Request{
				Headers:  transport.NewHeaders().With("foo", "bar"),
				ShardKey: "derp",
			},
		},
		{
			desc: "routing key",
			giveOptions: []CallOption{
				WithShardKey("derp"),
				WithRoutingKey("hello"),
			},
			wantRequest: transport.Request{
				ShardKey:   "derp",
				RoutingKey: "hello",
			},
		},
		{
			desc: "routing delegate",
			giveOptions: []CallOption{
				WithRoutingKey("hello"),
				WithRoutingDelegate("zzz"),
			},
			wantRequest: transport.Request{
				RoutingKey:      "hello",
				RoutingDelegate: "zzz",
			},
		},
	}

	for _, tt := range tests {
		call := NewOutboundCall(tt.giveOptions...)

		request := tt.giveRequest
		_, err := call.WriteToRequest(context.Background(), &request)
		if assert.NoError(t, err, tt.desc) {
			assert.Equal(t, tt.wantRequest, request, tt.desc)
		}
	}
}

func TestOutboundCallReadFromResponse(t *testing.T) {
	var headers map[string]string
	call := NewOutboundCall(ResponseHeaders(&headers))
	_, err := call.ReadFromResponse(context.Background(), &transport.Response{
		Headers: transport.HeadersFromMap(map[string]string{
			"hello":   "World",
			"Foo":     "bar",
			"success": "true",
		}),
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"hello":   "World",
		"foo":     "bar",
		"success": "true",
	}, headers)
}

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

func TestNilCall(t *testing.T) {
	call := CallFromContext(context.Background())
	require.Nil(t, call)

	assert.Equal(t, "", call.Caller())
	assert.Equal(t, "", call.Service())
	assert.Equal(t, "", string(call.Encoding()))
	assert.Equal(t, "", call.Procedure())
	assert.Equal(t, "", call.ShardKey())
	assert.Equal(t, "", call.RoutingKey())
	assert.Equal(t, "", call.RoutingDelegate())
	assert.Equal(t, "", call.Header("foo"))
	assert.Empty(t, call.HeaderNames())

	assert.Error(t, call.WriteResponseHeader("foo", "bar"))
}
