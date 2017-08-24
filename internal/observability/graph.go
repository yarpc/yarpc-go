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

package observability

import (
	"context"
	"sync"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/pally"
	"go.uber.org/zap"
)

var (
	_timeNow          = time.Now // for tests
	_defaultGraphSize = 128
	_digesterPool     = sync.Pool{New: func() interface{} {
		return &digester{make([]byte, 0, 128)}
	}}
	// Latency buckets for histograms. At some point, we may want to make these
	// configurable.
	_ms      = time.Millisecond
	_buckets = []time.Duration{
		1 * _ms,
		2 * _ms,
		3 * _ms,
		4 * _ms,
		5 * _ms,
		6 * _ms,
		7 * _ms,
		8 * _ms,
		9 * _ms,
		10 * _ms,
		12 * _ms,
		14 * _ms,
		16 * _ms,
		18 * _ms,
		20 * _ms,
		25 * _ms,
		30 * _ms,
		35 * _ms,
		40 * _ms,
		45 * _ms,
		50 * _ms,
		60 * _ms,
		70 * _ms,
		80 * _ms,
		90 * _ms,
		100 * _ms,
		120 * _ms,
		140 * _ms,
		160 * _ms,
		180 * _ms,
		200 * _ms,
		250 * _ms,
		300 * _ms,
		350 * _ms,
		400 * _ms,
		450 * _ms,
		500 * _ms,
		600 * _ms,
		700 * _ms,
		800 * _ms,
		900 * _ms,
		1000 * _ms,
		1500 * _ms,
		2000 * _ms,
		2500 * _ms,
		3000 * _ms,
		4000 * _ms,
		5000 * _ms,
		7500 * _ms,
		10000 * _ms,
	}
)

// A digester creates a null-delimited byte slice from a series of strings. It's
// an efficient way to create map keys.
//
// This helps because (1) appending to a string allocates and (2) converting a
// byte slice to a string allocates, but (3) the Go compiler optimizes away
// byte-to-string conversions in map lookups. Using this type to build up a key
// and doing map lookups with myMap[string(d.digest())] is fast and
// zero-allocation.
type digester struct {
	bs []byte
}

// For optimal performance, be sure to free each digester.
func newDigester() *digester {
	d := _digesterPool.Get().(*digester)
	d.bs = d.bs[:0]
	return d
}

func (d *digester) add(s string) {
	if len(d.bs) > 0 {
		// separate labels with a null byte
		d.bs = append(d.bs, '\x00')
	}
	d.bs = append(d.bs, s...)
}

func (d *digester) digest() []byte {
	return d.bs
}

func (d *digester) free() {
	_digesterPool.Put(d)
}

// A graph represents a collection of services: each service is a node, and we
// collect stats for each caller-callee-encoding-procedure-rk-sk-rd edge.
type graph struct {
	reg     *pally.Registry
	logger  *zap.Logger
	extract ContextExtractor

	edgesMu sync.RWMutex
	edges   map[string]*edge
}

func newGraph(reg *pally.Registry, logger *zap.Logger, extract ContextExtractor) graph {
	return graph{
		edges:   make(map[string]*edge, _defaultGraphSize),
		reg:     reg,
		logger:  logger,
		extract: extract,
	}
}

// begin starts a call along an edge.
func (g *graph) begin(ctx context.Context, rpcType transport.Type, isInbound bool, req *transport.Request) call {
	now := _timeNow()

	key, free := getKey(req)
	e := g.getOrCreateEdge(key, req)
	free()

	return call{
		edge:    e,
		extract: g.extract,
		started: now,
		ctx:     ctx,
		req:     req,
		rpcType: rpcType,
		inbound: isInbound,
	}
}

func getKey(req *transport.Request) (key []byte, free func()) {
	d := newDigester()
	d.add(req.Caller)
	d.add(req.Service)
	d.add(string(req.Encoding))
	d.add(req.Procedure)
	d.add(req.RoutingKey)
	d.add(req.RoutingDelegate)
	return d.digest(), d.free
}

func (g *graph) getOrCreateEdge(key []byte, req *transport.Request) *edge {
	if e := g.getEdge(key); e != nil {
		return e
	}
	return g.createEdge(key, req)
}

func (g *graph) getEdge(key []byte) *edge {
	g.edgesMu.RLock()
	e := g.edges[string(key)]
	g.edgesMu.RUnlock()
	return e
}

