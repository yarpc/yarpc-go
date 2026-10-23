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
	"fmt"
	"testing"
	"time"

	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

func benchGraphRequest() *transport.Request {
	return &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Encoding:        "thrift",
		Procedure:       "Svc::method",
		RoutingKey:      "rk",
		RoutingDelegate: "rd",
	}
}

// BenchmarkGraphBegin measures finding the edge of a hot call from many cores.
func BenchmarkGraphBegin(b *testing.B) {
	g := newGraph(metrics.New().Scope(), zap.NewNop(), nil, nil)
	req := benchGraphRequest()
	ctx := context.Background()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.begin(ctx, transport.Unary, _directionInbound, req)
		}
	})
}

// BenchmarkGraphCall measures the graph's share of one successful unary call,
// from finding the edge to recording its stats, on one hot edge from many
// cores.
func BenchmarkGraphCall(b *testing.B) {
	g := newGraph(metrics.New().Scope(), zap.NewNop(), nil, nil)
	req := benchGraphRequest()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res := callResult{requestSize: 200, responseSize: 1500}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := g.begin(ctx, transport.Unary, _directionInbound, req)
			c.endStats(3*time.Millisecond, res)
		}
	})
}

// BenchmarkGraphCallManyEdges is like BenchmarkGraphCall but spreads calls
// over many edges, as a service with many routing delegates or procedures
// does. The stats of different edges do not contend with each other, but the
// edge lookup is shared by all of them.
func BenchmarkGraphCallManyEdges(b *testing.B) {
	const edges = 1024

	g := newGraph(metrics.New().Scope(), zap.NewNop(), nil, nil)
	reqs := make([]*transport.Request, edges)
	for i := range reqs {
		reqs[i] = benchGraphRequest()
		reqs[i].RoutingDelegate = fmt.Sprintf("dispatch-city-%d", i)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res := callResult{requestSize: 200, responseSize: 1500}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		// A cheap per-goroutine sequence, so that no shared state is touched
		// to pick the next edge.
		i := uint32(len(reqs)) * 7919
		for pb.Next() {
			i = i*1664525 + 1013904223
			c := g.begin(ctx, transport.Unary, _directionInbound, reqs[(i>>8)%edges])
			c.endStats(3*time.Millisecond, res)
		}
	})
}
