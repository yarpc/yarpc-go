// Copyright (c) 2019 Uber Technologies, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
)

func TestOutboundCallWriteToRequestAndRequestMeta(t *testing.T) {
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
		t.Run(tt.desc+" regular", func(t *testing.T) {
			call := NewOutboundCall(tt.giveOptions...)

			request := tt.giveRequest
			requestMeta := tt.giveRequest.ToRequestMeta()

			_, err := call.WriteToRequest(context.Background(), &request)
			if assert.NoError(t, err, tt.desc) {
				assert.Equal(t, tt.wantRequest, request, tt.desc)
			}

			_, err = call.WriteToRequestMeta(context.Background(), requestMeta)
			if assert.NoError(t, err, tt.desc) {
				assert.Equal(t, tt.wantRequest.ToRequestMeta(), requestMeta, tt.desc)
			}
		})

		t.Run(tt.desc+" streaming", func(t *testing.T) {
			call, err := NewStreamOutboundCall(tt.giveOptions...)
			require.NoError(t, err)

			request := tt.giveRequest
			requestMeta := tt.giveRequest.ToRequestMeta()

			_, err = call.WriteToRequest(context.Background(), &request)
			if assert.NoError(t, err, tt.desc) {
				assert.Equal(t, tt.wantRequest, request, tt.desc)
			}

			_, err = call.WriteToRequestMeta(context.Background(), requestMeta)
			if assert.NoError(t, err, tt.desc) {
				assert.Equal(t, tt.wantRequest.ToRequestMeta(), requestMeta, tt.desc)
			}
		})
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

func TestStreamOutboundCallCannotReadFromResponse(t *testing.T) {
	var headers map[string]string
	call, err := NewStreamOutboundCall(ResponseHeaders(&headers))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code:invalid-argument")
	assert.Contains(t, err.Error(), "response headers are not supported for streams")
	assert.Nil(t, call)
}
