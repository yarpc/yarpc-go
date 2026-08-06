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

	"go.uber.org/yarpc/api/transport"
)

func BenchmarkFromHTTPHeadersCapacity(b *testing.B) {
	type headerMode struct {
		name    string
		prepare func(transport.Headers) transport.Headers
	}
	type capacityMode struct {
		name     string
		capacity func() int
	}
	type scenario struct {
		name                string
		applicationHeaders  int
		tracingHeaders      int
		ignoredHeaders      int
		poppedSystemHeaders int
		useCapacityHint     bool
		modes               []headerMode
	}

	defaultMode := headerMode{
		name:    "default",
		prepare: func(h transport.Headers) transport.Headers { return h },
	}
	headerCaseMapping := map[string][]string{
		"application-0": {"aPPlication-0", "APPLICATION-0"},
		"application-1": {"aPPlication-1", "APPLICATION-1"},
	}
	scenarios := []scenario{
		{
			name:                "small",
			applicationHeaders:  2,
			tracingHeaders:      2,
			poppedSystemHeaders: 2,
			modes:               []headerMode{defaultMode},
		},
		{
			name:                "typical",
			applicationHeaders:  10,
			tracingHeaders:      4,
			poppedSystemHeaders: 2,
			modes: []headerMode{
				defaultMode,
				{
					name: "case-mapping",
					prepare: func(h transport.Headers) transport.Headers {
						return h.WithHeaderCaseMapping(headerCaseMapping)
					},
				},
			},
		},
		{
			name:                "header-heavy",
			applicationHeaders:  25,
			tracingHeaders:      8,
			poppedSystemHeaders: 6,
			useCapacityHint:     true,
			modes:               []headerMode{defaultMode},
		},
		{
			name:                "mostly-ignored",
			applicationHeaders:  2,
			tracingHeaders:      2,
			ignoredHeaders:      20,
			poppedSystemHeaders: 2,
			modes:               []headerMode{defaultMode},
		},
	}

	for _, scenario := range scenarios {
		from := makeBenchmarkInboundHeaders(
			scenario.applicationHeaders,
			scenario.tracingHeaders,
			scenario.ignoredHeaders,
		)
		eligibleHeaders := scenario.applicationHeaders + scenario.tracingHeaders

		for _, mode := range scenario.modes {
			mode := mode
			b.Run(scenario.name+"/"+mode.name, func(b *testing.B) {
				benchmarks := []capacityMode{
					{
						name:     "zero",
						capacity: func() int { return 0 },
					},
					{
						name: "before-system-pop",
						capacity: func() int {
							return len(from) + scenario.poppedSystemHeaders
						},
					},
					{
						name:     "after-system-pop",
						capacity: func() int { return len(from) },
					},
					{
						name:     "estimated",
						capacity: func() int { return inboundHeaderCapacity(from) },
					},
				}
				if scenario.useCapacityHint {
					benchmarks = append(benchmarks, capacityMode{
						name: "hinted",
						capacity: func() int {
							return inboundHeaderCapacityWithHint(from, eligibleHeaders)
						},
					})
				}

				for _, benchmark := range benchmarks {
					benchmark := benchmark
					b.Run(benchmark.name, func(b *testing.B) {
						b.ReportAllocs()

						var got transport.Headers
						for range b.N {
							got = transport.NewHeadersWithCapacity(benchmark.capacity())
							got = mode.prepare(got)
							got = applicationHeaders.FromHTTPHeaders(from, got)
						}

						if got.Len() != eligibleHeaders {
							b.Fatalf("got %d headers, want %d", got.Len(), eligibleHeaders)
						}
					})
				}
			})
		}
	}
}

func makeBenchmarkInboundHeaders(application, tracing, ignored int) http.Header {
	headers := make(http.Header, application+tracing+ignored+1)
	headers.Set(TTLMSHeader, "1000")

	for i := 0; i < application; i++ {
		headers.Set(fmt.Sprintf("%sApplication-%d", ApplicationHeaderPrefix, i), "application-value")
	}
	if tracing > 0 {
		headers.Set(UberTraceContextHeaderKey, "trace-value")
	}
	for i := 1; i < tracing; i++ {
		headers.Set(fmt.Sprintf("%sBenchmark-%d", UberBaggageHeaderKeyPrefix, i), "baggage-value")
	}
	for i := 0; i < ignored; i++ {
		headers.Set(fmt.Sprintf("X-Ignored-%d", i), "ignored-value")
	}

	return headers
}
