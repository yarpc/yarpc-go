package encoding

import (
	"context"
	"testing"

	"go.uber.org/yarpc/api/transport"

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
