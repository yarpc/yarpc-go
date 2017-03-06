package pally

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally"
)

type counter struct {
	value
	last  int64
	tally tally.Counter
}

func newCounter(opts Opts) *counter {
	return &counter{value: newValue(opts)}
}

func (c *counter) diff() int64 {
	cur := c.Load()
	diff := cur - c.last
	c.last = cur
	return diff
}

func (c *counter) Collect(ch chan<- prometheus.Metric) {
	m, err := prometheus.NewConstMetric(
		c.desc,
		prometheus.CounterValue,
		float64(c.Load()),
		c.variableLabelVals...,
	)
	if err == nil {
		ch <- m
	}
}

func (c *counter) push(scope tally.Scope) {
	if c.opts.DisableTally {
		return
	}
	if c.tally == nil {
		labels := c.opts.copyLabels()
		for i, key := range c.opts.VariableLabels {
			labels[key] = c.variableLabelVals[i]
		}
		c.tally = scope.Tagged(labels).Counter(c.opts.Name)
	}
	c.tally.Inc(c.diff())
}

type counterVector vector

func newCounterVector(opts Opts) *counterVector {
	vec := counterVector(vector{
		opts:    opts,
		desc:    opts.describe(),
		factory: newDynamicCounter,
		metrics: make(map[string]metric, _defaultVectorSize),
	})
	return &vec
}

func (cv *counterVector) Get(variableLabelVals ...string) (Counter, error) {
	m, err := (*vector)(cv).getOrCreate(variableLabelVals...)
	if err != nil {
		return nil, err
	}
	return m.(Counter), nil
}

func (cv *counterVector) MustGet(variableLabelVals ...string) Counter {
	c, err := cv.Get(variableLabelVals...)
	if err != nil {
		panic(fmt.Sprintf("failed to get Counter with labels %v: %v", variableLabelVals, err))
	}
	return c
}

func (cv *counterVector) Describe(ch chan<- *prometheus.Desc) { (*vector)(cv).Describe(ch) }
func (cv *counterVector) Collect(ch chan<- prometheus.Metric) { (*vector)(cv).Collect(ch) }
func (cv *counterVector) push(scope tally.Scope)              { (*vector)(cv).push(scope) }

func newDynamicCounter(opts Opts, desc *prometheus.Desc, variableLabelVals []string) metric {
	return &counter{value: value{
		opts:              &opts,
		desc:              desc,
		variableLabelVals: variableLabelVals,
	}}
}
