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
)

func TestNopCounter(t *testing.T) {
	assertNopCounter(t, NewNopCounter())
}

func TestNopCounterVector(t *testing.T) {
	vec := NewNopCounterVector()
	c, err := vec.Get("foo", "bar")
	require.NoError(t, err, "Failed Get from no-op CounterVector.")
	assert.NotPanics(t, func() { vec.MustGet("foo", "bar") }, "Failed MustGet from no-op CounterVector.")
	assertNopCounter(t, c)
}

func TestNopGauge(t *testing.T) {
	assertNopGauge(t, NewNopGauge())
}

func TestNopGaugeVector(t *testing.T) {
	vec := NewNopGaugeVector()
	g, err := vec.Get("foo", "bar")
	require.NoError(t, err, "Failed Get from no-op GaugeVector.")
	assert.NotPanics(t, func() { vec.MustGet("foo", "bar") }, "Failed MustGet from no-op GaugeVector.")
	assertNopGauge(t, g)
}

func TestNopLatencies(t *testing.T) {
	assertNopLatencies(t, NewNopLatencies())
}

func TestNopLatenciesVector(t *testing.T) {
	vec := NewNopLatenciesVector()
	lat, err := vec.Get("foo", "bar")
	require.NoError(t, err, "Failed Get from no-op LatenciesVector.")
	assert.NotPanics(t, func() { vec.MustGet("foo", "bar") }, "Failed MustGet from no-op LatenciesVector.")
	assertNopLatencies(t, lat)
}

func assertNopCounter(t testing.TB, c Counter) {
	assert.Equal(t, int64(0), c.Add(42), "Unexpected result from no-op Add.")
	assert.Equal(t, int64(0), c.Inc(), "Unexpected result from no-op Inc.")
	assert.Equal(t, int64(0), c.Load(), "Unexpected result from no-op Load.")
}

func assertNopGauge(t testing.TB, g Gauge) {
	g.Store(42)
	assert.Equal(t, int64(0), g.Add(42), "Unexpected result from no-op Add.")
	assert.Equal(t, int64(0), g.Sub(1), "Unexpected result from no-op Sub.")
	assert.Equal(t, int64(0), g.Inc(), "Unexpected result from no-op Inc.")
	assert.Equal(t, int64(0), g.Dec(), "Unexpected result from no-op Dec.")
	assert.Equal(t, int64(0), g.Load(), "Unexpected result from no-op Load.")
}

func assertNopLatencies(t testing.TB, lat Latencies) {
	assert.NotPanics(
		t,
		func() { NewNopLatencies().Observe(time.Second) },
		"Unexpected panic using no-op latencies.",
	)
}