func (g *graph) createEdge(key []byte, req *transport.Request) *edge {
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

// An edge is a collection of RPC stats for a particular
// caller-callee-encoding-procedure-sk-rd-rk edge in the service graph.
type edge struct {
	logger *zap.Logger

	calls          pally.Counter
	successes      pally.Counter
	callerFailures pally.CounterVector
	serverFailures pally.CounterVector

	latencies          pally.Latencies
	callerErrLatencies pally.Latencies
	serverErrLatencies pally.Latencies
}

// newEdge constructs a new edge. Since Registries enforce metric uniqueness,
// edges should be cached and re-used for each RPC.
func newEdge(logger *zap.Logger, reg *pally.Registry, req *transport.Request) *edge {
	labels := pally.Labels{
		"source":           pally.ScrubLabelValue(req.Caller),
		"dest":             pally.ScrubLabelValue(req.Service),
		"procedure":        pally.ScrubLabelValue(req.Procedure),
		"encoding":         pally.ScrubLabelValue(string(req.Encoding)),
		"routing_key":      pally.ScrubLabelValue(req.RoutingKey),
		"routing_delegate": pally.ScrubLabelValue(req.RoutingDelegate),
	}
	calls, err := reg.NewCounter(pally.Opts{
		Name:        "calls",
		Help:        "Total number of RPCs.",
		ConstLabels: labels,
	})
	if err != nil {
		logger.Error("Failed to create calls counter.", zap.Error(err))
		calls = pally.NewNopCounter()
	}
	successes, err := reg.NewCounter(pally.Opts{
		Name:        "successes",
		Help:        "Number of successful RPCs.",
		ConstLabels: labels,
	})
	if err != nil {
		logger.Error("Failed to create successes counter.", zap.Error(err))
		successes = pally.NewNopCounter()
	}
	callerFailures, err := reg.NewCounterVector(pally.Opts{
		Name:           "caller_failures",
		Help:           "Number of RPCs failed because of caller error.",
		ConstLabels:    labels,
		VariableLabels: []string{"error"},
	})
	if err != nil {
		logger.Error("Failed to create caller failures vector.", zap.Error(err))
		callerFailures = pally.NewNopCounterVector()
	}
	serverFailures, err := reg.NewCounterVector(pally.Opts{
		Name:           "server_failures",
		Help:           "Number of RPCs failed because of server error.",
		ConstLabels:    labels,
		VariableLabels: []string{"error"},
	})
	if err != nil {
		logger.Error("Failed to create server failures vector.", zap.Error(err))
		serverFailures = pally.NewNopCounterVector()
	}
	latencies, err := reg.NewLatencies(pally.LatencyOpts{
		Opts: pally.Opts{
			Name:        "success_latency_ms",
			Help:        "Latency distribution of successful RPCs.",
			ConstLabels: labels,
		},
		Unit:    _ms,
		Buckets: _buckets,
	})
	if err != nil {
		logger.Error("Failed to create success latency distribution.", zap.Error(err))
		latencies = pally.NewNopLatencies()
	}
	callerErrLatencies, err := reg.NewLatencies(pally.LatencyOpts{
		Opts: pally.Opts{
			Name:        "caller_failure_latency_ms",
			Help:        "Latency distribution of RPCs failed because of caller error.",
			ConstLabels: labels,
		},
		Unit:    _ms,
		Buckets: _buckets,
	})
	if err != nil {
		logger.Error("Failed to create caller failure latency distribution.", zap.Error(err))
		callerErrLatencies = pally.NewNopLatencies()
	}
	serverErrLatencies, err := reg.NewLatencies(pally.LatencyOpts{
		Opts: pally.Opts{
			Name:        "server_failure_latency_ms",
			Help:        "Latency distribution of RPCs failed because of server error.",
			ConstLabels: labels,
		},
		Unit:    _ms,
		Buckets: _buckets,
	})
	if err != nil {
		logger.Error("Failed to create server failure latency distribution.", zap.Error(err))
		serverErrLatencies = pally.NewNopLatencies()
	}
	logger = logger.With(
		zap.String("source", req.Caller),
		zap.String("dest", req.Service),
		zap.String("procedure", req.Procedure),
		zap.String("encoding", string(req.Encoding)),
		zap.String("routingKey", req.RoutingKey),
		zap.String("routingDelegate", req.RoutingDelegate),
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
