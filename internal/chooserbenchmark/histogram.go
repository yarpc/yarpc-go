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

package chooserbenchmark

import (
	"fmt"

	"go.uber.org/atomic"
	"go.uber.org/net/metrics/bucket"
)

var (
	// BucketMs use rpc latency buckets in net metrics
	BucketMs = bucket.NewRPCLatency()
)

// Histogram contains buckets and counters for collecting metrics
type Histogram struct {
	buckets   []int64
	counters  []atomic.Int64
	bucketLen int
	unit      int64
}

// NewRequestCounterBuckets create global request counter bucket based on
// maximum request count and minimum request count
func NewRequestCounterBuckets(minCount, maxCount int64, bucketLen int) []int64 {
	buckets := make([]int64, bucketLen)
	diff := maxCount - minCount
	width, remain := diff/int64(bucketLen), int(diff%int64(bucketLen))
	if width == 0 {
		width = 1
		remain = 0
	}
	cur := minCount
	for i := 0; i < bucketLen; i++ {
		cur += width
		if i < remain {
			cur++
		}
		buckets[i] = cur
	}
	return buckets
}

// NewHistogram create a histogram based on buckets and metrics unit
func NewHistogram(buckets []int64, unit int64) *Histogram {
	bucketLen := len(buckets)
	return &Histogram{
		buckets:   buckets,
		counters:  make([]atomic.Int64, bucketLen),
		bucketLen: bucketLen,
		unit:      unit,
	}
}

// IncBucket increase the counter at the corresponding index of given value
func (h *Histogram) IncBucket(v int64) {
	v = v / h.unit
	bucketLen := h.bucketLen
	maxBucket := h.buckets[bucketLen-1]
	if v > maxBucket {
		v = maxBucket
	}
	i := 0
	for h.buckets[i] < v {
		i++
	}
	h.counters[i].Inc()
}

// MergeBucket merge the counters of two same buckets set
func (h *Histogram) MergeBucket(that *Histogram) error {
	if h.bucketLen != that.bucketLen {
		return fmt.Errorf("bucket length must be same, length1: %d, length2: %d", h.bucketLen, that.bucketLen)
	}
	for i := 0; i < h.bucketLen; i++ {
		if h.buckets[i] != that.buckets[i] {
			return fmt.Errorf("bucket value on index %d must be same, value1: %d, value2: %d",
				i, h.bucketLen, that.bucketLen)
		}
	}
	for i := 0; i < h.bucketLen; i++ {
		h.counters[i].Add(that.counters[i].Load())
	}
	return nil
}

// Sum returns sum of all counters
func (h *Histogram) Sum() int64 {
	sum := int64(0)
	for i := 0; i < h.bucketLen; i++ {
		sum += h.counters[i].Load()
	}
	return sum
}

// Max returns maximum of all counters
func (h *Histogram) Max() int64 {
	max := int64(0)
	for i := 0; i < h.bucketLen; i++ {
		count := h.counters[i].Load()
		if count > max {
			max = count
		}
	}
	return max
}

// WeightedSum returns sum of counter weighted with bucket
func (h *Histogram) WeightedSum() int64 {
	sum := int64(0)
	for i := 0; i < h.bucketLen; i++ {
		sum += h.counters[i].Load() * h.buckets[i]
	}
	return sum
}
