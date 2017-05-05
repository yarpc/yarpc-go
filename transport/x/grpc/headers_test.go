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
	"fmt"
	"testing"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/x/grpc/grpcheader"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestMetadataToTransportRequest(t *testing.T) {
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
				grpcheader.CallerHeader, "example-caller",
				grpcheader.ServiceHeader, "example-service",
				grpcheader.ShardKeyHeader, "example-shard-key",
				grpcheader.RoutingKeyHeader, "example-routing-key",
				grpcheader.RoutingDelegateHeader, "example-routing-delegate",
				grpcheader.EncodingHeader, "example-encoding",
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
				grpcheader.CallerHeader, "example-caller",
				grpcheader.ServiceHeader, "example-service",
				grpcheader.ShardKeyHeader, "example-shard-key",
				grpcheader.RoutingKeyHeader, "example-routing-key",
				grpcheader.RoutingDelegateHeader, "example-routing-delegate",
				grpcheader.EncodingHeader, "example-encoding",
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
					grpcheader.CallerHeader: "example-caller",
				}),
			},
			Error: fmt.Errorf("cannot use reserved header in application headers: %s", grpcheader.CallerHeader),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			md, err := transportRequestToMetadata(tt.TransportRequest)
			require.Equal(t, tt.Error, err)
			require.Equal(t, tt.MD, md)
		})
	}
}
