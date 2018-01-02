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
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/pally/pallytest"
)

func TestLatencies(t *testing.T) {
	r := NewRegistry(Labeled(Labels{"service": "users"}))
	lat, err := r.NewLatencies(LatencyOpts{
		Opts: Opts{
			Name:        "test_latency_ns",
			Help:        "Some help.",
			ConstLabels: Labels{"foo": "bar"},
		},
		Unit:    time.Nanosecond,
		Buckets: []time.Duration{10, 50, 100},
	})
	require.NoError(t, err, "Unexpected error constructing counter.")

	scope := newTestScope()
	stop, err := r.Push(scope, _tick)
	require.NoError(t, err, "Unexpected error starting Tally push.")

	lat.Observe(-1)
	lat.Observe(0)
	lat.Observe(10)
	lat.Observe(75)
	lat.Observe(150)

	time.Sleep(5 * _tick)
	stop()

	export := TallyExpectation{
		Type:   "latencies",
		Name:   "test_latency_ns",
		Labels: Labels{"foo": "bar", "service": "users"},
		Durations: map[time.Duration]int64{
			10:  3,
			50:  0,
			100: 1,
			time.Duration(math.MaxInt64): 1,
		},
	}
	export.Test(t, scope)

	pallytest.AssertPrometheus(t, r, "# HELP test_latency_ns Some help.\n"+
		"# TYPE test_latency_ns histogram\n"+
		`test_latency_ns_bucket{foo="bar",service="users",le="10"} 3`+"\n"+
		`test_latency_ns_bucket{foo="bar",service="users",le="50"} 3`+"\n"+
		`test_latency_ns_bucket{foo="bar",service="users",le="100"} 4`+"\n"+
		`test_latency_ns_bucket{foo="bar",service="users",le="+Inf"} 5`+"\n"+
		`test_latency_ns_sum{foo="bar",service="users"} 234`+"\n"+
		`test_latency_ns_count{foo="bar",service="users"} 5`)
}

func TestLatenciesVector(t *testing.T) {
	tests := []struct {
		desc      string
		opts      LatencyOpts
		f         func(testing.TB, *Registry, LatencyOpts)
		wantTally TallyExpectation
		wantProm  string
	}{
		{
			desc: "valid labels",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "test_latency_ms",
					Help:           "Some help.",
					VariableLabels: []string{"var"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			f: func(t testing.TB, r *Registry, opts LatencyOpts) {
				vec, err := r.NewLatenciesVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				lat, err := vec.Get("x")
				require.NoError(t, err, "Unexpected error getting a counter with correct number of labels.")
				lat.Observe(time.Millisecond)
				vec.MustGet("x").Observe(time.Millisecond)
			},
			wantTally: TallyExpectation{
				Type:   "latencies",
				Name:   "test_latency_ms",
				Labels: Labels{"var": "x"},
				Durations: map[time.Duration]int64{
					time.Second:                  2,
					time.Minute:                  0,
					time.Duration(math.MaxInt64): 0,
				},
			},
			wantProm: "# HELP test_latency_ms Some help.\n" +
				"# TYPE test_latency_ms histogram\n" +
				`test_latency_ms_bucket{var="x",le="1000"} 2` + "\n" +
				`test_latency_ms_bucket{var="x",le="60000"} 2` + "\n" +
				`test_latency_ms_bucket{var="x",le="+Inf"} 2` + "\n" +
				`test_latency_ms_sum{var="x"} 2` + "\n" +
				`test_latency_ms_count{var="x"} 2`,
		},
		{
			desc: "invalid labels",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "test_latency_ms",
					Help:           "Some help.",
					VariableLabels: []string{"var"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			f: func(t testing.TB, r *Registry, opts LatencyOpts) {
				vec, err := r.NewLatenciesVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				lat, err := vec.Get("x!")
				require.NoError(t, err, "Unexpected error getting a counter with correct number of labels.")
				lat.Observe(time.Millisecond)
				vec.MustGet("x!").Observe(time.Millisecond)
			},
			wantTally: TallyExpectation{
				Type:   "latencies",
				Name:   "test_latency_ms",
				Labels: Labels{"var": "x-"},
				Durations: map[time.Duration]int64{
					time.Second:                  2,
					time.Minute:                  0,
					time.Duration(math.MaxInt64): 0,
				},
			},
			wantProm: "# HELP test_latency_ms Some help.\n" +
				"# TYPE test_latency_ms histogram\n" +
				`test_latency_ms_bucket{var="x-",le="1000"} 2` + "\n" +
				`test_latency_ms_bucket{var="x-",le="60000"} 2` + "\n" +
				`test_latency_ms_bucket{var="x-",le="+Inf"} 2` + "\n" +
				`test_latency_ms_sum{var="x-"} 2` + "\n" +
				`test_latency_ms_count{var="x-"} 2`,
		},
		{
			desc: "wrong number of label values",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "test_latency_ms",
					Help:           "Some help.",
					VariableLabels: []string{"var"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			f: func(t testing.TB, r *Registry, opts LatencyOpts) {
				vec, err := r.NewLatenciesVector(opts)
				require.NoError(t, err, "Unexpected error constructing vector.")
				_, err = vec.Get("x", "y")
				require.Error(t, err, "Unexpected success calling Get with incorrect number of labels.")
				require.Panics(
					t,
					func() { vec.MustGet("x", "y").Observe(time.Millisecond) },
					"Expected a panic using MustGet with the wrong number of labels.",
				)
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

			time.Sleep(5 * _tick)
			tt.wantTally.Test(t, scope)

			pallytest.AssertPrometheus(t, r, tt.wantProm)
		})

	}

}

func TestLatenciesVectorIndependence(t *testing.T) {
	// Ensure that we're not erroneously sharing state across histograms in a
	// vector.
	r := NewRegistry()

	opts := LatencyOpts{
		Opts: Opts{
			Name:           "test_latency_ms",
			Help:           "Some help.",
			VariableLabels: []string{"var"},
		},
		Unit:    time.Millisecond,
		Buckets: []time.Duration{time.Second},
	}
	vec, err := r.NewLatenciesVector(opts)
	require.NoError(t, err, "Unexpected error constructing vector.")

	x, err := vec.Get("x")
	require.NoError(t, err, "Unexpected error calling Get.")

	y, err := vec.Get("y")
	require.NoError(t, err, "Unexpected error calling Get.")

	x.Observe(time.Millisecond)
	y.Observe(time.Millisecond)

	pallytest.AssertPrometheus(t, r, "# HELP test_latency_ms Some help.\n"+
		"# TYPE test_latency_ms histogram\n"+
		`test_latency_ms_bucket{var="x",le="1000"} 1`+"\n"+
		`test_latency_ms_bucket{var="x",le="+Inf"} 1`+"\n"+
		`test_latency_ms_sum{var="x"} 1`+"\n"+
		`test_latency_ms_count{var="x"} 1`+"\n"+
		`test_latency_ms_bucket{var="y",le="1000"} 1`+"\n"+
		`test_latency_ms_bucket{var="y",le="+Inf"} 1`+"\n"+
		`test_latency_ms_sum{var="y"} 1`+"\n"+
		`test_latency_ms_count{var="y"} 1`)
}
