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
	"github.com/uber-go/tally"
)

const _tick = 10 * time.Millisecond

func newTestScope() tally.TestScope {
	return tally.NewTestScope("" /* prefix */, nil /* tags */)
}

type TallyExpectation struct {
	Type      string
	Value     int64
	Durations map[time.Duration]int64
	Name      string
	Labels    Labels
}

// Test checks that the specified metric is the only metric exported to
// Tally.
func (exp TallyExpectation) Test(t testing.TB, scope tally.TestScope) {
	snap := scope.Snapshot()
	if exp.Name == "" {
		exp.assertEmpty(t, snap)
	} else if exp.Type == "counter" {
		exp.assertOnlyCounter(t, snap)
	} else if exp.Type == "gauge" {
		exp.assertOnlyGauge(t, snap)
	} else if exp.Type == "latencies" {
		exp.assertOnlyLatencies(t, snap)
	} else {
		t.Fatalf("Can't make Tally assertions about type %q.", exp.Type)
	}
}

func (exp TallyExpectation) assertOnlyCounter(t testing.TB, snap tally.Snapshot) {
	exp.assertNoGauges(t, snap)
	exp.assertNoTimers(t, snap)
	exp.assertNoLatencies(t, snap)

	counters := snap.Counters()
	require.Equal(t, 1, len(counters), "Expected exactly one counter in Tally snapshot.")
	key := tally.KeyForPrefixedStringMap(exp.Name, exp.Labels)
	c, ok := counters[key]
	require.True(t, ok, "Didn't find Tally counter with key %q.", key)
	assert.Equal(t, exp.Name, c.Name(), "Tally counter has an unexpected name.")
	assert.Equal(t, map[string]string(exp.Labels), c.Tags(), "Tally counter has unexpected tags.")
	assert.Equal(t, exp.Value, c.Value(), "Tally counter has unexpected value.")
}

func (exp TallyExpectation) assertOnlyGauge(t testing.TB, snap tally.Snapshot) {
	exp.assertNoCounters(t, snap)
	exp.assertNoTimers(t, snap)
	exp.assertNoLatencies(t, snap)

	gauges := snap.Gauges()
	require.Equal(t, 1, len(gauges), "Expected exactly one gauge in Tally snapshot.")
	key := tally.KeyForPrefixedStringMap(exp.Name, exp.Labels)
	g, ok := gauges[key]
	require.True(t, ok, "Didn't find Tally gauge with key %q.", key)
	assert.Equal(t, exp.Name, g.Name(), "Tally gauge has an unexpected name.")
	assert.Equal(t, map[string]string(exp.Labels), g.Tags(), "Tally gauge has unexpected tags.")
	assert.Equal(t, exp.Value, int64(g.Value()), "Tally gauge has unexpected value.")
}

func (exp TallyExpectation) assertOnlyLatencies(t testing.TB, snap tally.Snapshot) {
	exp.assertNoCounters(t, snap)
	exp.assertNoGauges(t, snap)
	exp.assertNoTimers(t, snap)

	histograms := snap.Histograms()
	require.Equal(t, 1, len(histograms), "Expected exactly one histogram in Tally snapshot.")
	key := tally.KeyForPrefixedStringMap(exp.Name, exp.Labels)
	h, ok := histograms[key]
	require.True(t, ok, "Didn't find Tally histogram with key %q.", key)
	assert.Equal(t, exp.Name, h.Name(), "Tally histogram has an unexpected name.")
	assert.Equal(t, map[string]string(exp.Labels), h.Tags(), "Tally histogram has unexpected tags.")
	assert.Equal(t, exp.Durations, h.Durations(), "Tally histogram has unexpected observed durations.")
}

func (exp TallyExpectation) assertEmpty(t testing.TB, snap tally.Snapshot) {
	exp.assertNoGauges(t, snap)
	exp.assertNoCounters(t, snap)
	exp.assertNoTimers(t, snap)
	exp.assertNoLatencies(t, snap)
}

func (exp TallyExpectation) assertNoGauges(t testing.TB, snap tally.Snapshot) {
	assert.Equal(t, 0, len(snap.Gauges()), "Unexpected gauges in exported Tally data.")
}

func (exp TallyExpectation) assertNoCounters(t testing.TB, snap tally.Snapshot) {
	assert.Equal(t, 0, len(snap.Counters()), "Unexpected counters in exported Tally data.")
}

func (exp TallyExpectation) assertNoTimers(t testing.TB, snap tally.Snapshot) {
	assert.Equal(t, 0, len(snap.Timers()), "Unexpected timers in exported Tally data.")
}

func (exp TallyExpectation) assertNoLatencies(t testing.TB, snap tally.Snapshot) {
	assert.Equal(t, 0, len(snap.Histograms()), "Unexpected latency histograms in Tally data.")
}
