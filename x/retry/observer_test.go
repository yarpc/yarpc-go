// Copyright (c) 2017 Uber Technologies, Inc.
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

package retry

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

func TestNopEdge(t *testing.T) {
	// If we fail to create any of the metrics required for the edge, we should
	// fall back to no-op implementations. The easiest way to trigger failures
	// is to re-use the same Registry.
	reg, _ := newRegistry(zap.NewNop(), nil)
	req := &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Encoding:        "raw",
		Procedure:       "procedure",
		ShardKey:        "sk",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
		Body:            strings.NewReader("body"),
	}

	// Should succeed, covered by middleware tests.
	_ = newEdge(zap.NewNop(), reg, req)

	// Should fall back to no-op metrics.
	e := newEdge(zap.NewNop(), reg, req)
	assert.NotNil(t, e.attempts, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.successes, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.retriesAfterError, "Expected to fall back to no-op metrics.")
	assert.NotNil(t, e.failures, "Expected to fall back to no-op metrics.")
}

func TestYarpcInternalErrorCounter(t *testing.T) {
	testScope := tally.NewTestScope("", map[string]string{})
	graph, _ := newObserverGraph(zap.NewNop(), testScope)

	call := graph.begin(&transport.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  "raw",
		Procedure: "procedure",
	})

	call.yarpcInternalError(yarpcerrors.InternalErrorf("test"))

	assert.Equal(t, int64(1), call.e.failures.MustGet(_yarpcInternal, yarpcerrors.CodeInternal.String()).Load())
}

func TestErrorName(t *testing.T) {
	type testStruct struct {
		msg      string
		giveErr  error
		wantName string
	}
	tests := []testStruct{
		{
			msg:      "internal",
			giveErr:  yarpcerrors.InternalErrorf("test"),
			wantName: yarpcerrors.CodeInternal.String(),
		},
		{
			msg:      "invalid request",
			giveErr:  yarpcerrors.InvalidArgumentErrorf("test"),
			wantName: yarpcerrors.CodeInvalidArgument.String(),
		},
		{
			msg:      "yarpc unknown",
			giveErr:  errors.New("unknown"),
			wantName: "unknown_internal_yarpc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			assert.Equal(t, tt.wantName, getErrorName(tt.giveErr))
		})
	}
}
