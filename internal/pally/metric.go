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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally"
)

// A Counter is a monotonically increasing value, like a car's odometer.
//
// Counters are exported to Prometheus as a snapshot of the current total, and
// to Tally as a diff since the last export.
type Counter interface {
	Inc() int64
	Add(int64) int64
	Load() int64
}

// A CounterVector is a collection of Counters that share a name and some
// constant labels, but also have an enumerated set of variable labels.
type CounterVector interface {
	// For a description of Get, MustGet, and vector types in general, see the
	// package-level documentation on vectors.
	Get(...string) (Counter, error)
	MustGet(...string) Counter
}

// A Gauge is a point-in-time measurement, like a car's speedometer.
//
// Gauges are exported to both Prometheus and Tally by simply reporting the
// current value.
type Gauge interface {
	Inc() int64
	Dec() int64
	Add(int64) int64
	Sub(int64) int64
	Store(int64)
	Load() int64
}

// A GaugeVector is a collection of Gauges that share a name and some constant
// labels, but also have an enumerated set of variable labels.
type GaugeVector interface {
	// For a description of Get, MustGet, and vector types in general, see the
	// package-level documentation on vectors.
	Get(...string) (Gauge, error)
	MustGet(...string) Gauge
}

// Latencies approximates a latency distribution with a histogram.
//
// Latencies are exported to both Prometheus and Tally using their native
// histogram types.
type Latencies interface {
	Observe(time.Duration)
}

// A LatenciesVector is a collection of Latencies that share a name and some
// constant labels, but also have an enumerated set of variable labels.
type LatenciesVector interface {
	// For a description of Get, MustGet, and vector types in general, see the
	// package-level documentation on vectors.
	Get(...string) (Latencies, error)
	MustGet(...string) Latencies
}

type metric interface {
	prometheus.Collector

	push(tally.Scope)
}
