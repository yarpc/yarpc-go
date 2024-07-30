// Copyright (c) 2024 Uber Technologies, Inc.
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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestToHTTPHeaders(t *testing.T) {
	tests := map[string]struct {
		prefix             string
		reqHeaders         transport.Headers
		httpHeaders        http.Header
		enforceHeaderRules bool
		expHeaders         http.Header
		expReportHeader    bool
		expError           error
	}{
		"http-headers-nil": {
			prefix: "Rpc-Header-",
			reqHeaders: transport.HeadersFromMap(map[string]string{
				"foo":     "bar",
				"foo-bar": "any-value",
			}),
			expHeaders: http.Header{
				"Rpc-Header-Foo":     []string{"bar"},
				"Rpc-Header-Foo-Bar": []string{"any-value"},
			},
		},
		"success": {
			prefix: "Rpc-Header-",
			reqHeaders: transport.HeadersFromMap(map[string]string{
				"foo":     "bar",
				"foo-bar": "value-2",
			}),
			httpHeaders: http.Header{
				"Rpc-Header-Foo-Bar": []string{"value-1"},
				"any-header":         []string{"any-value"},
			},
			expHeaders: http.Header{
				"Rpc-Header-Foo":     []string{"bar"},
				"Rpc-Header-Foo-Bar": []string{"value-1", "value-2"},
				"any-header":         []string{"any-value"},
			},
		},
		"reserved-rpc-header-passed": {
			prefix: "Rpc-Header-",
			reqHeaders: transport.HeadersFromMap(map[string]string{
				"rpc-foo": "any-value",
				"baz":     "bar",
			}),
			expHeaders: http.Header{
				"Rpc-Header-Rpc-Foo": []string{"any-value"},
				"Rpc-Header-Baz":     []string{"bar"},
			},
			expReportHeader: true,
		},
		"reserved-dollar-rpc-header-passed": {
			prefix: "Rpc-Header-",
			reqHeaders: transport.HeadersFromMap(map[string]string{
				"$rpc$-foo": "any-value",
				"baz":       "bar",
			}),
			expHeaders: http.Header{
				"Rpc-Header-$rpc$-Foo": []string{"any-value"},
				"Rpc-Header-Baz":       []string{"bar"},
			},
			expReportHeader: true,
		},
		"reserved-rpc-header-passed-enforce-header-rules": {
			prefix: "Rpc-Header-",
			reqHeaders: transport.HeadersFromMap(map[string]string{
				"rpc-foo": "any-value",
				"baz":     "bar",
			}),
			enforceHeaderRules: true,
			expReportHeader:    true,
			expError:           yarpcerrors.InternalErrorf("cannot use reserved header in application headers: rpc-foo"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRules)

			root := metrics.New()
			edgeMetrics := observability.NewReserveHeaderMetrics(root.Scope(), "http").With("any-source", "any-dest")

			hm := headerMapper{tt.prefix}

			origTransportHeaders := transport.HeadersFromMap(tt.reqHeaders.OriginalItems())

			httpHeaders, err := hm.ToHTTPHeaders(tt.reqHeaders, tt.httpHeaders, edgeMetrics)
			assert.Equal(t, origTransportHeaders, tt.reqHeaders, "passed request headers should not be modified")
			assert.Equal(t, tt.expHeaders, httpHeaders)
			assert.Equal(t, tt.expError, err)

			if tt.expReportHeader {
				if tt.expReportHeader {
					assertTuple(t, root.Snapshot().Counters, tuple{"http_reserved_headers_error", "any-source", "any-dest", 1})
				} else {
					assertEmptyMetrics(t, root.Snapshot())
				}
			}
		})

	}
}

