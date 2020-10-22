// Copyright (c) 2020 Uber Technologies, Inc.
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

package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
)

// CounterAssertion holds expected counter metric
type CounterAssertion struct {
	Name  string
	Tags  map[string]string
	Value int
}

// HistogramAssertion holds expected histogram metric
type HistogramAssertion struct {
	Name  string
	Tags  map[string]string
	Value []int64

	// Values are not compared, rather length of values is asserted
	IgnoreValueCompare bool
	ValueLength        int
}

// AssertCounters asserts expected counters with metrics snapshot
func AssertCounters(t *testing.T, counterAssertions []CounterAssertion, snapshot []metrics.Snapshot) {
	require.Len(t, counterAssertions, len(snapshot), snapshot)

	for i, wantCounter := range counterAssertions {
		require.Equal(t, wantCounter.Name, snapshot[i].Name, "unexpected counter %s", snapshot[i].Name)
		assert.EqualValues(t, wantCounter.Value, snapshot[i].Value, "unexpected counter value for %s", wantCounter.Name)
		for wantTagKey, wantTagVal := range wantCounter.Tags {
			assert.Equal(t, wantTagVal, snapshot[i].Tags[wantTagKey], "unexpected value for %q", wantTagKey)
		}
	}
}

// AssertHistograms asserts expected histograms with histogram snapshot
func AssertHistograms(t *testing.T, histogramAssertions []HistogramAssertion, snapshot []metrics.HistogramSnapshot) {
	require.Len(t, histogramAssertions, len(snapshot), "unexpected number of histograms")

	for i, wantHistogram := range histogramAssertions {
		require.Equal(t, wantHistogram.Name, snapshot[i].Name, "unexpected histogram %s", wantHistogram.Name)
		if wantHistogram.IgnoreValueCompare {
			assert.Equal(t, wantHistogram.ValueLength, len(snapshot[i].Values),
				"unexpected histogram value length for %s", wantHistogram.Name)
		} else {
			assert.EqualValues(t, wantHistogram.Value, snapshot[i].Values, "unexpected histogram value for %s", wantHistogram.Name)
		}
		for wantTagKey, wantTagVal := range wantHistogram.Tags {
			assert.Equal(t, wantTagVal, snapshot[i].Tags[wantTagKey], "unexpected value for %q", wantTagKey)
		}
	}
}

// AssertClientAndServerCounters asserts expected counters on client and server snapshots
func AssertClientAndServerCounters(t *testing.T, counterAssertions []CounterAssertion, clientSnapshot, serverSnapshot *metrics.Root) {
	t.Run("inbound", func(t *testing.T) {
		AssertCounters(t, counterAssertions, serverSnapshot.Snapshot().Counters)
	})
	t.Run("outbound", func(t *testing.T) {
		AssertCounters(t, counterAssertions, clientSnapshot.Snapshot().Counters)
	})
}

// AssertClientAndServerHistograms asserts expected histograms on client and server snapshots
func AssertClientAndServerHistograms(t *testing.T, histogramAssertions []HistogramAssertion,
	clientSnapshot, serverSnapshot *metrics.Root) {
	t.Run("inbound", func(t *testing.T) {
		AssertHistograms(t, histogramAssertions, serverSnapshot.Snapshot().Histograms)
	})
	t.Run("outbound", func(t *testing.T) {
		AssertHistograms(t, histogramAssertions, clientSnapshot.Snapshot().Histograms)
	})
}
