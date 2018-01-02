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
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	promproto "github.com/prometheus/client_model/go"
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
	labelPairs        []*promproto.LabelPair
}

func newValue(opts Opts) value {
	return value{
		opts:       opts,
		desc:       opts.describe(),
		labelPairs: opts.labelPairs(nil /* variable label vals */),
	}
}

// Desc implements half of prometheus.Metric.
func (v value) Desc() *prometheus.Desc {
	return v.desc
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
	digester := newDigester()
	for _, s := range labels {
		digester.add(ScrubLabelValue(s))
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
