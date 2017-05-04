// Copyright (c) 2017 Uber Technologies, Inc.
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

	"go.uber.org/yarpc/api/transport"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestMetadataToTransportRequest(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		Name             string
		MD               metadata.MD
		TransportRequest *transport.Request
	}{
		{
			Name: "Basic",
			MD: metadata.Pairs(
				callerHeader, "example-caller",
				serviceHeader, "example-service",
				shardKeyHeader, "example-shard-key",
				routingKeyHeader, "example-routing-key",
				routingDelegateHeader, "example-routing-delegate",
				encodingHeader, "example-encoding",
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
	} {
		t.Run(tt.Name, func(t *testing.T) {
			transportRequest, err := metadataToTransportRequest(tt.MD)
			require.NoError(t, err)
			require.Equal(t, tt.TransportRequest, transportRequest)
		})
	}
}
