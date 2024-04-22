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
	"go.uber.org/net/metrics"
)

type (
	// ReservedHeaderMetrics is a collection of metrics for reserved headers.
	ReservedHeaderMetrics struct {
		strippedVec *metrics.CounterVector
		errorVec    *metrics.CounterVector
	}

	// ReservedHeaderEdgeMetrics is a wrapper for ReservedHeaderMetrics that has source and dest.
	ReservedHeaderEdgeMetrics struct {
		m            *ReservedHeaderMetrics
		source, dest string
	}
)

// NewReserveHeaderMetrics creates a new ReservedHeaderMetrics.
func NewReserveHeaderMetrics(scope *metrics.Scope, transport string) *ReservedHeaderMetrics {
	return &ReservedHeaderMetrics{
		strippedVec: registerReservedHeaderStripped(scope, transport),
		errorVec:    registerReservedHeaderError(scope, transport),
	}
}

// IncStripped increments the stripped metric.
func (m *ReservedHeaderMetrics) IncStripped(source, dest string) {
	if m != nil {
		incHeaderVecMetric(m.strippedVec, source, dest)
	}
}

// IncError increments the error metric.
func (m *ReservedHeaderMetrics) IncError(source, dest string) {
	if m != nil {
		incHeaderVecMetric(m.errorVec, source, dest)
	}
}

// With returns a ReservedHeaderEdgeMetrics with source and dest.
func (m *ReservedHeaderMetrics) With(source, dest string) ReservedHeaderEdgeMetrics {
	return ReservedHeaderEdgeMetrics{
		m:      m,
		source: source,
		dest:   dest,
	}
}

// IncStripped increments the stripped metric.
func (m *ReservedHeaderEdgeMetrics) IncStripped() {
	m.m.IncStripped(m.source, m.dest)
}

// IncError increments the error metric.
func (m *ReservedHeaderEdgeMetrics) IncError() {
	m.m.IncError(m.source, m.dest)
}

func registerReservedHeaderStripped(scope *metrics.Scope, transport string) *metrics.CounterVector {
	v, _ := scope.CounterVector(metrics.Spec{
		Name:      transport + "_reserved_headers_stripped",
		Help:      "Total number of reserved headers being stripped.",
		ConstTags: map[string]string{"component": "yarpc-header-migration"},
		VarTags:   []string{"source", "dest"},
	})
	return v
}

func registerReservedHeaderError(scope *metrics.Scope, transport string) *metrics.CounterVector {
	v, _ := scope.CounterVector(metrics.Spec{
		Name:      transport + "_reserved_headers_error",
		Help:      "Total number of reserved headers led to error.",
		ConstTags: map[string]string{"component": "yarpc-header-migration"},
		VarTags:   []string{"source", "dest"},
	})
	return v
}

func incHeaderVecMetric(vector *metrics.CounterVector, source, dest string) {
	if vector != nil {
		if counter, err := vector.Get("source", source, "dest", dest); err == nil {
			counter.Inc()
		}
	}
}
