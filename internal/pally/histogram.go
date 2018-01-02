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
	"math"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	promproto "github.com/prometheus/client_model/go"
	"github.com/uber-go/tally"
	"go.uber.org/atomic"
)

type bucket struct {
	atomic.Int64

	last  int64 // last value pushed to Tally
	upper int64 // bucket upper bound, inclusive
}

func (b *bucket) diff() int64 {
	cur := b.Load()
	diff := cur - b.last
	b.last = cur
	return diff
}

type buckets []*bucket

func (bs buckets) get(val int64) *bucket {
	// Binary search to find the correct bucket for this observation.
	i, j := 0, len(bs)
	for i < j {
		h := i + (j-i)/2
		if val > bs[h].upper {
			i = h + 1
		} else {
			j = h
		}
	}
	return bs[i]
}

type histogram struct {
	buckets buckets
	// Prometheus requires us to track the sum of all observed values.
	sum atomic.Int64

	opts              LatencyOpts
	desc              *prometheus.Desc
	tally             tally.Histogram
	variableLabelVals []string
	labelPairs        []*promproto.LabelPair
}

func newHistogram(opts LatencyOpts) *histogram {
	return &histogram{
		buckets:    opts.buckets(),
		opts:       opts,
		desc:       opts.describe(),
		labelPairs: opts.labelPairs(nil /* variable label vals */),
	}
}

func (h *histogram) Observe(d time.Duration) {
	n := int64(d / h.opts.Unit)
	bucket := h.buckets.get(n)
	bucket.Inc()
	h.sum.Add(n)
}

func (h *histogram) push(scope tally.Scope) {
	if h.opts.DisableTally {
		return
	}
	if h.tally == nil {
		labels := h.opts.copyLabels()
		for i, key := range h.opts.VariableLabels {
			labels[key] = h.variableLabelVals[i]
		}
		h.tally = scope.Tagged(labels).Histogram(
			h.opts.Name,
			tally.DurationBuckets(h.opts.Buckets),
		)
	}
	for _, bucket := range h.buckets {
		diff := bucket.diff()
		// TODO: either add a Tally API to observe multiple values or roll our
		// own counter-based histogram implementation.
		for i := int64(0); i < diff; i++ {
			h.tally.RecordDuration(time.Duration(bucket.upper) * h.opts.Unit)
		}
	}
}

func (h *histogram) Desc() *prometheus.Desc {
	return h.desc
}

func (h *histogram) Write(m *promproto.Metric) error {
	n := uint64(0)
	promBuckets := make([]*promproto.Bucket, 0, len(h.buckets)-1)
	for _, b := range h.buckets {
		n += uint64(b.Load())
		if b.upper == math.MaxInt64 {
			// Prometheus doesn't want us to export the final catch-all bucket.
			continue
		}
		promBuckets = append(promBuckets, &promproto.Bucket{
			CumulativeCount: proto.Uint64(n),
			UpperBound:      proto.Float64(float64(b.upper)),
		})
	}

	m.Label = h.labelPairs
	m.Histogram = &promproto.Histogram{
		SampleCount: proto.Uint64(n),
		SampleSum:   proto.Float64(float64(h.sum.Load())),
		Bucket:      promBuckets,
	}
	return nil
}

func (h *histogram) Collect(ch chan<- prometheus.Metric) {
	ch <- h
}

func (h *histogram) Describe(ch chan<- *prometheus.Desc) {
	ch <- h.desc
}

type histogramVector struct {
	opts LatencyOpts
	desc *prometheus.Desc

	histogramsMu sync.RWMutex
	// map key is the variable label values, joined by a null byte
	histograms map[string]*histogram
}

func newHistogramVector(opts LatencyOpts) *histogramVector {
	return &histogramVector{
		opts:       opts,
		desc:       opts.describe(),
		histograms: make(map[string]*histogram, _defaultVectorSize),
	}
}

func (vec *histogramVector) MustGet(labels ...string) Latencies {
	l, err := vec.Get(labels...)
	if err != nil {
		panic(fmt.Sprintf("failed to get Latencies with labels %v: %v", labels, err))
	}
	return l
}

func (vec *histogramVector) Get(labels ...string) (Latencies, error) {
	digester := newDigester()
	for _, s := range labels {
		digester.add(ScrubLabelValue(s))
	}

	vec.histogramsMu.RLock()
	m, ok := vec.histograms[string(digester.digest())]
	vec.histogramsMu.RUnlock()
	if ok {
		digester.free()
		return m, nil
	}

	vec.histogramsMu.Lock()
	m, err := vec.newHistogram(digester.digest(), labels)
	vec.histogramsMu.Unlock()
	digester.free()

	return m, err
}

func (vec *histogramVector) newHistogram(key []byte, variableLabelVals []string) (*histogram, error) {
	m, ok := vec.histograms[string(key)]
	if ok {
		return m, nil
	}
	if len(vec.opts.VariableLabels) != len(variableLabelVals) {
		return nil, errInconsistentCardinality
	}
	scrubbed := scrubLabelValues(variableLabelVals)
	m = &histogram{
		buckets:           vec.opts.buckets(),
		opts:              vec.opts,
		desc:              vec.desc,
		variableLabelVals: scrubbed,
		labelPairs:        vec.opts.labelPairs(scrubbed),
	}
	vec.histograms[string(key)] = m
	return m, nil
}

func (vec *histogramVector) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
}

func (vec *histogramVector) Collect(ch chan<- prometheus.Metric) {
	vec.histogramsMu.RLock()
	for _, m := range vec.histograms {
		m.Collect(ch)
	}
	vec.histogramsMu.RUnlock()
}

func (vec *histogramVector) push(scope tally.Scope) {
	vec.histogramsMu.RLock()
	for _, m := range vec.histograms {
		m.push(scope)
	}
	vec.histogramsMu.RUnlock()
}
