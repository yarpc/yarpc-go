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

package retry

import (
	"context"
	"sync"
	"time"

	"github.com/uber-go/tally"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/digester"
	"go.uber.org/yarpc/internal/pally"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

const (
	// Sleep between pushes to Tally metrics. At some point, we may want this
	// to be configurable.
	_tallyPushInterval = 500 * time.Millisecond
	_packageName       = "yarpc-retry"
	_defaultGraphSize  = 128

	// Retry failure "reason" labels
	_unretryable   = "unretryable"
	_yarpcInternal = "yarpc_internal"
	_noTime        = "no_time"
	_maxAttempts   = "max_attempts"
)

type observerGraph struct {
	reg    *pally.Registry
	logger *zap.Logger

	edgesMu sync.RWMutex
	edges   map[string]*edge
}

func newObserverGraph(logger *zap.Logger, scope tally.Scope) (*observerGraph, context.CancelFunc) {
	reg, stopPush := newRegistry(logger, scope)

	return &observerGraph{
		edges:  make(map[string]*edge, _defaultGraphSize),
		reg:    reg,
		logger: logger,
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

func (g *observerGraph) begin(req *transport.Request) call {
	return call{e: g.getOrCreateEdge(req)}
}

func (g *observerGraph) getOrCreateEdge(req *transport.Request) *edge {
	d := digester.New()
	d.Add(req.Caller)
	d.Add(req.Service)
	d.Add(string(req.Encoding))
	d.Add(req.Procedure)
	d.Add(req.RoutingKey)
	d.Add(req.RoutingDelegate)
	e := g.getOrCreateEdgeForKey(d.Digest(), req)
	d.Free()
	return e
}

func (g *observerGraph) getOrCreateEdgeForKey(key []byte, req *transport.Request) *edge {
	if e := g.getEdge(key); e != nil {
		return e
	}
	return g.createEdge(key, req)
}

func (g *observerGraph) getEdge(key []byte) *edge {
	g.edgesMu.RLock()
	e := g.edges[string(key)]
	g.edgesMu.RUnlock()
	return e
}

func (g *observerGraph) createEdge(key []byte, req *transport.Request) *edge {
	g.edgesMu.Lock()
	// Since we'll rarely hit this code path, the overhead of defer is acceptable.
	defer g.edgesMu.Unlock()

	if e, ok := g.edges[string(key)]; ok {
		// Someone beat us to the punch.
		return e
	}

	e := newEdge(g.logger, g.reg, req)
	g.edges[string(key)] = e
	return e
}

func newEdge(logger *zap.Logger, reg *pally.Registry, req *transport.Request) *edge {
	labels := pally.Labels{
		"source":           pally.ScrubLabelValue(req.Caller),
		"dest":             pally.ScrubLabelValue(req.Service),
		"procedure":        pally.ScrubLabelValue(req.Procedure),
		"encoding":         pally.ScrubLabelValue(string(req.Encoding)),
		"routing_key":      pally.ScrubLabelValue(req.RoutingKey),
		"routing_delegate": pally.ScrubLabelValue(req.RoutingDelegate),
	}
	attempts, err := reg.NewCounter(pally.Opts{
		Name:        "attempts",
		Help:        "Total number of RPC attempts.",
		ConstLabels: labels,
	})
	if err != nil {
		logger.Error("Failed to create attempts counter.", zap.Error(err))
		attempts = pally.NewNopCounter()
	}
	successes, err := reg.NewCounter(pally.Opts{
		Name:        "successes",
		Help:        "Number of successful attempts, including successful initial attempts.",
		ConstLabels: labels,
	})
	if err != nil {
		logger.Error("Failed to create successes counter.", zap.Error(err))
		successes = pally.NewNopCounter()
	}
	retriesAfterError, err := reg.NewCounterVector(pally.Opts{
		Name:           "retries_after_error",
		Help:           "Total RPC retry attempts for each prior error.",
		ConstLabels:    labels,
		VariableLabels: []string{"error"},
	})
	if err != nil {
		logger.Error("Failed to create retry after error vector.", zap.Error(err))
		retriesAfterError = pally.NewNopCounterVector()
	}
	failures, err := reg.NewCounterVector(pally.Opts{
		Name:           "retry_failures",
		Help:           "Number of RPC final attempt failures.",
		ConstLabels:    labels,
		VariableLabels: []string{"reason", "error"},
	})
	if err != nil {
		logger.Error("Failed to create retry failures reason and error vector.", zap.Error(err))
		failures = pally.NewNopCounterVector()
	}
	return &edge{
		attempts:          attempts,
		successes:         successes,
		retriesAfterError: retriesAfterError,
		failures:          failures,
	}
}

type edge struct {
	attempts  pally.Counter
	successes pally.Counter

	// Retry counter that has the error being retried.
	retriesAfterError pally.CounterVector

	// Failures are hard exits from the retry loop.  Failures will log the
	// reason we didn't retry, and the error we just had.
	failures pally.CounterVector
}

// call is carried through an outbound request with retries.  It will record
// all appropriate data onto the edge.
type call struct {
	e *edge
}

func (c call) attempt() {
	c.e.attempts.Inc()
}

func (c call) success() {
	c.e.successes.Inc()
}

func (c call) retryOnError(err error) {
	if counter, err := c.e.retriesAfterError.Get(getErrorName(err)); err == nil {
		counter.Inc()
	}
}

func (c call) unretryableError(err error) {
	if counter, err := c.e.failures.Get(_unretryable, getErrorName(err)); err == nil {
		counter.Inc()
	}
}

func (c call) yarpcInternalError(err error) {
	if counter, err := c.e.failures.Get(_yarpcInternal, getErrorName(err)); err == nil {
		counter.Inc()
	}
}

func (c call) noTimeError(err error) {
	if counter, err := c.e.failures.Get(_noTime, getErrorName(err)); err == nil {
		counter.Inc()
	}
}

func (c call) maxAttemptsError(err error) {
	if counter, err := c.e.failures.Get(_maxAttempts, getErrorName(err)); err == nil {
		counter.Inc()
	}
}

func getErrorName(err error) string {
	return yarpcerrors.ErrorCode(err).String()
}
