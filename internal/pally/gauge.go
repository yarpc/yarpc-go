package pally

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally"
)

type gauge struct {
	value
	tally tally.Gauge
}

func newGauge(opts Opts) *gauge {
	return &gauge{value: newValue(opts)}
}

func (g *gauge) Collect(ch chan<- prometheus.Metric) {
	m, err := prometheus.NewConstMetric(
		g.desc,
		prometheus.GaugeValue,
		float64(g.Load()),
		g.variableLabelVals...,
	)
	if err == nil {
		ch <- m
	}
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
	return &gauge{value: value{
		opts:              opts,
		desc:              desc,
		variableLabelVals: variableLabelVals,
	}}
}
