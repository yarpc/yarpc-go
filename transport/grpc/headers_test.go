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

package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc/metadata"
)

func TestMetadataToTransportRequest(t *testing.T) {
	tests := map[string]struct {
		md                 metadata.MD
		req                *transport.Request
		enforceHeaderRules bool
		expErr             error
		expReportHeader    bool
	}{
		"basic": {
			md: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				ShardKeyHeader, "example-shard-key",
				RoutingKeyHeader, "example-routing-key",
				RoutingDelegateHeader, "example-routing-delegate",
				EncodingHeader, "example-encoding",
				CallerProcedureHeader, "example-caller-procedure",
				"foo", "bar",
				"baz", "bat",
			),
			req: &transport.Request{
				Caller:          "example-caller",
				Service:         "example-service",
				ShardKey:        "example-shard-key",
				RoutingKey:      "example-routing-key",
				RoutingDelegate: "example-routing-delegate",
				Encoding:        "example-encoding",
				CallerProcedure: "example-caller-procedure",
				Headers: transport.HeadersFromMap(map[string]string{
					"foo": "bar",
					"baz": "bat",
				}),
			},
		},
		"content-type": {
			md: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				ShardKeyHeader, "example-shard-key",
				RoutingKeyHeader, "example-routing-key",
				RoutingDelegateHeader, "example-routing-delegate",
				contentTypeHeader, "application/grpc+example-encoding",
				"foo", "bar",
				"baz", "bat",
			),
			req: &transport.Request{
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
		"content-type-overridden": {
			md: metadata.Pairs(
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
			req: &transport.Request{
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
		"Reserved header key with rpc prefix in application headers": {
			md: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				"rpc-any", "any-value",
			),
			req: &transport.Request{
				Caller:  "example-caller",
				Service: "example-service",
				Headers: transport.HeadersFromMap(map[string]string{"rpc-any": "any-value"}),
			},
			expReportHeader: true,
		},
		"Reserved header key with $rpc$ prefix in application headers": {
			md: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				"$rpc$-any", "any-value",
			),
			req: &transport.Request{
				Caller:  "example-caller",
				Service: "example-service",
				Headers: transport.HeadersFromMap(map[string]string{"$rpc$-any": "any-value"}),
			},
			expReportHeader: true,
		},
		"Reserved headers rules are enforced": {
			md: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				"rpc-any", "any-value",
				"$rpc$-any", "any-value",
				"foo", "bar",
				"baz", "bat",
			),
			req: &transport.Request{
				Caller:  "example-caller",
				Service: "example-service",
				Headers: transport.HeadersFromMap(map[string]string{
					"foo": "bar",
					"baz": "bat",
				}),
			},
			enforceHeaderRules: true,
			expReportHeader:    true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRules)

			root := metrics.New()
			m := observability.NewReserveHeaderMetrics(root.Scope(), "grpc")

			transportRequest, err := metadataToInboundRequest(tt.md, m)
			assert.Equal(t, tt.expErr, err)
			assert.Equal(t, tt.req, transportRequest)

			if tt.expReportHeader {
				assertTuple(t, root.Snapshot().Counters, tuple{"grpc_reserved_headers_stripped", "example-caller", "example-service", 1})
			} else {
				assertEmptyMetrics(t, root.Snapshot())
			}
		})
	}
}

