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

func (c *counter) Write(m *promproto.Metric) error {
	m.Label = c.labelPairs
	m.Counter = &promproto.Counter{Value: proto.Float64(float64(c.Load()))}
	return nil
}

func (c *counter) Collect(ch chan<- prometheus.Metric) {
	ch <- c
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
	scrubbed := scrubLabelValues(variableLabelVals)
	return &counter{value: value{
		opts:              opts,
		desc:              desc,
		variableLabelVals: scrubbed,
		labelPairs:        opts.labelPairs(scrubbed),
	}}
}
