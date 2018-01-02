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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/pally/pallytest"
)

func TestSimpleVectors(t *testing.T) {
	tests := []struct {
		desc      string
		opts      Opts
		f         func(testing.TB, *Registry, Opts)
		wantTally TallyExpectation
		wantProm  string
	}{
		{
			desc: "valid counter labels",
			opts: Opts{
				Name:           "test_counter",
				Help:           "Some help.",
				VariableLabels: []string{"var"},
			},
			f: func(t testing.TB, r *Registry, opts Opts) {
				vec, err := r.NewCounterVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				counter, err := vec.Get("x")
				require.NoError(t, err, "Unexpected error getting a counter with correct number of labels.")
				counter.Inc()
				vec.MustGet("x").Add(2)
				assert.Equal(t, int64(3), counter.Load(), "Unexpected in-memory metric value.")
			},
			wantTally: TallyExpectation{
				Type:   "counter",
				Name:   "test_counter",
				Labels: Labels{"var": "x"},
				Value:  3,
			},
			wantProm: "# HELP test_counter Some help.\n" +
				"# TYPE test_counter counter\n" +
				`test_counter{var="x"} 3`,
		},
		{
			desc: "invalid counter labels",
			opts: Opts{
				Name:           "test_counter",
				Help:           "Some help.",
				VariableLabels: []string{"var"},
			},
			f: func(t testing.TB, r *Registry, opts Opts) {
				vec, err := r.NewCounterVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				counter, err := vec.Get("x!")
				require.NoError(t, err, "Unexpected error getting a counter with correct number of labels.")
				counter.Inc()
				vec.MustGet("x!").Add(2)
				assert.Equal(t, int64(3), counter.Load(), "Unexpected in-memory metric value.")
			},
			wantTally: TallyExpectation{
				Type:   "counter",
				Name:   "test_counter",
				Labels: Labels{"var": "x-"},
				Value:  3,
			},
			wantProm: "# HELP test_counter Some help.\n" +
				"# TYPE test_counter counter\n" +
				`test_counter{var="x-"} 3`,
		},
		{
			desc: "wrong number of counter labels",
			opts: Opts{
				Name:           "test_counter",
				Help:           "Some help.",
				VariableLabels: []string{"var"},
			},
			f: func(t testing.TB, r *Registry, opts Opts) {
				vec, err := r.NewCounterVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				_, err = vec.Get("x", "y")
				require.Error(t, err, "Expected an error getting a counter with incorrect number of labels.")
				require.Panics(t, func() { vec.MustGet("x", "y") }, "Expected panic getting a counter with incorrect number of labels.")
			},
		},
		{
			desc: "valid gauge labels",
			opts: Opts{
				Name:           "test_gauge",
				Help:           "Some help.",
				VariableLabels: []string{"var"},
			},
			f: func(t testing.TB, r *Registry, opts Opts) {
				vec, err := r.NewGaugeVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				gauge, err := vec.Get("x")
				require.NoError(t, err, "Unexpected error getting a gauge with correct number of labels.")
				gauge.Inc()
				vec.MustGet("x").Store(2)
				assert.Equal(t, int64(2), gauge.Load(), "Unexpected in-memory metric value.")
			},
			wantTally: TallyExpectation{
				Type:   "gauge",
				Name:   "test_gauge",
				Labels: Labels{"var": "x"},
				Value:  2,
			},
			wantProm: "# HELP test_gauge Some help.\n" +
				"# TYPE test_gauge gauge\n" +
				`test_gauge{var="x"} 2`,
		},
		{
			desc: "invalid gauge labels",
			opts: Opts{
				Name:           "test_gauge",
				Help:           "Some help.",
				VariableLabels: []string{"var"},
			},
			f: func(t testing.TB, r *Registry, opts Opts) {
				vec, err := r.NewGaugeVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				gauge, err := vec.Get("x!")
				require.NoError(t, err, "Unexpected error getting a gauge with correct number of labels.")
				gauge.Inc()
				vec.MustGet("x!").Store(2)
				assert.Equal(t, int64(2), gauge.Load(), "Unexpected in-memory metric value.")
			},
			wantTally: TallyExpectation{
				Type:   "gauge",
				Name:   "test_gauge",
				Labels: Labels{"var": "x-"},
				Value:  2,
			},
			wantProm: "# HELP test_gauge Some help.\n" +
				"# TYPE test_gauge gauge\n" +
				`test_gauge{var="x-"} 2`,
		},
		{
			desc: "wrong number of gauge labels",
			opts: Opts{
				Name:           "test_gauge",
				Help:           "Some help.",
				VariableLabels: []string{"var"},
			},
			f: func(t testing.TB, r *Registry, opts Opts) {
				vec, err := r.NewGaugeVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				_, err = vec.Get("x", "y")
				require.Error(t, err, "Expected an error getting a gauge with incorrect number of labels.")
				require.Panics(t, func() { vec.MustGet("x", "y") }, "Expected panic getting a gauge with incorrect number of labels.")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			r := NewRegistry()

			scope := newTestScope()
			stop, err := r.Push(scope, _tick)
			require.NoError(t, err, "Unexpected error starting Tally push.")
			defer stop()

			tt.f(t, r, tt.opts)

			time.Sleep(10 * _tick)
			tt.wantTally.Test(t, scope)

			pallytest.AssertPrometheus(t, r, tt.wantProm)
		})
	}
}
