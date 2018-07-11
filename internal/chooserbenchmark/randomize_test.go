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
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestLogNormalLatency(t *testing.T) {
	inputs := []int{1000, 0, 1, -1, 10}

	for _, n := range inputs {
		latency := time.Millisecond * time.Duration(n)
		logNormal := NewLogNormalLatency(latency, DefaultLogNormalSigma)
		if latency < 0 {
			assert.Equal(t, time.Duration(1), logNormal.Median())
			continue
		}
		median := logNormal.Median()
		value := logNormal.Random()
		assert.True(t, median >= 0 && value >= 0, fmt.Sprintf("median: %v, value: %v", median, value))
		cdf := logNormal.CDF(float64(latency))
		assert.True(t, cdf >= 0 && cdf <= 1, fmt.Sprintf("latency: %v, cdf: %v", latency, cdf))
	}
}

func TestNormalDistSleepTime(t *testing.T) {
	inputs := []int{100, 0, 1, -1}

	for _, n := range inputs {
		normal := NewNormalDistSleepTime(n)
		value := normal.Random()
		assert.True(t, value >= 0, fmt.Sprintf("RPS: %v, random sleep time: %v", n, value))
	}
}

func TestP99LatencyComputation(t *testing.T) {
	logNormal := NewLogNormalLatency(time.Millisecond*300, DefaultLogNormalSigma)
	latencies := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		latencies[i] = logNormal.PXXLatency(float64(i) / float64(100))
		assert.True(t, latencies[i] > 0)
	}
}
