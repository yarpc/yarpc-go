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
	"context"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	"github.com/uber-go/tally/m3"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/internal/pally/pallytest"
)

func TestSimpleMetricDuplicates(t *testing.T) {
	r := NewRegistry()
	opts := Opts{
		Name: "foo",
		Help: "help",
	}
	_, err := r.NewCounter(opts)
	assert.NoError(t, err, "Unexpected error registering metadata for the first time.")

	t.Run("same type", func(t *testing.T) {
		// You can't reuse options with the same metric type.
		_, err := r.NewCounter(opts)
		assert.Error(t, err, "Unexpected success re-using metrics metadata.")
		assert.Panics(t, func() { r.MustCounter(opts) }, "Unexpected success re-using metrics metadata.")
	})

	t.Run("different type", func(t *testing.T) {
		// Even if you change the metric type, you still can't re-use metadata.
		_, err := r.NewGauge(opts)
		assert.Error(t, err, "Unexpected success re-using metrics metadata.")
		assert.Panics(t, func() { r.MustGauge(opts) }, "Unexpected success re-using metrics metadata.")
	})
}

func TestVectorMetricDuplicates(t *testing.T) {
	r := NewRegistry()
	opts := Opts{
		Name:           "foo",
		Help:           "help",
		VariableLabels: []string{"foo"},
	}
	_, err := r.NewCounterVector(opts)
	assert.NoError(t, err, "Unexpected error registering vector metadata for the first time.")

	t.Run("same type", func(t *testing.T) {
		// You can't reuse options with the same metric type.
		_, err := r.NewCounterVector(opts)
		assert.Error(t, err, "Unexpected success re-using vector metrics metadata.")
		assert.Panics(t, func() { r.MustCounterVector(opts) }, "Unexpected success re-using vector metrics metadata.")
	})

	t.Run("different type", func(t *testing.T) {
		// Even if you change the metric type, you still can't re-use metadata.
		_, err := r.NewGaugeVector(opts)
		assert.Error(t, err, "Unexpected success re-using vector metrics metadata.")
		assert.Panics(t, func() { r.MustGaugeVector(opts) }, "Unexpected success re-using vector metrics metadata.")
	})
}

func TestFederatedMetrics(t *testing.T) {
	prom := prometheus.NewRegistry()
	r := NewRegistry(Federated(prom))
	opts := Opts{
		Name: "foo",
		Help: "Some help.",
	}
	c, err := r.NewCounter(opts)
	assert.NoError(t, err, "Unexpected error registering vector metadata for the first time.")

	c.Inc()
	expected := "# HELP foo Some help.\n" +
		"# TYPE foo counter\n" +
		"foo 1"

	pallytest.AssertPrometheus(t, promhttp.HandlerFor(prom, promhttp.HandlerOpts{}), expected)
}

func TestConstLabelValidation(t *testing.T) {
	r := NewRegistry(Labeled(Labels{
		"invalid-prometheus-name": "foo",
		"tally":                   "invalid!value",
		"ok":                      "yes",
	}))
	_, err := r.NewCounter(Opts{
		Name: "test",
		Help: "help",
	})
	require.NoError(t, err, "Unexpected error creating a counter.")
	pallytest.AssertPrometheus(t, r, "# HELP test help\n"+
		"# TYPE test counter\n"+
		`test{ok="yes"} 0`)
}

