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
	"sync"

	"go.uber.org/net/metrics"
)

var (
	reservedHeaderStripped *metrics.CounterVector
	reservedHeaderError    *metrics.CounterVector

	registerHeaderMetricsOnce sync.Once
)

// IncReservedHeaderStripped increments the counter for reserved headers being stripped.
func IncReservedHeaderStripped(m *metrics.Scope, source, dest string) {
	registerHeaderMetrics(m)
	incHeaderMetric(reservedHeaderStripped, source, dest)
}

// IncReservedHeaderError increments the counter for reserved headers led to error.
func IncReservedHeaderError(m *metrics.Scope, source, dest string) {
	registerHeaderMetrics(m)
	incHeaderMetric(reservedHeaderError, source, dest)
}

func registerHeaderMetrics(m *metrics.Scope) {
	if m == nil {
		return
	}

	registerHeaderMetricsOnce.Do(func() {
		reservedHeaderStripped, _ = m.CounterVector(metrics.Spec{
			Name:      "reserved_headers_stripped",
			Help:      "Total number of reserved headers being stripped.",
			ConstTags: map[string]string{"component": "yarpc-header-migration"},
			VarTags:   []string{"source", "dest"},
		})

		reservedHeaderError, _ = m.CounterVector(metrics.Spec{
			Name:      "reserved_headers_error",
			Help:      "Total number of reserved headers led to error.",
			ConstTags: map[string]string{"component": "yarpc-header-migration"},
			VarTags:   []string{"source", "dest"},
		})
	})
}

func incHeaderMetric(vector *metrics.CounterVector, source, dest string) {
	if vector != nil {
		if counter, err := vector.Get("source", source, "dest", dest); counter != nil && err == nil {
			counter.Inc()
		}
	}
}
