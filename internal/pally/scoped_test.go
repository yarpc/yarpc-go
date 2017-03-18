package pally_test

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/yarpc/internal/pally"

	"github.com/uber-go/tally"
)

// If you'd prefer to use pure dependency injection and scope your metrics
// to a single struct, create a new pally.Registry in your struct's
// constructor. In this case, we're also exporting our metrics to a Tally
// scope, which can report to StatsD- or M3-aware systems.
type Resolver struct {
	registry        *pally.Registry
	watches         pally.Gauge
	resolves        pally.CounterVector
	stopTallyExport context.CancelFunc
}

func NewResolver(scope tally.Scope) (*Resolver, error) {
	reg := pally.NewRegistry()
	stop, err := reg.Push(scope, time.Second)
	if err != nil {
		return nil, err
	}

	watches, err := _reg.NewGauge(pally.Opts{
		Name: "watch_count",
		Help: "Current number of active service name watches.",
		ConstLabels: pally.Labels{
			"foo": "bar",
		},
	})
	if err != nil {
		return nil, err
	}

	resolves, err := _reg.NewCounterVector(pally.Opts{
		Name: "resolve_count",
		Help: "Total name resolves by path.",
		ConstLabels: pally.Labels{
			"foo": "bar",
		},
		VariableLabels: []string{"service"},
	})
	if err != nil {
		return nil, err
	}

	return &Resolver{
		registry:        reg,
		watches:         watches,
		resolves:        resolves,
		stopTallyExport: stop,
	}, nil
}

func (r *Resolver) Watch() {
	r.watches.Inc()
}

func (r *Resolver) Resolve(name string) {
	if c, err := r.resolves.Get(name); err == nil {
		c.Inc()
	}
}

func (r *Resolver) Close() {
	r.stopTallyExport()
}

func (r *Resolver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Our registry can report its own metrics via a Prometheus-compatible HTTP
	// handler.
	r.registry.ServeHTTP(w, req)
}

func Example_dependencyInjection() {
	scope := tally.NewTestScope("testing", nil /* labels */)
	reg, err := NewResolver(scope)
	if err != nil {
		panic(err.Error())
	}
	reg.Watch()
	reg.Resolve("some_service")
}
