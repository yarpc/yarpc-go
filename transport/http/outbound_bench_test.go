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
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
)

// BenchmarkOutboundBuildRequest measures the end-to-end CPU and allocation
// cost of constructing an outbound *http.Request — the per-call hot path —
// comparing the HTTP/1.1 path (which canonicalizes header names via
// textproto.CanonicalMIMEHeaderKey) against the HTTP/2 fast path (which
// writes directly into the http.Header map).
//
// Three header profiles are exercised:
//
//   - many headers, all lowercase  (best case for canonicalization)
//   - many headers, mixed case     (worst case — every key allocates)
//   - few headers, mixed case      (typical YARPC workload)
//
// "many" = 24 (a realistic upper bound); "few" = 4.
func BenchmarkOutboundBuildRequest(b *testing.B) {
	const benchTTL = 500 * time.Millisecond

	for _, tc := range benchScenarios() {
		b.Run(tc.name, func(b *testing.B) {
			b.Run("http1_canonical", func(b *testing.B) {
				benchBuildRequest(b, benchOutbound(false), tc.headers, benchTTL)
			})
			b.Run("http2_direct", func(b *testing.B) {
				benchBuildRequest(b, benchOutbound(true), tc.headers, benchTTL)
			})
		})
	}
}

func benchBuildRequest(b *testing.B, o *Outbound, headers transport.Headers, ttl time.Duration) {
	treq := benchTransportRequest(headers)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hreq, err := o.createRequest(treq)
		if err != nil {
			b.Fatal(err)
		}
		o.withCoreHeaders(hreq, treq, ttl)
	}
}

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

type headerBenchScenario struct {
	name    string
	headers transport.Headers
}

func benchScenarios() []headerBenchScenario {
	return []headerBenchScenario{
		{
			name:    "many_headers_all_lowercase",
			headers: makeBenchHeaders(24, lowercaseOnly),
		},
		{
			name:    "many_headers_mixed_case",
			headers: makeBenchHeaders(24, mixedCase),
		},
		{
			name:    "few_headers_mixed_case",
			headers: makeBenchHeaders(4, mixedCase),
		},
	}
}

type caseStyle int

const (
	lowercaseOnly caseStyle = iota
	mixedCase
)

func makeBenchHeaders(n int, style caseStyle) transport.Headers {
	h := transport.NewHeadersWithCapacity(n)
	for i := 0; i < n; i++ {
		var key string
		switch style {
		case lowercaseOnly:
			key = fmt.Sprintf("x-custom-header-%03d", i)
		case mixedCase:
			switch i % 3 {
			case 0:
				key = fmt.Sprintf("X-Custom-Header-%03d", i)
			case 1:
				key = fmt.Sprintf("x-custom-HeaDer-%03d", i)
			default:
				key = fmt.Sprintf("x-custom-header-%03d", i)
			}
		}
		// Vary value length so the compiler can't constant-fold.
		h = h.With(key, fmt.Sprintf("value-%d-padding-abcdefgh", i))
	}
	return h
}

// benchOutbound returns an Outbound with two static extra headers, matching a
// realistic config (e.g. addHeaders in YAML).
func benchOutbound(useHTTP2 bool) *Outbound {
	return &Outbound{
		urlTemplate:       defaultURLTemplate,
		bothResponseError: true,
		useHTTP2:          useHTTP2,
		headers: http.Header{
			"X-Static-Token": []string{"tok_abc123"},
			"X-Request-Id":   []string{"rid-00000000"},
		},
	}
}

func benchTransportRequest(headers transport.Headers) *transport.Request {
	return &transport.Request{
		Caller:          "bench-caller",
		Service:         "bench-service",
		Encoding:        "raw",
		Procedure:       "BenchProcedure",
		ShardKey:        "shard-42",
		RoutingKey:      "routing-key",
		RoutingDelegate: "delegate-svc",
		CallerProcedure: "CallerProc",
		Headers:         headers,
	}
}
