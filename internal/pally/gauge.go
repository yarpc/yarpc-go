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
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	promproto "github.com/prometheus/client_model/go"
	"github.com/uber-go/tally"
)

type gauge struct {
	value
	tally tally.Gauge
}

func newGauge(opts Opts) *gauge {
	return &gauge{value: newValue(opts)}
}

func (g *gauge) Write(m *promproto.Metric) error {
	m.Label = g.labelPairs
	m.Gauge = &promproto.Gauge{Value: proto.Float64(float64(g.Load()))}
	return nil
}

func (g *gauge) Collect(ch chan<- prometheus.Metric) {
	ch <- g
}

func (g *gauge) push(scope tally.Scope) {
	if g.opts.DisableTally {
		return
	}
	if g.tally == nil {
		labels := g.opts.copyLabels()
		for i, key := range g.opts.VariableLabels {
			labels[key] = g.variableLabelVals[i]
		}
		g.tally = scope.Tagged(labels).Gauge(g.opts.Name)
	}
	g.tally.Update(float64(g.Load()))
}

type gaugeVector vector

func newGaugeVector(opts Opts) *gaugeVector {
	vec := gaugeVector(vector{
		opts:    opts,
		desc:    opts.describe(),
		factory: newDynamicGauge,
		metrics: make(map[string]metric, _defaultVectorSize),
	})
	return &vec
}

func (gv *gaugeVector) Get(variableLabelVals ...string) (Gauge, error) {
	m, err := (*vector)(gv).getOrCreate(variableLabelVals...)
	if err != nil {
		return nil, err
	}
	return m.(Gauge), nil
}

func (gv *gaugeVector) MustGet(variableLabelVals ...string) Gauge {
	g, err := gv.Get(variableLabelVals...)
	if err != nil {
		panic(fmt.Sprintf("failed to get Gauge with labels %v: %v", variableLabelVals, err))
	}
	return g
}

func (gv *gaugeVector) Describe(ch chan<- *prometheus.Desc) { (*vector)(gv).Describe(ch) }
func (gv *gaugeVector) Collect(ch chan<- prometheus.Metric) { (*vector)(gv).Collect(ch) }
func (gv *gaugeVector) push(scope tally.Scope)              { (*vector)(gv).push(scope) }

func newDynamicGauge(opts Opts, desc *prometheus.Desc, variableLabelVals []string) metric {
	scrubbed := scrubLabelValues(variableLabelVals)
	return &gauge{value: value{
		opts:              opts,
		desc:              desc,
		variableLabelVals: scrubbed,
		labelPairs:        opts.labelPairs(scrubbed),
	}}
}
