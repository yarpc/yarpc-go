// Copyright (c) 2017 Uber Technologies, Inc.
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

package throttle

import (
	"context"

	"github.com/uber-go/tally"
	"go.uber.org/yarpc/internal/pally"
	"go.uber.org/zap"
)

type metrics struct {
	drops    pally.Counter
	passes   pally.Counter
	overhead pally.Latencies
}

func newMetrics(logger *zap.Logger, scope tally.Scope) (metrics, context.CancelFunc) {
	reg, stopPush := newRegistry(logger, scope)

	labels := pally.Labels{}

	overhead, err := reg.NewLatencies(pally.LatencyOpts{
		Opts: pally.Opts{
			Name:        "overhead_ms",
			Help:        "Latency overhead introduced by the outbound throttle per RPC.",
			ConstLabels: labels,
		},
		Unit:    _ms,
		Buckets: _buckets,
	})
	if err != nil {
		logger.Error("Failed to create throttle overhead latencies metric.")
	}

	passes, err := reg.NewCounter(pally.Opts{
		Name:        "passes",
		Help:        "Number of RPCs allowed by the outbound throttle.",
		ConstLabels: labels,
	})
	if err != nil {
		logger.Error("Failed to create throttle passes metric.")
	}

	drops, err := reg.NewCounter(pally.Opts{
		Name:        "drops",
		Help:        "Number of RPCs dropped while waiting by the outbound throttle.",
		ConstLabels: labels,
	})
	if err != nil {
		logger.Error("Failed to create throttle drops metric.")
	}

	return metrics{
		overhead: overhead,
		passes:   passes,
		drops:    drops,
	}, stopPush
}

func newRegistry(logger *zap.Logger, scope tally.Scope) (*pally.Registry, context.CancelFunc) {
	r := pally.NewRegistry(
		pally.Labeled(pally.Labels{
			"component": _packageName,
		}),
	)

	if scope == nil {
		return r, func() {}
	}

	stop, err := r.Push(scope, _tallyPushInterval)
	if err != nil {
		logger.Error("Failed to start pushing metrics to Tally.", zap.Error(err))
		return r, func() {}
	}
	return r, stop
}
