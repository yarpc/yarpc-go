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

package pally_test

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/yarpc/internal/pally"
)

// For expvar-style usage (where all metrics are package-global), create a
// package-global pally.Registry and use the Must* constructors. This
// guarantees that all your metrics are unique.
var (
	_reg = pally.NewRegistry(
		// Also register all metrics with the package-global Prometheus
		// registry.
		pally.Federated(prometheus.DefaultRegisterer),
	)
	_watches = _reg.MustGauge(pally.Opts{
		Name: "watch_count",
		Help: "Current number of active service name watches.",
		ConstLabels: pally.Labels{
			"foo": "bar",
		},
	})
	_resolvesPerName = _reg.MustCounterVector(pally.Opts{
		Name: "resolve_count",
		Help: "Total name resolves by service.",
		ConstLabels: pally.Labels{
			"foo": "bar",
		},
		VariableLabels: []string{"service"},
	})
)

func Example_globalMetrics() {
	_watches.Store(42)
	_resolvesPerName.MustGet("some_service").Inc()
}