func TestTransportRequestToMetadata(t *testing.T) {
	for name, tt := range map[string]struct {
		md                 metadata.MD
		req                *transport.Request
		enforceHeaderRules bool
		expErr             error
		expReportHeader    bool
	}{
		"basic": {
			md: metadata.Pairs(
				CallerHeader, "example-caller",
				ServiceHeader, "example-service",
				ShardKeyHeader, "example-shard-key",
				RoutingKeyHeader, "example-routing-key",
				RoutingDelegateHeader, "example-routing-delegate",
				CallerProcedureHeader, "example-caller-procedure",
				EncodingHeader, "example-encoding",
				"foo", "bar",
				"baz", "bat",
			),
			req: &transport.Request{
				Caller:          "example-caller",
				Service:         "example-service",
				ShardKey:        "example-shard-key",
				RoutingKey:      "example-routing-key",
				RoutingDelegate: "example-routing-delegate",
				CallerProcedure: "example-caller-procedure",
				Encoding:        "example-encoding",
				Headers: transport.HeadersFromMap(map[string]string{
					"foo": "bar",
					"baz": "bat",
				}),
			},
		},
		"Reserved header key in application headers": {
			md: metadata.Pairs(),
			req: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{
					CallerHeader: "example-caller",
				}),
			},
			expErr: yarpcerrors.InvalidArgumentErrorf("cannot use reserved header in application headers: %s", CallerHeader),
		},
		"Reserved header key with $rpc$ prefix in application headers": {
			md: metadata.Pairs("$rpc$-any", "example-caller"),
			req: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{
					"$rpc$-any": "example-caller",
				}),
			},
			expErr:          nil,
			expReportHeader: true,
		},
		"Reserved header key with $rpc$ prefix in application headers with enforced rules": {
			md: metadata.Pairs(),
			req: &transport.Request{
				Headers: transport.HeadersFromMap(map[string]string{
					"$rpc$-any": "example-caller",
				}),
			},
			enforceHeaderRules: true,
			expErr:             yarpcerrors.InternalErrorf("cannot use reserved header in application headers: $rpc$-any"),
			expReportHeader:    true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRules)
			root := metrics.New()
			edgeMetrics := observability.NewReserveHeaderMetrics(root.Scope(), "grpc").With("any-source", "any-dest")

			md, err := outboundRequestToMetadata(tt.req, edgeMetrics)
			assert.Equal(t, tt.expErr, err)
			assert.Equal(t, tt.md, md)

			if tt.expReportHeader {
				assertTuple(t, root.Snapshot().Counters, tuple{"grpc_reserved_headers_error", "any-source", "any-dest", 1})
			} else {
				assertEmptyMetrics(t, root.Snapshot())
			}
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

func TestIsReservedHeaderPrefixV1(t *testing.T) {
	assert.True(t, isReservedHeaderPrefixV1(CallerHeader))
	assert.True(t, isReservedHeaderPrefixV1(ServiceHeader))
	assert.True(t, isReservedHeaderPrefixV1(ShardKeyHeader))
	assert.True(t, isReservedHeaderPrefixV1(RoutingKeyHeader))
	assert.True(t, isReservedHeaderPrefixV1(RoutingDelegateHeader))
	assert.True(t, isReservedHeaderPrefixV1(EncodingHeader))
	assert.True(t, isReservedHeaderPrefixV1("rpc-foo"))
	assert.False(t, isReservedHeaderPrefixV1("$rpc$-foo"))
}

func TestIsReservedHeaderPrefixV2(t *testing.T) {
	assert.True(t, isReservedHeaderPrefixV2("rpc-foo"))
	assert.True(t, isReservedHeaderPrefixV2("$rpc$-foo"))
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

func TestGetApplicationHeaders(t *testing.T) {
	tests := map[string]struct {
		md                 metadata.MD
		enforceHeaderRules bool
		expHeaders         map[string]string
		expErr             error
		expReportHeader    bool
	}{
		"nil": {
			md:         nil,
			expHeaders: nil,
		},
		"empty": {
			md:         metadata.MD{},
			expHeaders: nil,
		},
		"success": {
			md: metadata.MD{
				"rpc-service":         []string{"foo"}, // reserved header
				"test-header-empty":   []string{},      // no value
				"test-header-valid-1": []string{"test-value-1"},
				"test-Header-Valid-2": []string{"test-value-2"},
			},
			expHeaders: map[string]string{
				"test-header-valid-1": "test-value-1",
				"test-header-valid-2": "test-value-2",
			},
		},
		"error: multiple values for one header": {
			md: metadata.MD{
				"test-header-valid": []string{"test-value"},
				"test-header-dup":   []string{"test-value-1", "test-value-2"},
			},
			expErr: yarpcerrors.InvalidArgumentErrorf("header has more than one value: test-header-dup:[test-value-1 test-value-2]"),
		},
		"reserved header": {
			md: metadata.MD{
				"$rpc$-any": []string{"test-value"},
			},
			expHeaders:      map[string]string{"$rpc$-any": "test-value"},
			expReportHeader: true,
		},
		"reserved header with enforced header rules": {
			md: metadata.MD{
				"rpc-any":   []string{"test-value"},
				"$rpc$-any": []string{"test-value"},
				"foo":       []string{"bar"},
			},
			enforceHeaderRules: true,
			expHeaders:         map[string]string{"foo": "bar"},
			expReportHeader:    true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRules)
			switchEnforceHeaderRules(t, tt.enforceHeaderRules)
			root := metrics.New()
			edgeMetrics := observability.NewReserveHeaderMetrics(root.Scope(), "grpc").With("any-source", "any-dest")

			headers, err := getOutboundResponseApplicationHeaders(tt.md, edgeMetrics)
			assert.Equal(t, tt.expErr, err)
			assert.Equal(t, tt.expHeaders, headers.Items())

			if tt.expReportHeader {
				assertTuple(t, root.Snapshot().Counters, tuple{"grpc_reserved_headers_stripped", "any-source", "any-dest", 1})
			} else {
				assertEmptyMetrics(t, root.Snapshot())
			}
		})
	}
}

