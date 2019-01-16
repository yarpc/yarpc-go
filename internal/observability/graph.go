// Copyright (c) 2019 Uber Technologies, Inc.
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

package observability

import (
	"context"
	"sync"
	"time"

	"go.uber.org/net/metrics"
	"go.uber.org/net/metrics/bucket"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/digester"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	_timeNow          = time.Now // for tests
	_defaultGraphSize = 128
	// Latency buckets for histograms. At some point, we may want to make these
	// configurable.
	_bucketsMs = bucket.NewRPCLatency()
)

type directionName string

const (
	_directionOutbound directionName = "outbound"
	_directionInbound  directionName = "inbound"
)

// A graph represents a collection of services: each service is a node, and we
// collect stats for each caller-callee-transport-encoding-procedure-rk-sk-rd edge.
type graph struct {
	meter   *metrics.Scope
	logger  *zap.Logger
	extract ContextExtractor

	edgesMu sync.RWMutex
	edges   map[string]*edge

	succLevel, failLevel, appErrLevel zapcore.Level
}

func newGraph(meter *metrics.Scope, logger *zap.Logger, extract ContextExtractor) graph {
	return graph{
		edges:       make(map[string]*edge, _defaultGraphSize),
		meter:       meter,
		logger:      logger,
		extract:     extract,
		succLevel:   zapcore.DebugLevel,
		failLevel:   zapcore.ErrorLevel,
		appErrLevel: zapcore.ErrorLevel,
	}
}

// begin starts a call along an edge.
func (g *graph) begin(ctx context.Context, rpcType transport.Type, direction directionName, req *transport.Request) call {
	now := _timeNow()

	d := digester.New()
	d.Add(req.Caller)
	d.Add(req.Service)
	d.Add(req.Transport)
	d.Add(string(req.Encoding))
	d.Add(req.Procedure)
	d.Add(req.RoutingKey)
	d.Add(req.RoutingDelegate)
	d.Add(string(direction))
	e := g.getOrCreateEdge(d.Digest(), req, string(direction))
	d.Free()

	return call{
		edge:        e,
		extract:     g.extract,
		started:     now,
		ctx:         ctx,
		req:         req,
		rpcType:     rpcType,
		direction:   direction,
		succLevel:   g.succLevel,
		failLevel:   g.failLevel,
		appErrLevel: g.appErrLevel,
	}
}

func (g *graph) getOrCreateEdge(key []byte, req *transport.Request, direction string) *edge {
	if e := g.getEdge(key); e != nil {
		return e
	}
	return g.createEdge(key, req, direction)
}

func (g *graph) getEdge(key []byte) *edge {
	g.edgesMu.RLock()
	e := g.edges[string(key)]
	g.edgesMu.RUnlock()
	return e
}

func (g *graph) createEdge(key []byte, req *transport.Request, direction string) *edge {
	g.edgesMu.Lock()
	// Since we'll rarely hit this code path, the overhead of defer is acceptable.
	defer g.edgesMu.Unlock()

	if e, ok := g.edges[string(key)]; ok {
		// Someone beat us to the punch.
		return e
	}

	e := newEdge(g.logger, g.meter, req, direction)
	g.edges[string(key)] = e
	return e
}

// An edge is a collection of RPC stats for a particular
// caller-callee-encoding-procedure-sk-rd-rk edge in the service graph.
type edge struct {
	logger *zap.Logger

	calls          *metrics.Counter
	successes      *metrics.Counter
	callerFailures *metrics.CounterVector
	serverFailures *metrics.CounterVector

	latencies          *metrics.Histogram
	callerErrLatencies *metrics.Histogram
	serverErrLatencies *metrics.Histogram
}

// newEdge constructs a new edge. Since Registries enforce metric uniqueness,
// edges should be cached and re-used for each RPC.
func newEdge(logger *zap.Logger, meter *metrics.Scope, req *transport.Request, direction string) *edge {
	tags := metrics.Tags{
		"source":           req.Caller,
		"dest":             req.Service,
		"transport":        unknownIfEmpty(req.Transport),
		"procedure":        req.Procedure,
		"encoding":         string(req.Encoding),
		"routing_key":      req.RoutingKey,
		"routing_delegate": req.RoutingDelegate,
		"direction":        direction,
	}
	calls, err := meter.Counter(metrics.Spec{
		Name:      "calls",
		Help:      "Total number of RPCs.",
		ConstTags: tags,
	})
	if err != nil {
		logger.Error("Failed to create calls counter.", zap.Error(err))
	}
	successes, err := meter.Counter(metrics.Spec{
		Name:      "successes",
		Help:      "Number of successful RPCs.",
		ConstTags: tags,
	})
	if err != nil {
		logger.Error("Failed to create successes counter.", zap.Error(err))
	}
	callerFailures, err := meter.CounterVector(metrics.Spec{
		Name:      "caller_failures",
		Help:      "Number of RPCs failed because of caller error.",
		ConstTags: tags,
		VarTags:   []string{_error},
	})
	if err != nil {
		logger.Error("Failed to create caller failures vector.", zap.Error(err))
	}
	serverFailures, err := meter.CounterVector(metrics.Spec{
		Name:      "server_failures",
		Help:      "Number of RPCs failed because of server error.",
		ConstTags: tags,
		VarTags:   []string{_error},
	})
	if err != nil {
		logger.Error("Failed to create server failures vector.", zap.Error(err))
	}
	latencies, err := meter.Histogram(metrics.HistogramSpec{
		Spec: metrics.Spec{
			Name:      "success_latency_ms",
			Help:      "Latency distribution of successful RPCs.",
			ConstTags: tags,
		},
		Unit:    time.Millisecond,
		Buckets: _bucketsMs,
	})
	if err != nil {
		logger.Error("Failed to create success latency distribution.", zap.Error(err))
	}
	callerErrLatencies, err := meter.Histogram(metrics.HistogramSpec{
		Spec: metrics.Spec{
			Name:      "caller_failure_latency_ms",
			Help:      "Latency distribution of RPCs failed because of caller error.",
			ConstTags: tags,
		},
		Unit:    time.Millisecond,
		Buckets: _bucketsMs,
	})
	if err != nil {
		logger.Error("Failed to create caller failure latency distribution.", zap.Error(err))
	}
	serverErrLatencies, err := meter.Histogram(metrics.HistogramSpec{
		Spec: metrics.Spec{
			Name:      "server_failure_latency_ms",
			Help:      "Latency distribution of RPCs failed because of server error.",
			ConstTags: tags,
		},
		Unit:    time.Millisecond,
		Buckets: _bucketsMs,
	})
	if err != nil {
		logger.Error("Failed to create server failure latency distribution.", zap.Error(err))
	}
	logger = logger.With(
		zap.String("source", req.Caller),
		zap.String("dest", req.Service),
		zap.String("transport", unknownIfEmpty(req.Transport)),
		zap.String("procedure", req.Procedure),
		zap.String("encoding", string(req.Encoding)),
		zap.String("routingKey", req.RoutingKey),
		zap.String("routingDelegate", req.RoutingDelegate),
		zap.String("direction", direction),
	)
	return &edge{
		logger:             logger,
		calls:              calls,
		successes:          successes,
		callerFailures:     callerFailures,
		serverFailures:     serverFailures,
		latencies:          latencies,
		callerErrLatencies: callerErrLatencies,
		serverErrLatencies: serverErrLatencies,
	}
}

// unknownIfEmpty works around hard-coded default value of "default" in go.uber.org/net/metrics
func unknownIfEmpty(t string) string {
	if t == "" {
		t = "unknown"
	}
	return t
}
