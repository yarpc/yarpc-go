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

package http

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
)

// BenchmarkTTLFormatting compares three approaches to formatting the TTL header value.
//
//	go test -bench=BenchmarkTTLFormatting -benchmem ./transport/http
func BenchmarkTTLFormatting(b *testing.B) {
	ttl := 5 * time.Second
	ttlMS := int64(ttl / time.Millisecond)

	b.Run("fmt.Sprintf", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("%d", ttlMS)
		}
	})

	b.Run("strconv.FormatInt", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = strconv.FormatInt(ttlMS, 10)
		}
	})

	b.Run("strconv.AppendInt", func(b *testing.B) {
		b.ReportAllocs()
		var buf [20]byte
		for i := 0; i < b.N; i++ {
			_ = string(strconv.AppendInt(buf[:0], ttlMS, 10))
		}
	})
}

// BenchmarkWithCoreHeaders measures the performance of withCoreHeaders,
// which is called on every HTTP outbound request.
//
//	go test -bench=BenchmarkWithCoreHeaders -benchmem ./transport/http
func BenchmarkWithCoreHeaders(b *testing.B) {
	o := &Outbound{
		bothResponseError: true,
	}

	treq := &transport.Request{
		Caller:          "my-caller-service",
		Service:         "my-target-service",
		Procedure:       "MyService::MyMethod",
		Encoding:        "proto",
		ShardKey:        "shard-123",
		RoutingKey:      "routing-key-456",
		RoutingDelegate: "routing-delegate-789",
		CallerProcedure: "CallerService::CallerMethod",
	}

	ttl := 5 * time.Second

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "http://localhost/", nil)
		req.Header = make(http.Header, 10)
		o.withCoreHeaders(req, treq, ttl)
	}
}

// BenchmarkWithCoreHeaders_MinimalFields benchmarks withCoreHeaders with only
// required fields populated (no optional shard/routing keys).
func BenchmarkWithCoreHeaders_MinimalFields(b *testing.B) {
	o := &Outbound{
		bothResponseError: true,
	}

	treq := &transport.Request{
		Caller:    "my-caller-service",
		Service:   "my-target-service",
		Procedure: "MyService::MyMethod",
		Encoding:  "proto",
	}

	ttl := 1 * time.Second

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "http://localhost/", nil)
		req.Header = make(http.Header, 10)
		o.withCoreHeaders(req, treq, ttl)
	}
}
