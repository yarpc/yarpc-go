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

package observability

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// observedBuckets returns, for each histogram, the upper bound of the bucket of
// each observation.
func observedBuckets(root *metrics.Root) map[string][]int64 {
	out := map[string][]int64{}
	for _, h := range root.Snapshot().Histograms {
		out[h.Name] = append(out[h.Name], h.Values...)
	}
	return out
}

func TestDefaultBuckets(t *testing.T) {
	root := metrics.New()
	g := newGraph(root.Scope(), zap.NewNop(), NewNopContextExtractor(), nil, nil)
	now := time.Now()
	defer stubTimeWithTimeVal(now)()
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(time.Hour))
	defer cancel()

	g.begin(ctx, transport.Unary, _directionInbound, procedureRequest("p")).
		endStats(5*time.Millisecond, callResult{requestSize: 200, responseSize: 3})

	got := observedBuckets(root)
	assert.Equal(t, []int64{5}, got["success_latency_ms"], "1ms steps up to 10ms")
	assert.Equal(t, []int64{256}, got["request_payload_size_bytes"], "powers of two")
	assert.Equal(t, []int64{4}, got["response_payload_size_bytes"])
	assert.Equal(t, []int64{math.MaxInt64}, got["ttl_ms"], "an hour is above the last default bucket")
}

func TestConfiguredBuckets(t *testing.T) {
	root := metrics.New()
	g := newGraph(root.Scope(), zap.NewNop(), NewNopContextExtractor(), nil, nil)
	g.setBuckets([]int64{10, 100, 1000}, []int64{0, 512, 4096})

	now := time.Now()
	defer stubTimeWithTimeVal(now)() // so that the ttl below is exactly 50ms
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(50*time.Millisecond))
	defer cancel()
	req := procedureRequest("p")
	call := func(elapsed time.Duration, res callResult) {
		g.begin(ctx, transport.Unary, _directionInbound, req).endStats(elapsed, res)
	}

	call(5*time.Millisecond, callResult{requestSize: 200, responseSize: 5000})
	call(2*time.Second, callResult{err: yarpcerrors.InvalidArgumentErrorf("bad"), requestSize: 0})
	call(500*time.Millisecond, callResult{err: yarpcerrors.DeadlineExceededErrorf("slow"), requestSize: 4096})

	got := observedBuckets(root)
	assert.Equal(t, []int64{10}, got["success_latency_ms"])
	assert.Equal(t, []int64{math.MaxInt64}, got["caller_failure_latency_ms"], "above the last bound")
	assert.Equal(t, []int64{1000}, got["server_failure_latency_ms"])
	assert.Equal(t, []int64{100, 100, 100}, got["ttl_ms"], "a 50ms deadline, three times")
	assert.Equal(t, []int64{100}, got["timeout_ttl_ms"])
	assert.ElementsMatch(t, []int64{512, 0, 4096}, got["request_payload_size_bytes"])
	assert.Equal(t, []int64{math.MaxInt64}, got["response_payload_size_bytes"])
}

func TestConfiguredBucketsStreams(t *testing.T) {
	root := metrics.New()
	g := newGraph(root.Scope(), zap.NewNop(), NewNopContextExtractor(), nil, nil)
	g.setBuckets([]int64{10, 100}, []int64{1, 1000})

	c := g.begin(context.Background(), transport.Streaming, _directionInbound, procedureRequest("stream"))
	c.EndStreamHandshake()
	c.edge.streaming.streamRequestPayloadSizes.IncBucket(600)
	c.edge.streaming.streamResponsePayloadSizes.IncBucket(5)
	c.EndStream(nil)

	got := observedBuckets(root)
	assert.Equal(t, []int64{10}, got["stream_duration_ms"], "a stream that took no time")
	assert.Equal(t, []int64{1000}, got["stream_request_payload_size_bytes"])
	assert.Equal(t, []int64{1000}, got["stream_response_payload_size_bytes"])
}

func TestConfiguredBucketsKeepTheOtherKindDefault(t *testing.T) {
	g := newGraph(metrics.New().Scope(), zap.NewNop(), nil, nil, nil)

	g.setBuckets([]int64{10, 100}, nil)
	assert.Equal(t, []int64{10, 100}, g.buckets.latencyMs)
	assert.Equal(t, _bucketsBytes, g.buckets.payloadBytes)

	g = newGraph(metrics.New().Scope(), zap.NewNop(), nil, nil, nil)
	g.setBuckets(nil, []int64{1, 2})
	assert.Equal(t, _bucketsMs, g.buckets.latencyMs)
	assert.Equal(t, []int64{1, 2}, g.buckets.payloadBytes)
}

