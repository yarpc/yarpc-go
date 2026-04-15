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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

func TestHTTPHeaders(t *testing.T) {
	tests := []struct {
		name                 string
		transHeaders         transport.Headers
		expectedHTTP         http.Header
		httpHeaders          http.Header
		expectedTransHeaders transport.Headers
	}{
		{
			name: "application only",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"foo":     "bar",
				"foo-bar": "hello",
			}),
			expectedHTTP: http.Header{
				"Rpc-Header-Foo":     []string{"bar"},
				"Rpc-Header-Foo-Bar": []string{"hello"},
			},
			httpHeaders: http.Header{
				"Rpc-Header-Foo":     []string{"bar"},
				"Rpc-Header-Foo-Bar": []string{"hello"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{
				"foo":     "bar",
				"foo-bar": "hello",
			}),
		},
		{
			name: "tracing only",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"uber-trace-id": "tid",
				"uberctx-foo":   "ctxval",
			}),
			expectedHTTP: http.Header{
				"Uber-Trace-Id": []string{"tid"},
				"Uberctx-Foo":   []string{"ctxval"},
			},
			httpHeaders: http.Header{
				"Uber-Trace-Id": []string{"tid"},
				"Uberctx-Foo":   []string{"ctxval"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{
				"uber-trace-id": "tid",
				"uberctx-foo":   "ctxval",
			}),
		},
		{
			name: "mixed headers",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"foo":           "bar",
				"uber-trace-id": "tid",
			}),
			expectedHTTP: http.Header{
				"Rpc-Header-Foo": []string{"bar"},
				"Uber-Trace-Id":  []string{"tid"},
			},
			httpHeaders: http.Header{
				"Rpc-Header-Foo": []string{"bar"},
				"Uber-Trace-Id":  []string{"tid"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{
				"foo":           "bar",
				"uber-trace-id": "tid",
			}),
		},
		{
			name: "routing headers",
			transHeaders: transport.HeadersFromMap(map[string]string{
				RoutingRegionHeader: "routingRegion",
				RoutingZoneHeader:   "routingZone",
			}),
			expectedHTTP: http.Header{
				RoutingRegionHeader: []string{"routingRegion"},
				RoutingZoneHeader:   []string{"routingZone"},
			},
			httpHeaders: http.Header{
				RoutingRegionHeader: []string{"routingRegion"},
				RoutingZoneHeader:   []string{"routingZone"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{}),
		},
		{
			name: "proxy header outbound still prefixed",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"x-forwarded-for": "203.0.113.42",
			}),
			expectedHTTP: http.Header{
				"Rpc-Header-X-Forwarded-For": []string{"203.0.113.42"},
			},
			httpHeaders: http.Header{
				"X-Forwarded-For": []string{"203.0.113.42"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{
				"X-Forwarded-For": "203.0.113.42",
			}),
		},
		{
			name: "proxy header inbound with mixed headers",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"foo":           "bar",
				"uber-trace-id": "tid",
			}),
			expectedHTTP: http.Header{
				"Rpc-Header-Foo": []string{"bar"},
				"Uber-Trace-Id":  []string{"tid"},
			},
			httpHeaders: http.Header{
				"Rpc-Header-Foo":  []string{"bar"},
				"X-Forwarded-For": []string{"203.0.113.42"},
				"Uber-Trace-Id":   []string{"tid"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{
				"foo":             "bar",
				"X-Forwarded-For": "203.0.113.42",
				"uber-trace-id":   "tid",
			}),
		},
		{
			name: "all proxy headers inbound",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"foo": "bar",
			}),
			expectedHTTP: http.Header{
				"Rpc-Header-Foo": []string{"bar"},
			},
			httpHeaders: http.Header{
				"Rpc-Header-Foo":  []string{"bar"},
				"X-Forwarded-For": []string{"203.0.113.42"},
				"X-Forwarded-Proto": []string{"https"},
				"X-Forwarded-Port":  []string{"8080"},
				"X-Request-Id":      []string{"req-123"},
				"X-Uber-Source":     []string{"service-a"},
				"Via":               []string{"1.1 proxy"},
				"User-Agent":        []string{"yarpc/1.0"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{
				"foo":              "bar",
				"X-Forwarded-For":  "203.0.113.42",
				"X-Forwarded-Proto": "https",
				"X-Forwarded-Port":  "8080",
				"X-Request-Id":      "req-123",
				"X-Uber-Source":     "service-a",
				"Via":               "1.1 proxy",
				"User-Agent":        "yarpc/1.0",
			}),
		},
		{
			name: "prefixed proxy header wins over unprefixed",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"x-forwarded-for": "203.0.113.42",
			}),
			expectedHTTP: http.Header{
				"Rpc-Header-X-Forwarded-For": []string{"203.0.113.42"},
			},
			httpHeaders: http.Header{
				"X-Forwarded-For":            []string{"10.0.0.1"},
				"Rpc-Header-X-Forwarded-For": []string{"203.0.113.42"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{
				"X-Forwarded-For": "203.0.113.42",
			}),
		},
		{
			name: "prefixed tracing header wins over unprefixed",
			transHeaders: transport.HeadersFromMap(map[string]string{
				"uber-trace-id": "correct-tid",
			}),
			expectedHTTP: http.Header{
				"Uber-Trace-Id": []string{"correct-tid"},
			},
			httpHeaders: http.Header{
				"Uber-Trace-Id":            []string{"stale-tid"},
				"Rpc-Header-Uber-Trace-Id": []string{"correct-tid"},
			},
			expectedTransHeaders: transport.HeadersFromMap(map[string]string{
				"Uber-Trace-Id": "correct-tid",
			}),
		},
	}

	for _, tt := range tests {
		hm := headerMapper{ApplicationHeaderPrefix}
		gotHTTP := hm.ToHTTPHeaders(tt.transHeaders, nil)
		assertHeadersEqualCaseInsensitive(t, tt.expectedHTTP, gotHTTP, fmt.Sprintf("%s: ToHTTPHeaders", tt.name))
		gotTrans := hm.FromHTTPHeaders(tt.httpHeaders, transport.Headers{})
		assert.Equal(t, tt.expectedTransHeaders.Items(), gotTrans.Items(), "%s: FromHTTPHeaders", tt.name)
	}
}

func assertHeadersEqualCaseInsensitive(t *testing.T, expected, actual http.Header, msg string) {
	normalize := func(headers http.Header) http.Header {
		normalized := make(http.Header, len(headers))
		for k, v := range headers {
			normalized[strings.ToLower(k)] = v
		}
		return normalized
	}

	assert.Equal(t, normalize(expected), normalize(actual), msg)
}

func TestIsProxyHeader(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"x-forwarded-for", true},
		{"X-Forwarded-For", true},
		{"X-FORWARDED-FOR", true},
		{"x-forwarded-proto", true},
		{"X-Forwarded-Proto", true},
		{"x-forwarded-port", true},
		{"X-Forwarded-Port", true},
		{"x-request-id", true},
		{"X-Request-Id", true},
		{"x-uber-source", true},
		{"X-Uber-Source", true},
		{"via", true},
		{"Via", true},
		{"user-agent", true},
		{"User-Agent", true},
		{"x-other-header", false},
		{"x-forwarded-host", false},
		{"rpc-header-foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.want, isProxyHeader(tt.key))
		})
	}
}

// TODO(abg): Test handling of duplicate HTTP headers when
// https://github.com/yarpc/yarpc/issues/21 is resolved.