func BenchmarkCreateNewMetrics(b *testing.B) {
	b.Run("create Pally counter", func(b *testing.B) {
		r := NewRegistry()
		var count atomic.Int64
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				opts := Opts{
					Name:        "foo",
					Help:        "Some help.",
					ConstLabels: Labels{"iteration": strconv.FormatInt(count.Inc(), 10)},
				}
				r.NewCounter(opts)
			}
		})
	})
	b.Run("create Prometheus counter", func(b *testing.B) {
		r := prometheus.NewRegistry()
		var count atomic.Int64
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c := prometheus.NewCounter(prometheus.CounterOpts{
					Name:        "foo",
					Help:        "Some help.",
					ConstLabels: prometheus.Labels{"iteration": strconv.FormatInt(count.Inc(), 10)},
				})
				r.MustRegister(c)
			}
		})
	})
	b.Run("create Tally counter", func(b *testing.B) {
		scope, close := newTallyScope(b)
		defer close()
		var count atomic.Int64
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tags := map[string]string{"iteration": strconv.FormatInt(count.Inc(), 10)}
				scope.Tagged(tags).Counter("foo")
			}
		})
	})
	b.Run("create dynamic Pally counter", func(b *testing.B) {
		vec := NewRegistry().MustCounterVector(Opts{
			Name:           "foo",
			Help:           "Some help.",
			VariableLabels: []string{"foo", "bar"},
		})
		var count atomic.Int64
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				foo := strconv.FormatInt(count.Inc(), 10)
				bar := strconv.FormatInt(count.Inc(), 10)
				vec.MustGet(foo, bar)
			}
		})
	})
	b.Run("create dynamic Prometheus counter", func(b *testing.B) {
		r := prometheus.NewRegistry()
		vec := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "foo",
			Help: "Some help.",
		}, []string{"foo", "bar"})
		r.MustRegister(vec)
		var count atomic.Int64
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				foo := strconv.FormatInt(count.Inc(), 10)
				bar := strconv.FormatInt(count.Inc(), 10)
				vec.WithLabelValues(foo, bar)
			}
		})
	})
	b.Run("create dynamic Tally counter", func(b *testing.B) {
		scope, close := newTallyScope(b)
		defer close()
		var count atomic.Int64
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				foo := strconv.FormatInt(count.Inc(), 10)
				bar := strconv.FormatInt(count.Inc(), 10)
				scope.Tagged(map[string]string{"foo": foo, "bar": bar}).Counter("foo")
			}
		})
	})
	b.Run("increment Pally counter", func(b *testing.B) {
		c := NewRegistry().MustCounter(Opts{
			Name: "foo",
			Help: "Some help.",
		})
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.Inc()
			}
		})
	})
	b.Run("increment Prometheus counter", func(b *testing.B) {
		r := prometheus.NewRegistry()
		c := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "foo",
			Help: "Some help.",
		})
		r.MustRegister(c)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.Inc()
			}
		})
	})
	b.Run("increment Tally counter", func(b *testing.B) {
		scope, close := newTallyScope(b)
		defer close()
		c := scope.Counter("foo")
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.Inc(1)
			}
		})
	})
	b.Run("increment dynamic Pally counter", func(b *testing.B) {
		vec := NewRegistry().MustCounterVector(Opts{
			Name:           "foo",
			Help:           "Some help.",
			VariableLabels: []string{"foo", "bar"},
		})
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				vec.MustGet("one", "two").Inc()
			}
		})
	})
	b.Run("increment dynamic Prometheus counter", func(b *testing.B) {
		r := prometheus.NewRegistry()
		vec := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "foo",
			Help: "Some help.",
		}, []string{"foo", "bar"})
		r.MustRegister(vec)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				vec.WithLabelValues("one", "two").Inc()
			}
		})
	})
	b.Run("increment dynamic Tally counter", func(b *testing.B) {
		scope, close := newTallyScope(b)
		defer close()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				scope.Tagged(map[string]string{"foo": "one", "bar": "two"}).Counter("foo").Inc(1)
			}
		})
	})
}

func BenchmarkScrape(b *testing.B) {
	r := NewRegistry()
	// Populate the registry with a few metrics.
	r.MustCounter(Opts{
		Name:        "foo",
		Help:        "Some help.",
		ConstLabels: Labels{"bar": "baz1"},
	})
	r.MustGauge(Opts{
		Name:        "foo",
		Help:        "Some help.",
		ConstLabels: Labels{"bar": "baz2"},
	})
	r.MustLatencies(LatencyOpts{
		Opts: Opts{
			Name:        "foo",
			Help:        "Some help.",
			ConstLabels: Labels{"bar": "baz3"},
		},
		Unit:    time.Millisecond,
		Buckets: []time.Duration{time.Millisecond, 10 * time.Millisecond, 100 * time.Millisecond},
	})

	req := httptest.NewRequest("GET", "/" /* target */, nil /* body */)
	resw := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(resw, req)
		resw.Body.Reset()
	}
}

// Create a real, M3-backed Tally scope.
func newTallyScope(t testing.TB) (tally.Scope, context.CancelFunc) {
	reporter, err := m3.NewReporter(m3.Options{
		HostPorts: []string{"localhost:1234"},
		Service:   "benchmark",
		Env:       "production",
	})
	require.NoError(t, err, "Failed to construct an M3 reporter.")
	scope, close := tally.NewRootScope(
		tally.ScopeOptions{CachedReporter: reporter},
		time.Second,
	)
	return scope, func() { close.Close() }
}
