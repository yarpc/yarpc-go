package pally

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally"
)

// A Counter is a monotonically increasing value, like a car's odometer.
//
// Counters are exported to Prometheus as a snapshot of the current total, and
// to Tally as a diff since the last export. They implement
// prometheus.Collector, so they can also be registered directly with
// Prometheus Registries.
type Counter interface {
	prometheus.Collector

	Inc() int64
	Add(int64) int64
	Load() int64
}

// A CounterVector is a collection of Counters that share a name and some
// constant labels, but also have an enumerated set of variable labels. It
// implements prometheus.Collector, so it can also be registered directly with
// Prometheus Registries.
type CounterVector interface {
	prometheus.Collector

	Get(...string) (Counter, error)
	MustGet(...string) Counter
}

// A Gauge is a point-in-time measurement, like a car's speedometer.
//
// Gauges are exported to both Prometheus and Tally by simply reporting the
// current value. They implement prometheus.Collector, so they can also be
// registered directly with Prometheus registries.
type Gauge interface {
	prometheus.Collector

	Inc() int64
	Dec() int64
	Add(int64) int64
	Sub(int64) int64
	Store(int64)
	Load() int64
}

// A GaugeVector is a collection of Gauges that share a name and some constant
// labels, but also have an enumerated set of variable labels. It implements
// prometheus.Collector, so it can also be registered directly with Prometheus
// registries.
type GaugeVector interface {
	prometheus.Collector

	Get(...string) (Gauge, error)
	MustGet(...string) Gauge
}

type metric interface {
	prometheus.Collector

	push(tally.Scope)
}
