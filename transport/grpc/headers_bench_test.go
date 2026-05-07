// Copyright (c) 2026 Uber Technologies, Inc.
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
	"google.golang.org/grpc/metadata"
)

// BenchmarkTransportRequestToMetadata measures outbound header serialization.
//
//	go test -bench=BenchmarkTransportRequestToMetadata -benchmem ./transport/grpc/
func BenchmarkTransportRequestToMetadata(b *testing.B) {
	request := &transport.Request{
		Caller:          "my-caller",
		Service:         "my-service",
		Procedure:       "MyService::MyMethod",
		Encoding:        "proto",
		ShardKey:        "shard-123",
		RoutingKey:      "routing-456",
		RoutingDelegate: "delegate-789",
		CallerProcedure: "CallerService::CallerMethod",
		Headers: transport.HeadersFromMap(map[string]string{
			"x-custom-1": "value1",
			"x-custom-2": "value2",
			"x-custom-3": "value3",
		}),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = transportRequestToMetadata(request)
	}
}

// BenchmarkMetadataToTransportRequest measures inbound header deserialization.
//
//	go test -bench=BenchmarkMetadataToTransportRequest -benchmem ./transport/grpc/
func BenchmarkMetadataToTransportRequest(b *testing.B) {
	md := metadata.New(map[string]string{
		CallerHeader:          "my-caller",
		ServiceHeader:         "my-service",
		ShardKeyHeader:        "shard-123",
		RoutingKeyHeader:      "routing-456",
		RoutingDelegateHeader: "delegate-789",
		EncodingHeader:        "proto",
		CallerProcedureHeader: "CallerService::CallerMethod",
		"x-custom-1":          "value1",
		"x-custom-2":          "value2",
		"x-custom-3":          "value3",
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = metadataToTransportRequest(md)
	}
}

// BenchmarkGetApplicationHeaders measures inbound response header extraction.
//
//	go test -bench=BenchmarkGetApplicationHeaders -benchmem ./transport/grpc/
func BenchmarkGetApplicationHeaders(b *testing.B) {
	md := metadata.New(map[string]string{
		CallerHeader:   "my-caller",
		ServiceHeader:  "my-service",
		EncodingHeader: "proto",
		"x-custom-1":   "value1",
		"x-custom-2":   "value2",
		"x-custom-3":   "value3",
		"x-custom-4":   "value4",
		"x-custom-5":   "value5",
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getApplicationHeaders(md)
	}
}
