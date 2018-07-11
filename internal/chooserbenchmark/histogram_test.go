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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestNewRequestCounterBuckets(t *testing.T) {
	tests := []struct {
		msg    string
		input  []int64
		output []int64
	}{
		{
			msg:    "diff is divisible by bucket length",
			input:  []int64{1, 11, 10},
			output: []int64{2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
		{
			msg:    "diff is not divisible by bucket length",
			input:  []int64{1, 12, 10},
			output: []int64{3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		{
			msg:    "diff is smaller than bucket length",
			input:  []int64{1, 4, 10},
			output: []int64{2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			minCount, maxCount, bucketLen := tt.input[0], tt.input[1], int(tt.input[2])
			assert.Equal(t, tt.output, NewRequestCounterBuckets(minCount, maxCount, bucketLen))
		})
	}
}

func TestHistogram(t *testing.T) {
	histogram := NewHistogram(BucketMs, int64(time.Millisecond))
	goRoutineCount := 4
	responseCount := 1000
	min, max := BucketMs[0], BucketMs[len(BucketMs)-1]
	wg := sync.WaitGroup{}
	wg.Add(4)
	for i := 0; i < goRoutineCount; i++ {
		go func() {
			for j := 0; j < responseCount; j++ {
				value := int64(rand.Intn(int(max-min))) + min
				histogram.IncBucket(value)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(goRoutineCount*responseCount), histogram.Sum())
}

func TestMergeBucket(t *testing.T) {
	tests := []struct {
		msg          string
		thisBuckets  []int64
		thatBuckets  []int64
		thisCounters []int64
		thatCounters []int64
		expect       []int64
		wantError    string
	}{
		{
			msg:         "bucket length must be same",
			thisBuckets: []int64{1},
			thatBuckets: []int64{1, 2},
			wantError:   "bucket length must be same",
		},
		{
			msg:         "bucket values must be same",
			thisBuckets: []int64{1, 3},
			thatBuckets: []int64{1, 2},
			wantError:   "bucket value on index",
		},
		{
			msg:          "normal case",
			thisBuckets:  []int64{2, 11, 23, 432},
			thatBuckets:  []int64{2, 11, 23, 432},
			thisCounters: []int64{0, 1, 2, 4},
			thatCounters: []int64{10, 123, 4234, 123},
			expect:       []int64{10, 124, 4236, 127},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			this := NewHistogram(tt.thisBuckets, 1)
			that := NewHistogram(tt.thatBuckets, 1)
			for i, counter := range tt.thisCounters {
				this.counters[i].Add(counter)
			}
			for i, counter := range tt.thatCounters {
				that.counters[i].Add(counter)
			}
			if tt.wantError != "" {
				err := this.MergeBucket(that)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			} else {
				err := this.MergeBucket(that)
				assert.NoError(t, err)
				for i := 0; i < len(tt.thisBuckets); i++ {
					assert.Equal(t, tt.expect[i], this.counters[i].Load())
				}
			}
		})
	}
}

func TestHistogramMetrics(t *testing.T) {
	buckets := []int64{1, 2, 3, 4, 5}
	counters := []int64{1, 2, 3, 4, 5}
	bucketLen := len(buckets)
	histogram := NewHistogram(buckets, 1)
	for i := 0; i < bucketLen; i++ {
		histogram.counters[i].Add(counters[i])
	}
	assert.Equal(t, int64(5), histogram.Max())
	assert.Equal(t, int64(15), histogram.Sum())
	assert.Equal(t, int64(55), histogram.WeightedSum())
}
