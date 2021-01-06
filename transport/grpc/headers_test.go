// Copyright (c) 2021 Uber Technologies, Inc.
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

package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc/metadata"
)

func TestMetadataToTransportRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Name             string
		MD               metadata.MD
		TransportRequest *transport.Request
		Error            error
	}{
		{
			Name: "Basic",
			MD: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				ShardKeyHeader, "example-shard-key",
				RoutingKeyHeader, "example-routing-key",
				RoutingDelegateHeader, "example-routing-delegate",
				EncodingHeader, "example-encoding",
				"foo", "bar",
				"baz", "bat",
			),
			TransportRequest: &transport.Request{
				Caller:          "example-caller",
				Service:         "example-service",
				ShardKey:        "example-shard-key",
				RoutingKey:      "example-routing-key",
				RoutingDelegate: "example-routing-delegate",
				Encoding:        "example-encoding",
				Headers: transport.HeadersFromMap(map[string]string{
					"foo": "bar",
					"baz": "bat",
				}),
			},
		},
		{
			Name: "Content-type",
			MD: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				ShardKeyHeader, "example-shard-key",
				RoutingKeyHeader, "example-routing-key",
				RoutingDelegateHeader, "example-routing-delegate",
				contentTypeHeader, "application/grpc+example-encoding",
				"foo", "bar",
				"baz", "bat",
			),
			TransportRequest: &transport.Request{
				Caller:          "example-caller",
				Service:         "example-service",
				ShardKey:        "example-shard-key",
				RoutingKey:      "example-routing-key",
				RoutingDelegate: "example-routing-delegate",
				Encoding:        "example-encoding",
				Headers: transport.HeadersFromMap(map[string]string{
					"foo": "bar",
					"baz": "bat",
				}),
			},
		},
		{
			Name: "Content-type overridden",
			MD: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				ShardKeyHeader, "example-shard-key",
				RoutingKeyHeader, "example-routing-key",
				RoutingDelegateHeader, "example-routing-delegate",
				EncodingHeader, "example-encoding-override",
				contentTypeHeader, "application/grpc+example-encoding",
				"foo", "bar",
				"baz", "bat",
			),
			TransportRequest: &transport.Request{
				Caller:          "example-caller",
				Service:         "example-service",
				ShardKey:        "example-shard-key",
				RoutingKey:      "example-routing-key",
				RoutingDelegate: "example-routing-delegate",
				Encoding:        "example-encoding-override",
				Headers: transport.HeadersFromMap(map[string]string{
					"foo": "bar",
					"baz": "bat",
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			transportRequest, err := metadataToTransportRequest(tt.MD)
			require.Equal(t, tt.Error, err)
			require.Equal(t, tt.TransportRequest, transportRequest)
		})
	}
}

func TestTransportRequestToMetadata(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name             string
		MD               metadata.MD
		TransportRequest *transport.Request
		Error            error
	}{
		{
			Name: "Basic",
			MD: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				ShardKeyHeader, "example-shard-key",
				RoutingKeyHeader, "example-routing-key",
				RoutingDelegateHeader, "example-routing-delegate",
				EncodingHeader, "example-encoding",
				"foo", "bar",
				"baz", "bat",
			),
			TransportRequest: &transport.Request{
				Caller:          "example-caller",
				Service:         "example-service",
				ShardKey:        "example-shard-key",
				RoutingKey:      "example-routing-key",
				RoutingDelegate: "example-routing-delegate",
				Encoding:        "example-encoding",
				Headers: transport.HeadersFromMap(map[string]string{
					"foo": "bar",
					"baz": "bat",
				}),
			},
		},
		{
			Name: "Reserved header key in application headers",
			MD:   metadata.Pairs(),
			TransportRequest: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{
					CallerHeader: "example-caller",
				}),
			},
			Error: yarpcerrors.InvalidArgumentErrorf("cannot use reserved header in application headers: %s", CallerHeader),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			md, err := transportRequestToMetadata(tt.TransportRequest)
			require.Equal(t, tt.Error, err)
			require.Equal(t, tt.MD, md)
		})
	}
}

func TestGetContentSubtype(t *testing.T) {
	tests := []struct {
		contentType    string
		contentSubtype string
	}{
		{"application/grpc", ""},
		{"application/grpc+proto", "proto"},
		{"application/grpc;proto", "proto"},
		{"application/grpc-proto", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.contentSubtype, getContentSubtype(tt.contentType))
	}
}

func TestIsReserved(t *testing.T) {
	assert.True(t, isReserved(CallerHeader))
	assert.True(t, isReserved(ServiceHeader))
	assert.True(t, isReserved(ShardKeyHeader))
	assert.True(t, isReserved(RoutingKeyHeader))
	assert.True(t, isReserved(RoutingDelegateHeader))
	assert.True(t, isReserved(EncodingHeader))
	assert.True(t, isReserved("rpc-foo"))
}

func TestMDReadWriterDuplicateKey(t *testing.T) {
	const key = "uber-trace-id"
	md := map[string][]string{
		key: {"to-override"},
	}
	mdRW := mdReadWriter(md)
	mdRW.Set(key, "overwritten")
	assert.Equal(t, []string{"overwritten"}, md[key], "expected overwritten values")
}
