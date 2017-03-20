package pally_test

import (
	"go.uber.org/yarpc/internal/pally"

	"github.com/prometheus/client_golang/prometheus"
)

// For expvar-style usage (where all metrics are package-global), create a
// package-global pally.Registry and use the Must* constructors. This
// guarantees that all your metrics are unique.
var (
	_reg = pally.NewRegistry(
		// Also register all metrics with the package-global Prometheus
		// registry.
		pally.Federated(prometheus.DefaultRegisterer),
	)
	_watches = _reg.MustGauge(pally.Opts{
		Name: "watch_count",
		Help: "Current number of active service name watches.",
		ConstLabels: pally.Labels{
			"foo": "bar",
		},
	})
	_resolvesPerName = _reg.MustCounterVector(pally.Opts{
		Name: "resolve_count",
		Help: "Total name resolves by service.",
		ConstLabels: pally.Labels{
			"foo": "bar",
		},
		VariableLabels: []string{"service"},
	})
)

func Example_globalMetrics() {
	_watches.Store(42)
	_resolvesPerName.MustGet("some_service").Inc()
}
