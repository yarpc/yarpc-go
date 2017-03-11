package pally

import (
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

type metric interface {
	prometheus.Collector

	push(tally.Scope)
}