func TestAddApplicationHeaders(t *testing.T) {
	tests := map[string]struct {
		md                 metadata.MD
		h                  transport.Headers
		enforceHeaderRules bool
		expMD              metadata.MD
		expErr             error
		expReportHeader    bool
	}{
		"success": {
			md: metadata.Pairs("foo", "bar"),
			h: transport.HeadersFromMap(map[string]string{
				"baz": "qux",
			}),
			expMD: metadata.Pairs("foo", "bar", "baz", "qux"),
		},
		"reserved-rpc-prefix": {
			md: metadata.Pairs("foo", "bar"),
			h: transport.HeadersFromMap(map[string]string{
				"rpc-baz": "qux",
			}),
			expMD:           metadata.Pairs("foo", "bar"),
			expErr:          yarpcerrors.InvalidArgumentErrorf("cannot use reserved header in application headers: rpc-baz"),
			expReportHeader: false, // it's not a new behaviour
		},
		"reserved-dollar-rpc-prefix": {
			md: metadata.Pairs("foo", "bar"),
			h: transport.HeadersFromMap(map[string]string{
				"$rpc$-baz": "qux",
			}),
			expMD:           metadata.Pairs("foo", "bar", "$rpc$-baz", "qux"),
			expErr:          nil,
			expReportHeader: true,
		},
		"reserved-dollar-rpc-prefix-enforced-rule": {
			md: metadata.Pairs("foo", "bar"),
			h: transport.HeadersFromMap(map[string]string{
				"$rpc$-baz": "qux",
			}),
			enforceHeaderRules: true,
			expMD:              metadata.Pairs("foo", "bar"),
			expErr:             yarpcerrors.InternalErrorf("cannot use reserved header in application headers: $rpc$-baz"),
			expReportHeader:    true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRules)
			root := metrics.New()
			edgeMetrics := observability.NewReserveHeaderMetrics(root.Scope(), "grpc").With("any-source", "any-dest")

			err := addApplicationHeaders(tt.md, tt.h, edgeMetrics)
			assert.Equal(t, err, tt.expErr)
			assert.Equal(t, tt.expMD, tt.md)

			if tt.expReportHeader {
				assertTuple(t, root.Snapshot().Counters, tuple{"grpc_reserved_headers_error", "any-source", "any-dest", 1})
			} else {
				assertEmptyMetrics(t, root.Snapshot())
			}
		})

	}
}

func TestAddToMetadata(t *testing.T) {
	tests := map[string]struct {
		md     metadata.MD
		key    string
		value  string
		expErr error
		expMD  metadata.MD
	}{
		"nil-md": {
			md:    nil,
			key:   "foo",
			value: "bar",
			expMD: nil,
		},
		"empty-value-ignored": {
			md:    metadata.Pairs(),
			key:   "foo",
			value: "",
			expMD: metadata.Pairs(),
		},
		"duplicate-key": {
			md:     metadata.Pairs("foo", "bar"),
			key:    "foo",
			value:  "baz",
			expErr: yarpcerrors.InvalidArgumentErrorf("duplicate key: foo"),
			expMD:  metadata.Pairs("foo", "bar"),
		},
		"success": {
			md:    metadata.Pairs("foo", "bar"),
			key:   "baz",
			value: "qux",
			expMD: metadata.Pairs("foo", "bar", "baz", "qux"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := addToMetadata(tt.md, tt.key, tt.value)
			assert.Equal(t, err, tt.expErr)
			assert.Equal(t, tt.expMD, tt.md)
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
