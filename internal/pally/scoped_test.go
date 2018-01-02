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
	"context"
	"net/http"
	"time"

	"github.com/uber-go/tally"
	"go.uber.org/yarpc/internal/pally"
)

// If you'd prefer to use pure dependency injection and scope your metrics
// to a single struct, create a new pally.Registry in your struct's
// constructor. In this case, we're also exporting our metrics to a Tally
// scope, which can report to StatsD- or M3-aware systems.
type Resolver struct {
	registry        *pally.Registry
	watches         pally.Gauge
	resolves        pally.CounterVector
	stopTallyExport context.CancelFunc
}

func NewResolver(scope tally.Scope) (*Resolver, error) {
	reg := pally.NewRegistry()
	stop, err := reg.Push(scope, time.Second)
	if err != nil {
		return nil, err
	}

	watches, err := _reg.NewGauge(pally.Opts{
		Name: "watch_count",
		Help: "Current number of active service name watches.",
		ConstLabels: pally.Labels{
			"foo": "bar",
		},
	})
	if err != nil {
		return nil, err
	}

	resolves, err := _reg.NewCounterVector(pally.Opts{
		Name: "resolve_count",
		Help: "Total name resolves by path.",
		ConstLabels: pally.Labels{
			"foo": "bar",
		},
		VariableLabels: []string{"service"},
	})
	if err != nil {
		return nil, err
	}

	return &Resolver{
		registry:        reg,
		watches:         watches,
		resolves:        resolves,
		stopTallyExport: stop,
	}, nil
}

func (r *Resolver) Watch() {
	r.watches.Inc()
}

func (r *Resolver) Resolve(name string) {
	if c, err := r.resolves.Get(name); err == nil {
		c.Inc()
	}
}

func (r *Resolver) Close() {
	r.stopTallyExport()
}

func (r *Resolver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Our registry can report its own metrics via a Prometheus-compatible HTTP
	// handler.
	r.registry.ServeHTTP(w, req)
}

func Example_dependencyInjection() {
	scope := tally.NewTestScope("testing", nil /* labels */)
	reg, err := NewResolver(scope)
	if err != nil {
		panic(err.Error())
	}
	reg.Watch()
	reg.Resolve("some_service")
}
