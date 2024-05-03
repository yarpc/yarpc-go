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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/net/metrics"
)

func TestReservedHeaderMetrics(t *testing.T) {
	t.Run("nil-scope", func(t *testing.T) {
		assert.NotPanics(t, func() {
			m := NewReserveHeaderMetrics(nil, "test")

			m.IncStripped("", "")
			m.IncError("", "")
		})
	})

	t.Run("empty-source-dest", func(t *testing.T) {
		root := metrics.New()
		m := NewReserveHeaderMetrics(root.Scope(), "test")

		m.IncStripped("", "")
		m.IncError("", "")

		assertTuples(t, root.Snapshot().Counters, []tuple{
			{"test_reserved_headers_stripped", "default", "default", 1},
			{"test_reserved_headers_error", "default", "default", 1},
		})
	})

	t.Run("inc-header-metric", func(t *testing.T) {
		root := metrics.New()
		m := NewReserveHeaderMetrics(root.Scope(), "test")

		m.IncStripped("source", "dest")
		m.IncError("source", "dest")

		m.IncStripped("source", "dest")
		m.IncStripped("source", "dest-2")
		m.IncStripped("source-2", "dest-2")

		assertTuples(t, root.Snapshot().Counters, []tuple{
			{"test_reserved_headers_stripped", "source", "dest", 2},
			{"test_reserved_headers_stripped", "source", "dest-2", 1},
			{"test_reserved_headers_stripped", "source-2", "dest-2", 1},
			{"test_reserved_headers_error", "source", "dest", 1},
		})
	})
}

func TestReservedHeaderMetricsWith(t *testing.T) {
	root := metrics.New()
	m := NewReserveHeaderMetrics(root.Scope(), "test")

	e := m.With("source", "dest")
	assert.Same(t, m, e.m)
	assert.Equal(t, "source", e.source)
	assert.Equal(t, "dest", e.dest)
}

func TestEmptyEdgeMetrics(t *testing.T) {
	assert.NotPanics(t, func() {
		e := ReservedHeaderEdgeMetrics{}
		e.IncStripped()
		e.IncError()
	})
}

func TestReservedHeaderEdgeMetricsIncs(t *testing.T) {
	root := metrics.New()
	m := NewReserveHeaderMetrics(root.Scope(), "test")

	e1 := m.With("source", "dest")

	e1.IncStripped()
	e1.IncError()
	e1.IncError()

	assertTuples(t, root.Snapshot().Counters, []tuple{
		{"test_reserved_headers_stripped", "source", "dest", 1},
		{"test_reserved_headers_error", "source", "dest", 2},
	})

	e2 := m.With("source", "dest-2")
	e2.IncStripped()
	e2.IncStripped()
	e2.IncError()

	assertTuples(t, root.Snapshot().Counters, []tuple{
		{"test_reserved_headers_stripped", "source", "dest", 1},
		{"test_reserved_headers_stripped", "source", "dest-2", 2},
		{"test_reserved_headers_error", "source", "dest", 2},
		{"test_reserved_headers_error", "source", "dest-2", 1},
	})
}

type tuple struct {
	name, tag1, tag2 string
	value            int64
}

func assertTuples(t *testing.T, snapshots []metrics.Snapshot, expected []tuple) {
	actual := make([]tuple, 0, len(snapshots))

	for _, c := range snapshots {
		actual = append(actual, tuple{c.Name, c.Tags["source"], c.Tags["dest"], c.Value})
	}

	assert.ElementsMatch(t, expected, actual)
}
