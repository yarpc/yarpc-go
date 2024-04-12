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

package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestHandleWithReservedField(t *testing.T) {
	root := metrics.New()
	meter := root.Scope()

	loggerCore, loggerObs := observer.New(zap.ErrorLevel)
	logger := zap.New(loggerCore)

	mw := NewMiddleware(Config{Scope: meter, Logger: logger, MetricTagsBlocklist: []string{"routing_delegate"}})

	for _, rd := range []string{"rd1", "rd2"} {
		req := &transport.Request{
			Caller:          "caller",
			Service:         "service",
			Transport:       "",
			Encoding:        "raw",
			Procedure:       "procedure",
			ShardKey:        "sk",
			RoutingDelegate: rd,
		}
		// if "routing_delegate" is part of metricTagsBlockMap
		// multiple requests with all other same field value but RoutingDelegate
		// should successfully create a single metrics edge, without any error logging.
		assert.NoError(t, mw.Handle(context.Background(), req, &transporttest.FakeResponseWriter{}, &fakeHandler{}))
		assert.Len(t, loggerObs.All(), 0)
	}
}

func TestMetricsTagIgnore(t *testing.T) {
	req := &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Transport:       "",
		Encoding:        "proto",
		Procedure:       "procedure",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
	}

	tests := []struct {
		desc            string
		metricsToIgnore []string
		expected        *metricsTagIgnore
		expectedTags    metrics.Tags
	}{
		{
			desc:     "empty ignore list",
			expected: &metricsTagIgnore{}, // all fields default to false
			expectedTags: metrics.Tags{
				_source:          "caller",
				_dest:            "service",
				_transport:       "unknown",
				_procedure:       "procedure",
				_encoding:        "proto",
				_routingKey:      "rk",
				_routingDelegate: "rd",
				_direction:       "inbound",
				_rpcType:         "Unary",
			},
		},
		{
			desc:            "only reserved fields in ignore list",
			metricsToIgnore: []string{_source, _dest, _transport, _procedure, _encoding, _routingKey, _routingDelegate, _direction, _rpcType},
			expected: &metricsTagIgnore{
				source:          true,
				dest:            true,
				transport:       true,
				procedure:       true,
				encoding:        true,
				routingKey:      true,
				routingDelegate: true,
				direction:       true,
				rpcType:         true,
			},
			expectedTags: metrics.Tags{
				_source:          "__dropped__",
				_dest:            "__dropped__",
				_transport:       "__dropped__",
				_procedure:       "__dropped__",
				_encoding:        "__dropped__",
				_routingKey:      "__dropped__",
				_routingDelegate: "__dropped__",
				_direction:       "__dropped__",
				_rpcType:         "__dropped__",
			},
		},
		{
			desc:            "reserved fields and other fields in ignore list",
			metricsToIgnore: []string{_source, _transport, _rpcType, "user_defined1", "user_defined2"},
			expected: &metricsTagIgnore{
				source:    true,
				transport: true,
				rpcType:   true,
			},
			expectedTags: metrics.Tags{
				_source:          "__dropped__",
				_dest:            "service",
				_transport:       "__dropped__",
				_procedure:       "procedure",
				_encoding:        "proto",
				_routingKey:      "rk",
				_routingDelegate: "rd",
				_direction:       "inbound",
				_rpcType:         "__dropped__",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			actual := newMetricsTagIgnore(tt.metricsToIgnore)
			assert.Equal(t, tt.expected, actual, "wrong metricsToIgnore")
			actualTags := actual.tags(req, "inbound", transport.Unary)
			assert.Equal(t, tt.expectedTags, actualTags, "tags mismatch")
		})
	}
}

func TestEdgeNopFallbacks(t *testing.T) {
	// If we fail to create any of the metrics required for the edge, we should
	// fall back to no-op implementations. The easiest way to trigger failures
	// is to re-use the same Registry.
	root := metrics.New()
	meter := root.Scope()
	req := &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Transport:       "",
		Encoding:        "raw",
		Procedure:       "procedure",
		ShardKey:        "sk",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
	}

	// Should succeed, covered by middleware tests.
	_ = newEdge(zap.NewNop(), meter, &metricsTagIgnore{}, req, string(_directionOutbound), transport.Unary)

	// Should fall back to no-op metrics.
	// Usage of nil metrics should not panic, should not observe changes.
	e := newEdge(zap.NewNop(), meter, &metricsTagIgnore{}, req, string(_directionOutbound), transport.Unary)

	e.calls.Inc()
	assert.Equal(t, int64(0), e.calls.Load(), "Expected to fall back to no-op metrics.")

	e.successes.Load()
	assert.Equal(t, int64(0), e.successes.Load(), "Expected to fall back to no-op metrics.")

	cf, err := e.callerFailures.Get()
	assert.NoError(t, err, "Unexpected error getting caller failure counter")
	cf.Inc()
	assert.Equal(t, int64(0), cf.Load(), "Expected to fall back to no-op metrics.")

	sf, err := e.serverFailures.Get()
	assert.NoError(t, err, "Unexpected error getting server failure counter")
	sf.Inc()
	assert.Equal(t, int64(0), sf.Load(), "Expected to fall back to no-op metrics.")

	e.latencies.Observe(0)
	e.callerErrLatencies.Observe(0)
	e.serverErrLatencies.Observe(0)
}

func TestUnknownIfEmpty(t *testing.T) {
	tests := []struct {
		transport string
		expected  string
	}{
		{
			expected: "unknown",
		},
		{
			transport: "tchannel",
			expected:  "tchannel",
		},
		{
			transport: "http",
			expected:  "http",
		},
		{
			transport: "grpc",
			expected:  "grpc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			actual := unknownIfEmpty(tt.transport)
			assert.Equal(t, tt.expected, actual, "expected: %s, got: %s", tt.transport, actual)

		})
	}
}
