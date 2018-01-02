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

package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/pally"
	"go.uber.org/zap"
)

func TestEdgeNopFallbacks(t *testing.T) {
	// If we fail to create any of the metrics required for the edge, we should
	// fall back to no-op implementations. The easiest way to trigger failures
	// is to re-use the same Registry.
	reg := pally.NewRegistry()
	req := &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Encoding:        "raw",
		Procedure:       "procedure",
		ShardKey:        "sk",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
	}

	// Should succeed, covered by middleware tests.
	_ = newEdge(zap.NewNop(), reg, req, string(_directionOutbound))

	// Should fall back to no-op metrics.
	e := newEdge(zap.NewNop(), reg, req, string(_directionOutbound))
	assert.NotNil(t, e.calls, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.successes, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.callerFailures, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.serverFailures, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.latencies, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.callerErrLatencies, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.serverErrLatencies, "Expected to fall back to no-op metrics.")
}