func TestSetBucketsDoesNotAliasTheCallersSlice(t *testing.T) {
	g := newGraph(metrics.New().Scope(), zap.NewNop(), nil, nil, nil)
	latency := []int64{10, 100}
	g.setBuckets(latency, nil)
	latency[0] = 99999
	assert.Equal(t, []int64{10, 100}, g.buckets.latencyMs)
}

func TestSetBucketsEdgeCases(t *testing.T) {
	tests := []struct {
		desc    string
		give    []int64
		invalid bool
	}{
		{desc: "one bucket", give: []int64{100}},
		{desc: "starts at zero", give: []int64{0, 1}},
		{desc: "explicit catch-all", give: []int64{10, math.MaxInt64}},
		{desc: "not sorted", give: []int64{10, 5}, invalid: true},
		{desc: "duplicate", give: []int64{10, 10}, invalid: true},
		{desc: "negative", give: []int64{-1, 10}, invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			core, logs := observer.New(zap.WarnLevel)
			g := newGraph(metrics.New().Scope(), zap.New(core), nil, nil, nil)
			g.setBuckets(tt.give, nil)

			if tt.invalid {
				assert.Equal(t, _bucketsMs, g.buckets.latencyMs, "falls back to the default")
				require.Equal(t, 1, logs.Len())
				assert.Equal(t, "latencyBucketsMs", logs.All()[0].ContextMap()["config"])
			} else {
				assert.Equal(t, tt.give, g.buckets.latencyMs)
				assert.Zero(t, logs.Len())
			}
		})
	}
}

// An invalid list must leave the service with the default metrics, not none:
// go.uber.org/net/metrics refuses to create a histogram with such buckets.
func TestInvalidBucketsStillEmitMetrics(t *testing.T) {
	root := metrics.New()
	core, logs := observer.New(zap.WarnLevel)
	mw := NewMiddleware(Config{
		Scope:                   root.Scope(),
		Logger:                  zap.New(core),
		ContextExtractor:        NewNopContextExtractor(),
		LatencyBucketsMs:        []int64{100, 10},
		PayloadSizeBucketsBytes: []int64{1, 1},
	})

	mw.graph.begin(context.Background(), transport.Unary, _directionInbound, procedureRequest("p")).
		endStats(time.Millisecond, callResult{requestSize: 1, responseSize: 1})

	assert.Equal(t, 2, logs.Len(), "one warning for each list")
	got := observedBuckets(root)
	assert.Equal(t, []int64{1}, got["success_latency_ms"])
	assert.Equal(t, []int64{1}, got["request_payload_size_bytes"])
}

func TestMiddlewareConfigBuckets(t *testing.T) {
	mw := NewMiddleware(Config{
		Scope:                   metrics.New().Scope(),
		Logger:                  zap.NewNop(),
		LatencyBucketsMs:        []int64{1, 2, 3},
		PayloadSizeBucketsBytes: []int64{4, 5},
	})
	assert.Equal(t, []int64{1, 2, 3}, mw.graph.buckets.latencyMs)
	assert.Equal(t, []int64{4, 5}, mw.graph.buckets.payloadBytes)
}

// Overflow edges are made like any edge, so they use the configured buckets,
// which also makes them register the same histograms as the edges they stand
// in for.
func TestConfiguredBucketsApplyToOverflowEdges(t *testing.T) {
	root := metrics.New()
	core, logs := observer.New(zap.ErrorLevel)
	g := newGraph(root.Scope(), zap.New(core), NewNopContextExtractor(), nil, nil)
	g.setBuckets([]int64{10, 100}, []int64{1, 1000})
	g.setMaxEdges(1)

	for _, procedure := range []string{"a", "b", "c"} {
		g.begin(context.Background(), transport.Unary, _directionInbound, procedureRequest(procedure)).
			endStats(5*time.Millisecond, callResult{})
	}

	assert.Zero(t, logs.Len(), "%v", logs.All())
	assert.Equal(t, []int64{10, 10, 10}, observedBuckets(root)["success_latency_ms"])
}
