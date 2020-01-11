// Copyright (c) 2020 Uber Technologies, Inc.
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

	inboundLevels, outboundLevels levels
}

func newGraph(meter *metrics.Scope, logger *zap.Logger, extract ContextExtractor) graph {
	return graph{
		edges:   make(map[string]*edge, _defaultGraphSize),
		meter:   meter,
		logger:  logger,
		extract: extract,
		inboundLevels: levels{
			success:          zapcore.DebugLevel,
			failure:          zapcore.ErrorLevel,
			applicationError: zapcore.ErrorLevel,
		},
		outboundLevels: levels{
			success:          zapcore.DebugLevel,
			failure:          zapcore.ErrorLevel,
			applicationError: zapcore.ErrorLevel,
		},
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
	d.Add(rpcType.String())
	e := g.getOrCreateEdge(d.Digest(), req, string(direction), rpcType)
	d.Free()

	levels := &g.inboundLevels
	if direction != _directionInbound {
		levels = &g.outboundLevels
	}

	return call{
		edge:      e,
		extract:   g.extract,
		started:   now,
		ctx:       ctx,
		req:       req,
		rpcType:   rpcType,
		direction: direction,
		levels:    levels,
	}
}

func (g *graph) getOrCreateEdge(key []byte, req *transport.Request, direction string, rpcType transport.Type) *edge {
	if e := g.getEdge(key); e != nil {
		return e
	}
	return g.createEdge(key, req, direction, rpcType)
}

func (g *graph) getEdge(key []byte) *edge {
	g.edgesMu.RLock()
	e := g.edges[string(key)]
	g.edgesMu.RUnlock()
	return e
}

func (g *graph) createEdge(key []byte, req *transport.Request, direction string, rpcType transport.Type) *edge {
	g.edgesMu.Lock()
	// Since we'll rarely hit this code path, the overhead of defer is acceptable.
	defer g.edgesMu.Unlock()

	if e, ok := g.edges[string(key)]; ok {
		// Someone beat us to the punch.
		return e
	}

	e := newEdge(g.logger, g.meter, req, direction, rpcType)
	g.edges[string(key)] = e
	return e
}

// An edge is a collection of RPC stats for a particular
// caller-callee-encoding-procedure-sk-rd-rk edge in the service graph.
type edge struct {
	logger *zap.Logger

	calls          *metrics.Counter
	successes      *metrics.Counter
	panics         *metrics.Counter
	callerFailures *metrics.CounterVector
	serverFailures *metrics.CounterVector

	latencies          *metrics.Histogram
	callerErrLatencies *metrics.Histogram
	serverErrLatencies *metrics.Histogram

	streaming *streamEdge
}

// streamEdge metrics should only be used for streaming requests.
type streamEdge struct {
	sends         *metrics.Counter
	sendSuccesses *metrics.Counter
	sendFailures  *metrics.CounterVector

	receives         *metrics.Counter
	receiveSuccesses *metrics.Counter
	receiveFailures  *metrics.CounterVector

	streamDurations *metrics.Histogram
	streamsActive   *metrics.Gauge
}

// newEdge constructs a new edge. Since Registries enforce metric uniqueness,
// edges should be cached and re-used for each RPC.
func newEdge(logger *zap.Logger, meter *metrics.Scope, req *transport.Request, direction string, rpcType transport.Type) *edge {
	tags := metrics.Tags{
		"source":           req.Caller,
		"dest":             req.Service,
		"transport":        unknownIfEmpty(req.Transport),
		"procedure":        req.Procedure,
		"encoding":         string(req.Encoding),
		"routing_key":      req.RoutingKey,
		"routing_delegate": req.RoutingDelegate,
		"direction":        direction,
		"rpc_type":         rpcType.String(),
	}

	// metrics for all RPCs
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
	panics, err := meter.Counter(metrics.Spec{
		Name:      "panics",
		Help:      "Number of RPCs failed because of panic.",
		ConstTags: tags,
	})
	if err != nil {
		logger.Error("Failed to create panics counter.", zap.Error(err))
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

	// metrics for only unary and oneway
	var latencies, callerErrLatencies, serverErrLatencies *metrics.Histogram
	if rpcType == transport.Unary || rpcType == transport.Oneway {
		latencies, err = meter.Histogram(metrics.HistogramSpec{
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
		callerErrLatencies, err = meter.Histogram(metrics.HistogramSpec{
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
		serverErrLatencies, err = meter.Histogram(metrics.HistogramSpec{
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
	}

	// metrics for only streams
	var streaming *streamEdge
	if rpcType == transport.Streaming {
		// sends
		sends, err := meter.Counter(metrics.Spec{
			Name:      "stream_sends",
			Help:      "Total number of streaming messages sent.",
			ConstTags: tags,
		})
		if err != nil {
			logger.DPanic("Failed to create streaming sends counter.", zap.Error(err))
		}
		sendSuccesses, err := meter.Counter(metrics.Spec{
			Name:      "stream_send_successes",
			Help:      "Number of successful streaming messages sent.",
			ConstTags: tags,
		})
		if err != nil {
			logger.DPanic("Failed to create streaming sends successes counter.", zap.Error(err))
		}
		sendFailures, err := meter.CounterVector(metrics.Spec{
			Name:      "stream_send_failures",
			Help:      "Number streaming messages that failed to send.",
			ConstTags: tags,
			VarTags:   []string{_error},
		})
		if err != nil {
			logger.DPanic("Failed to create streaming sends failure counter.", zap.Error(err))
		}

		// receives
		receives, err := meter.Counter(metrics.Spec{
			Name:      "stream_receives",
			Help:      "Total number of streaming messages recevied.",
			ConstTags: tags,
		})
		if err != nil {
			logger.DPanic("Failed to create streaming receives counter.", zap.Error(err))
		}
		receiveSuccesses, err := meter.Counter(metrics.Spec{
			Name:      "stream_receive_successes",
			Help:      "Number of successful streaming messages received.",
			ConstTags: tags,
		})
		if err != nil {
			logger.DPanic("Failed to create streaming receives successes counter.", zap.Error(err))
		}
		receiveFailures, err := meter.CounterVector(metrics.Spec{
			Name:      "stream_receive_failures",
			Help:      "Number streaming messages failed to be recieved.",
			ConstTags: tags,
			VarTags:   []string{_error},
		})
		if err != nil {
			logger.DPanic("Failed to create streaming receives failure counter.", zap.Error(err))
		}

		// entire stream
		streamDurations, err := meter.Histogram(metrics.HistogramSpec{
			Spec: metrics.Spec{
				Name:      "stream_duration_ms",
				Help:      "Latency distribution of total stream duration.",
				ConstTags: tags,
			},
			Unit:    time.Millisecond,
			Buckets: _bucketsMs,
		})
		if err != nil {
			logger.DPanic("Failed to create stream duration histogram.", zap.Error(err))
		}
		streamsActive, err := meter.Gauge(metrics.Spec{
			Name:      "streams_active",
			Help:      "Number of active streams.",
			ConstTags: tags,
		})
		if err != nil {
			logger.DPanic("Failed to create active stream gauge.", zap.Error(err))
		}

		streaming = &streamEdge{
			sends:            sends,
			sendSuccesses:    sendSuccesses,
			sendFailures:     sendFailures,
			receives:         receives,
			receiveSuccesses: receiveSuccesses,
			receiveFailures:  receiveFailures,
			streamDurations:  streamDurations,
			streamsActive:    streamsActive,
		}
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
		panics:             panics,
		callerFailures:     callerFailures,
		serverFailures:     serverFailures,
		latencies:          latencies,
		callerErrLatencies: callerErrLatencies,
		serverErrLatencies: serverErrLatencies,
		streaming:          streaming,
	}
}

// unknownIfEmpty works around hard-coded default value of "default" in go.uber.org/net/metrics
func unknownIfEmpty(t string) string {
	if t == "" {
		t = "unknown"
	}
	return t
}
