package pally

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/uber-go/tally"
	"go.uber.org/atomic"
)

// A Registry is a collection of metrics, usually scoped to a single package or
// object. Each Registry is also its own http.Handler, and can serve
// Prometheus-flavored text and protocol buffer pages for metrics introspection
// or scraping.
type Registry struct {
	metricsMu   sync.RWMutex
	metrics     []metric
	constLabels Labels

	// TODO: To serve our own Prometheus endpoints, we only need to implement
	// prometheus.Gatherer. The vanilla Prometheus registry is a convenient
	// implementation, but it enforces metric uniqueness via FNV64a; if we run
	// into hash collisions in our more-demanding applications, we should use a
	// simple map[string]struct{} instead.
	prom      *prometheus.Registry
	federated []prometheus.Registerer
	handler   http.Handler

	pushing atomic.Bool
	scope   tally.Scope
	ticker  *time.Ticker
	stop    chan struct{}
	stopped chan struct{}
}

// A RegistryOption configures a Registry.
type RegistryOption func(*Registry)

// Federated links a pally.Registry with a prometheus.Registerer, so that all
// metrics created in one also appear in the other.
func Federated(prom prometheus.Registerer) RegistryOption {
	return func(r *Registry) {
		r.federated = append(r.federated, prom)
	}
}

// Labeled adds constant labels to a Registry. All metrics created by a
// Registry inherit its constant labels.
func Labeled(ls Labels) RegistryOption {
	return func(r *Registry) {
		for k, v := range ls {
			if !model.LabelName(k).IsValid() || !model.LabelValue(v).IsValid() {
				continue
			}
			if !isValidTallyString(k) || !isValidTallyString(v) {
				continue
			}
			r.constLabels[k] = v
		}
	}
}

// NewRegistry constructs a new Registry.
func NewRegistry(opts ...RegistryOption) *Registry {
	prom := prometheus.NewRegistry()
	handler := promhttp.HandlerFor(prom, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError, // 500 on errors
	})

	r := &Registry{
		metrics:     make([]metric, 0, _defaultVectorSize),
		constLabels: make(Labels),
		prom:        prom,
		// Assume that we'll be federated with the global prometheus Registry.
		federated: make([]prometheus.Registerer, 0, 1),
		handler:   handler,
		stop:      make(chan struct{}),
		stopped:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Push starts a goroutine that periodically exports all registered metrics to
// a Tally scope. Each Registry can only push to a single Scope; calling Push a
// second time returns an error.
func (r *Registry) Push(scope tally.Scope, tick time.Duration) (context.CancelFunc, error) {
	if r.pushing.Swap(true) {
		return nil, errors.New("already pushing to Tally")
	}
	r.scope = scope
	r.ticker = time.NewTicker(tick)
	go r.exportToTally()
	return func() {
		r.ticker.Stop()
		close(r.stop)
		<-r.stopped
	}, nil
}

// NewCounter constructs a new Counter.
func (r *Registry) NewCounter(opts Opts) (Counter, error) {
	opts = r.addConstLabels(opts)
	if err := opts.validate(); err != nil {
		return nil, err
	}
	c := newCounter(opts)
	if err := r.register(c); err != nil {
		return nil, err
	}
	return c, nil
}

// MustCounter constructs a new Counter. It panics if it encounters an error.
func (r *Registry) MustCounter(opts Opts) Counter {
	c, err := r.NewCounter(opts)
	if err != nil {
		panic(fmt.Sprintf("failed to create Counter with options %+v: %v", opts, err))
	}
	return c
}

// NewGauge constructs a new Gauge.
func (r *Registry) NewGauge(opts Opts) (Gauge, error) {
	opts = r.addConstLabels(opts)
	if err := opts.validate(); err != nil {
		return nil, err
	}
	g := newGauge(opts)
	if err := r.register(g); err != nil {
		return nil, err
	}
	return g, nil
}

// MustGauge constructs a new Gauge. It panics if it encounters an error.
func (r *Registry) MustGauge(opts Opts) Gauge {
	g, err := r.NewGauge(opts)
	if err != nil {
		panic(fmt.Sprintf("failed to create Gauge with options %+v: %v", opts, err))
	}
	return g
}

// NewCounterVector constructs a new CounterVector.
func (r *Registry) NewCounterVector(opts Opts) (CounterVector, error) {
	opts = r.addConstLabels(opts)
	if err := opts.validateVector(); err != nil {
		return nil, err
	}
	v := newCounterVector(opts)
	if err := r.register(v); err != nil {
		return nil, err
	}
	return v, nil
}

// MustCounterVector constructs a new CounterVector. It panics if it encounters
// an error.
func (r *Registry) MustCounterVector(opts Opts) CounterVector {
	v, err := r.NewCounterVector(opts)
	if err != nil {
		panic(fmt.Sprintf("failed to create CounterVector with options %+v: %v", opts, err))
	}
	return v
}

// NewGaugeVector constructs a new CounterVector.
func (r *Registry) NewGaugeVector(opts Opts) (GaugeVector, error) {
	opts = r.addConstLabels(opts)
	if err := opts.validateVector(); err != nil {
		return nil, err
	}
	v := newGaugeVector(opts)
	if err := r.register(v); err != nil {
		return nil, err
	}
	return v, nil
}

// MustGaugeVector constructs a new GaugeVector. It panics if it encounters an
// error.
func (r *Registry) MustGaugeVector(opts Opts) GaugeVector {
	v, err := r.NewGaugeVector(opts)
	if err != nil {
		panic(fmt.Sprintf("failed to create GaugeVector with options %+v: %v", opts, err))
	}
	return v
}

// ServeHTTP implements http.Handler.
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *Registry) exportToTally() {
	defer close(r.stopped)
	// When stopping, do one last export to catch any stragglers.
	defer r.push()

	for {
		select {
		case <-r.stop:
			return
		case <-r.ticker.C:
			r.push()
		}
	}
}

func (r *Registry) push() {
	r.metricsMu.RLock()
	for _, m := range r.metrics {
		m.push(r.scope)
	}
	r.metricsMu.RUnlock()
}

func (r *Registry) register(m metric) error {
	r.metricsMu.Lock()
	r.metrics = append(r.metrics, m)
	r.metricsMu.Unlock()

	if err := r.prom.Register(m); err != nil {
		return err
	}
	for _, fed := range r.federated {
		if err := fed.Register(m); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) addConstLabels(opts Opts) Opts {
	if len(r.constLabels) == 0 {
		return opts
	}
	labels := opts.copyLabels()
	for k, v := range r.constLabels {
		labels[k] = v
	}
	opts.ConstLabels = labels
	return opts
}
