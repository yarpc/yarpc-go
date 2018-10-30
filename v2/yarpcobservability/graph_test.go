// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcobservability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

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
	_ = newEdge(zap.NewNop(), meter, req, string(_directionOutbound))

	// Should fall back to no-op metrics.
	// Usage of nil metrics should not panic, should not observe changes.
	e := newEdge(zap.NewNop(), meter, req, string(_directionOutbound))

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
