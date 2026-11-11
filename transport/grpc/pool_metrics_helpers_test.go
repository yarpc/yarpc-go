// Copyright (c) 2026 Uber Technologies, Inc.
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

package grpc

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
)

// poolMetricSample is one conn_pool_* series scraped from a metrics.Root.
type poolMetricSample struct {
	name   string
	kind   string // "gauge" or "counter"
	labels map[string]string
	value  int64
}

var (
	promTypeLine   = regexp.MustCompile(`^# TYPE (conn_pool_\w+) (gauge|counter)$`)
	promSampleLine = regexp.MustCompile(`^(conn_pool_\w+)(?:\{([^}]*)\})? (\S+)$`)
	promLabelPair  = regexp.MustCompile(`(\w+)="([^"]*)"`)
)

// scrapePoolMetrics reads every conn_pool_* series from root through its
// Prometheus handler. Unlike Root.Snapshot, which copies the metrics' atomic
// values non-atomically and so races with pool goroutines that are still
// updating them, the handler reads each value with an atomic Load, so it is
// safe to call while the pool is live (for example from require.Eventually).
func scrapePoolMetrics(tb testing.TB, root *metrics.Root) []poolMetricSample {
	tb.Helper()
	rec := httptest.NewRecorder()
	root.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(tb, http.StatusOK, rec.Code)

	kinds := make(map[string]string)
	var samples []poolMetricSample
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if m := promTypeLine.FindStringSubmatch(line); m != nil {
			kinds[m[1]] = m[2]
			continue
		}
		m := promSampleLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		f, err := strconv.ParseFloat(m[3], 64)
		require.NoError(tb, err, "parsing sample %q", line)
		labels := make(map[string]string)
		for _, kv := range promLabelPair.FindAllStringSubmatch(m[2], -1) {
			labels[kv[1]] = kv[2]
		}
		samples = append(samples, poolMetricSample{name: m[1], kind: kinds[m[1]], labels: labels, value: int64(f)})
	}
	return samples
}

// poolMetricValues returns the current conn_pool_* gauge and counter values
// keyed by metric name. See scrapePoolMetrics for why this exists.
func poolMetricValues(tb testing.TB, root *metrics.Root) (gauges, counters map[string]int64) {
	tb.Helper()
	gauges = make(map[string]int64)
	counters = make(map[string]int64)
	for _, s := range scrapePoolMetrics(tb, root) {
		if s.kind == "counter" {
			counters[s.name] = s.value
		} else {
			gauges[s.name] = s.value
		}
	}
	return gauges, counters
}