func TestFromHTTPHeaders(t *testing.T) {
	test := map[string]struct {
		prefix             string
		httpHeaders        http.Header
		reqHeaders         transport.Headers
		enforceHeaderRules bool
		expHeaders         transport.Headers
		expReportHeader    bool
		expError           error
	}{
		"empty-req-header": {
			prefix: "Rpc-Header-",
			httpHeaders: http.Header{
				"Rpc-Header-Foo": []string{"bar"},
			},
			expHeaders: transport.NewHeaders().With("Foo", "bar"),
		},
		"req-header-is-passed": {
			prefix: "Rpc-Header-",
			httpHeaders: http.Header{
				"Rpc-Header-Foo": []string{"bar"},
			},
			reqHeaders: transport.NewHeaders().With("any-key", "any-value"),
			expHeaders: transport.NewHeaders().With("Foo", "bar").With("any-key", "any-value"),
		},
		"conflicting-req-header": {
			prefix: "Rpc-Header-",
			httpHeaders: http.Header{
				"Rpc-Header-Foo": []string{"value-1"},
			},
			reqHeaders: transport.NewHeaders().With("Foo", "value-2"),
			expHeaders: transport.NewHeaders().With("Foo", "value-1"),
		},
		"multiple-http-header-values": {
			prefix: "Rpc-Header-",
			httpHeaders: http.Header{
				"Rpc-Header-Foo": []string{"value-1", "value-2"},
			},
			expHeaders: transport.NewHeaders().With("Foo", "value-1"),
		},
		"reserved-rpc-header-passed": {
			prefix: "Rpc-Header-",
			httpHeaders: http.Header{
				"Rpc-Header-Rpc-Foo": []string{"any-value"},
				"Rpc-Header-Baz":     []string{"bar"},
			},
			expHeaders:      transport.NewHeaders().With("Rpc-Foo", "any-value").With("Baz", "bar"),
			expReportHeader: true,
		},
		"reserved-dollar-rpc-header-passed": {
			prefix: "Rpc-Header-",
			httpHeaders: http.Header{
				"Rpc-Header-$rpc$-Foo": []string{"any-value"},
				"Rpc-Header-Baz":       []string{"bar"},
			},
			expHeaders:      transport.NewHeaders().With("$rpc$-Foo", "any-value").With("Baz", "bar"),
			expReportHeader: true,
		},
		"reserved-rpc-header-passed-enforce-header-rules": {
			prefix: "Rpc-Header-",
			httpHeaders: http.Header{
				"Rpc-Header-Rpc-Foo": []string{"any-value"},
				"Rpc-Header-Baz":     []string{"bar"},
			},
			enforceHeaderRules: true,
			expHeaders:         transport.NewHeaders().With("Baz", "bar"),
			expReportHeader:    true,
		},
	}

	for name, tt := range test {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRules)

			root := metrics.New()
			edgeMetrics := observability.NewReserveHeaderMetrics(root.Scope(), "http").With("any-source", "any-dest")

			hm := headerMapper{tt.prefix}

			origHTTPHeaders := tt.httpHeaders.Clone()

			reqHeaders := hm.FromHTTPHeaders(tt.httpHeaders, tt.reqHeaders, edgeMetrics)
			assert.Equal(t, origHTTPHeaders, tt.httpHeaders, "passed http headers should not be modified")
			assert.Equal(t, tt.expHeaders, reqHeaders)

			if tt.expReportHeader {
				if tt.expReportHeader {
					assertTuple(t, root.Snapshot().Counters, tuple{"http_reserved_headers_stripped", "any-source", "any-dest", 1})
				} else {
					assertEmptyMetrics(t, root.Snapshot())
				}
			}
		})
	}
}

func switchEnforceHeaderRules(t *testing.T, cond bool) {
	if !cond {
		return
	}

	enforceHeaderRules = true
	t.Cleanup(func() {
		enforceHeaderRules = false
	})
}

type tuple struct {
	name, tag1, tag2 string
	value            int64
}

func assertTuple(t *testing.T, snapshots []metrics.Snapshot, expected tuple) {
	assertTuples(t, snapshots, []tuple{expected})
}

func assertTuples(t *testing.T, snapshots []metrics.Snapshot, expected []tuple) {
	actual := make([]tuple, 0, len(snapshots))

	for _, c := range snapshots {
		actual = append(actual, tuple{c.Name, c.Tags["source"], c.Tags["dest"], c.Value})
	}

	assert.ElementsMatch(t, expected, actual)
}

func assertEmptyMetrics(t *testing.T, snapshot *metrics.RootSnapshot) {
	assert.Empty(t, snapshot.Counters)
	assert.Empty(t, snapshot.Gauges)
	assert.Empty(t, snapshot.Histograms)
}

// TODO(abg): Test handling of duplicate HTTP headers when
// https://github.com/yarpc/yarpc/issues/21 is resolved.
