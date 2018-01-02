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

package pally

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/pally/pallytest"
	"go.uber.org/yarpc/internal/testtime"
)

func TestGauge(t *testing.T) {
	r := NewRegistry(Labeled(Labels{"service": "users"}))
	gauge, err := r.NewGauge(Opts{
		Name:        "test_gauge",
		Help:        "Some help.",
		ConstLabels: Labels{"foo": "bar"},
	})
	require.NoError(t, err, "Unexpected error constructing gauge.")

	scope := newTestScope()
	stop, err := r.Push(scope, _tick)
	require.NoError(t, err, "Unexpected error starting Tally push.")

	gauge.Inc()
	gauge.Store(42)
	assert.Equal(t, int64(42), gauge.Load(), "Unexpected in-memory gauge value.")

	testtime.Sleep(5 * _tick)
	gauge.Store(4)
	assert.Equal(t, int64(4), gauge.Load(), "Unexpected in-memory gauge value after sleep.")

	stop()

	export := TallyExpectation{
		Type:   "gauge",
		Name:   "test_gauge",
		Labels: Labels{"foo": "bar", "service": "users"},
		Value:  4,
	}
	export.Test(t, scope)

	pallytest.AssertPrometheus(t, r, "# HELP test_gauge Some help.\n"+
		"# TYPE test_gauge gauge\n"+
		`test_gauge{foo="bar",service="users"} 4`)
}
