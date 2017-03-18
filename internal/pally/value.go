package pally

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally"
	"go.uber.org/atomic"
)

const _defaultVectorSize = 100

// Value is an atomic with some associated metadata. It's a building block
// for higher-level metric types.
type value struct {
	atomic.Int64

	opts              Opts
	desc              *prometheus.Desc
	variableLabelVals []string
}

func newValue(opts Opts) value {
	return value{opts: opts, desc: opts.describe()}
}

// Describe implements half of prometheus.Collector.
func (v value) Describe(ch chan<- *prometheus.Desc) {
	ch <- v.desc
}

// A vector is a collection of values that share the same metadata.
type vector struct {
	opts Opts
	desc *prometheus.Desc

	// The factory function creates a new metric given the vector's metadata
	// and values for the variable labels.
	factory func(Opts, *prometheus.Desc, []string) metric

	metricsMu sync.RWMutex
	metrics   map[string]metric
}

func (vec *vector) getOrCreate(labels ...string) (metric, error) {
	for _, l := range labels {
		if !isValidTallyString(l) {
			return nil, errors.New("variable label values must also be Tally-compatible")
		}
	}

	digester := newDigester()
	for _, s := range labels {
		digester.add(s)
	}

	vec.metricsMu.RLock()
	m, ok := vec.metrics[string(digester.digest())]
	vec.metricsMu.RUnlock()
	if ok {
		digester.free()
		return m, nil
	}

	vec.metricsMu.Lock()
	m, err := vec.newValue(digester.digest(), labels)
	vec.metricsMu.Unlock()
	digester.free()

	return m, err
}

func (vec *vector) newValue(key []byte, variableLabelVals []string) (metric, error) {
	m, ok := vec.metrics[string(key)]
	if ok {
		return m, nil
	}
	if len(vec.opts.VariableLabels) != len(variableLabelVals) {
		return nil, errInconsistentCardinality
	}
	m = vec.factory(vec.opts, vec.desc, variableLabelVals)
	vec.metrics[string(key)] = m
	return m, nil
}

func (vec *vector) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
}

func (vec *vector) Collect(ch chan<- prometheus.Metric) {
	vec.metricsMu.RLock()
	for _, m := range vec.metrics {
		m.Collect(ch)
	}
	vec.metricsMu.RUnlock()
}

func (vec *vector) push(scope tally.Scope) {
	vec.metricsMu.RLock()
	for _, m := range vec.metrics {
		m.push(scope)
	}
	vec.metricsMu.RUnlock()
}
