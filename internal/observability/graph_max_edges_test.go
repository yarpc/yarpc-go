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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func procedureRequest(procedure string) *transport.Request {
	return &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  "raw",
		Procedure: procedure,
	}
}

func overflowTags(dest, transport, direction, rpcType string) metrics.Tags {
	return metrics.Tags{
		_source:          _overflowTagValue,
		_dest:            dest,
		_transport:       transport,
		_encoding:        _overflowTagValue,
		_procedure:       _overflowTagValue,
		_routingKey:      _overflowTagValue,
		_routingDelegate: _overflowTagValue,
		_direction:       direction,
		_rpcType:         rpcType,
	}
}

// callsByProcedure returns the value of the calls counter of each edge, by
// the procedure tag.
func callsByProcedure(root *metrics.Root) map[string]int64 {
	calls := map[string]int64{}
	for _, c := range root.Snapshot().Counters {
		if c.Name == "calls" {
			calls[c.Tags[_procedure]] += c.Value
		}
	}
	return calls
}

func TestMaxEdgesOverflow(t *testing.T) {
	root := metrics.New()
	core, logs := observer.New(zap.WarnLevel)
	g := newGraph(root.Scope(), zap.New(core), NewNopContextExtractor(), nil, nil)
	g.setMaxEdges(2)

	call := func(procedure string) {
		c := g.begin(context.Background(), transport.Unary, _directionInbound, procedureRequest(procedure))
		c.endStats(time.Millisecond, callResult{})
	}

	call("p1")
	call("p2")
	assert.Zero(t, logs.Len(), "nothing overflowed yet")

	// Everything past the limit is recorded together.
	call("p3")
	call("p4")
	call("p3")
	// And edges that exist are still recorded on their own.
	call("p1")
	call("p1")

	assert.Equal(t, map[string]int64{
		"p1":              3,
		"p2":              1,
		_overflowTagValue: 3,
	}, callsByProcedure(root))
	assert.Equal(t, 2, g.edges.len(), "overflow edges do not count toward the limit")
	assert.Equal(t, 1, g.overflow.len())

	var tags []metrics.Tags
	for _, c := range root.Snapshot().Counters {
		if c.Name == "calls" && c.Tags[_procedure] == _overflowTagValue {
			tags = append(tags, c.Tags)
		}
	}
	assert.Equal(t, []metrics.Tags{overflowTags("service", "unknown", "inbound", "Unary")}, tags,
		"destination, transport, direction and RPC type are kept, everything else is replaced")

	require.Equal(t, 1, logs.Len(), "must warn once, not for every overflowed call")
	entry := logs.All()[0]
	assert.Equal(t, zapcore.WarnLevel, entry.Level)
	assert.Contains(t, entry.Message, "maximum number of YARPC metric edges")
	assert.EqualValues(t, 2, entry.ContextMap()["maxEdges"])
}

func TestMaxEdgesOverflowEdgesAreSeparateByDirectionAndRPCType(t *testing.T) {
	root := metrics.New()
	g := newGraph(root.Scope(), zap.NewNop(), NewNopContextExtractor(), nil, nil)
	g.setMaxEdges(1)

	g.begin(context.Background(), transport.Unary, _directionInbound, procedureRequest("owns-the-only-edge")).
		endStats(time.Millisecond, callResult{})

	for _, tt := range []struct {
		direction directionName
		rpcType   transport.Type
	}{
		{_directionInbound, transport.Unary},
		{_directionInbound, transport.Oneway},
		{_directionOutbound, transport.Unary},
		{_directionOutbound, transport.Oneway},
	} {
		for i := 0; i < 2; i++ {
			c := g.begin(context.Background(), transport.Unary, tt.direction, procedureRequest(fmt.Sprint("p", i)))
			if tt.rpcType == transport.Oneway {
				c = g.begin(context.Background(), transport.Oneway, tt.direction, procedureRequest(fmt.Sprint("p", i)))
			}
			c.endStats(time.Millisecond, callResult{})
		}
	}

	seen := map[string]int64{}
	for _, c := range root.Snapshot().Counters {
		if c.Name == "calls" && c.Tags[_procedure] == _overflowTagValue {
			seen[c.Tags[_direction]+"/"+c.Tags[_rpcType]] += c.Value
		}
	}
	assert.Equal(t, map[string]int64{
		"inbound/Unary":   2,
		"inbound/Oneway":  2,
		"outbound/Unary":  2,
		"outbound/Oneway": 2,
	}, seen)
}

func TestMaxEdgesOverflowStream(t *testing.T) {
	root := metrics.New()
	g := newGraph(root.Scope(), zap.NewNop(), NewNopContextExtractor(), nil, nil)
	g.setMaxEdges(1)

	g.begin(context.Background(), transport.Unary, _directionInbound, procedureRequest("owns-the-only-edge")).
		endStats(time.Millisecond, callResult{})

	// The overflow edge of a stream must have the stream metrics, not the
	// unary ones.
	c := g.begin(context.Background(), transport.Streaming, _directionInbound, procedureRequest("stream"))
	c.EndStreamHandshake()
	c.EndStream(nil)

	var gauges []string
	for _, gauge := range root.Snapshot().Gauges {
		if gauge.Tags[_procedure] == _overflowTagValue {
			gauges = append(gauges, gauge.Name)
		}
	}
	assert.Equal(t, []string{"streams_active"}, gauges)
	assert.Contains(t, histogramNames(root), "stream_duration_ms")
}

func TestMaxEdgesZeroIsUnlimited(t *testing.T) {
	root := metrics.New()
	core, logs := observer.New(zap.WarnLevel)
	mw := NewMiddleware(Config{Scope: root.Scope(), Logger: zap.New(core)}) // MaxEdges: 0

	for i := 0; i < 300; i++ {
		req := procedureRequest(fmt.Sprint("p", i))
		require.NoError(t, mw.Handle(context.Background(), req, &transporttest.FakeResponseWriter{}, &fakeHandler{}))
	}

	assert.Len(t, callsByProcedure(root), 300)
	assert.Zero(t, logs.Len())
}

func TestMaxEdgesWithTagsBlocklist(t *testing.T) {
	root := metrics.New()
	core, logs := observer.New(zap.WarnLevel)
	mw := NewMiddleware(Config{
		Scope:               root.Scope(),
		Logger:              zap.New(core),
		MetricTagsBlocklist: []string{"routing_delegate"},
		MaxEdges:            1,
	})

	// The blocklist collapses all of these into one edge, which fits.
	for i := 0; i < 100; i++ {
		req := procedureRequest("p")
		req.RoutingDelegate = fmt.Sprint("dispatch-city-", i)
		require.NoError(t, mw.Handle(context.Background(), req, &transporttest.FakeResponseWriter{}, &fakeHandler{}))
	}

	assert.Equal(t, map[string]int64{"p": 100}, callsByProcedure(root))
	assert.Zero(t, logs.Len(), "the limit was not reached")
}

// Logs of calls that overflow must still say which request they are for, even
// though their metrics are shared.
func TestMaxEdgesOverflowLogsDescribeTheRequest(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	mw := NewMiddleware(Config{
		Scope:            metrics.New().Scope(),
		Logger:           zap.New(core),
		ContextExtractor: NewNopContextExtractor(),
		MaxEdges:         1,
	})

	for _, procedure := range []string{"owns-the-only-edge", "overflowed"} {
		req := procedureRequest(procedure)
		req.Caller = "caller-of-" + procedure
		require.NoError(t, mw.Handle(context.Background(), req, &transporttest.FakeResponseWriter{}, &fakeHandler{}))
	}

	var served []map[string]interface{}
	for _, e := range logs.All() {
		if strings.HasPrefix(e.Message, "Handled") {
			served = append(served, e.ContextMap())
		}
	}
	require.Len(t, served, 2)
	for i, procedure := range []string{"owns-the-only-edge", "overflowed"} {
		assert.Equal(t, procedure, served[i]["procedure"])
		assert.Equal(t, "caller-of-"+procedure, served[i]["source"])
		assert.Equal(t, "service", served[i]["dest"])
		assert.Equal(t, "inbound", served[i]["direction"])
	}

	for _, e := range logs.All() {
		for k, v := range e.ContextMap() {
			assert.NotEqual(t, _overflowTagValue, v, "field %q of %q must be the real value", k, e.Message)
		}
	}
}

// BenchmarkGraphCallOverflow is BenchmarkGraphCallManyEdges with a limit that
// most of the edges do not fit in.
func BenchmarkGraphCallOverflow(b *testing.B) {
	const edges = 1024

	g := newGraph(metrics.New().Scope(), zap.NewNop(), nil, nil, nil)
	g.setMaxEdges(16)
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
		i := uint32(len(reqs)) * 7919
		for pb.Next() {
			i = i*1664525 + 1013904223
			c := g.begin(ctx, transport.Unary, _directionInbound, reqs[(i>>8)%edges])
			c.endStats(3*time.Millisecond, res)
		}
	})
}

// Overflow edges are kept apart by destination and transport, which are few
// and worth telling apart.
func TestMaxEdgesOverflowIsPerDestinationAndTransport(t *testing.T) {
	root := metrics.New()
	g := newGraph(root.Scope(), zap.NewNop(), NewNopContextExtractor(), nil, nil)
	g.setMaxEdges(100)

	// Fill the limit with edges on an unrelated destination.
	for i := 0; i < 100; i++ {
		req := procedureRequest(fmt.Sprint("p", i))
		req.Service = "filler"
		g.begin(context.Background(), transport.Unary, _directionOutbound, req).endStats(time.Millisecond, callResult{})
	}

	for _, dest := range []string{"a", "b"} {
		for _, tr := range []string{"tchannel", "http"} {
			for i := 0; i < 3; i++ { // different procedures, same overflow edge
				req := procedureRequest(fmt.Sprint("p", i))
				req.Service, req.Transport = dest, tr
				g.begin(context.Background(), transport.Unary, _directionOutbound, req).endStats(time.Millisecond, callResult{})
			}
		}
	}

	got := map[string]int64{}
	for _, c := range root.Snapshot().Counters {
		if c.Name == "calls" && c.Tags[_procedure] == _overflowTagValue {
			assert.Equal(t, _overflowTagValue, c.Tags[_source])
			assert.Equal(t, _overflowTagValue, c.Tags[_routingDelegate])
			got[c.Tags[_dest]+"/"+c.Tags[_transport]] += c.Value
		}
	}
	assert.Equal(t, map[string]int64{"a/tchannel": 3, "a/http": 3, "b/tchannel": 3, "b/http": 3}, got)
	assert.Equal(t, 4, g.overflow.len())
}

// The overflow edges are limited too, so that a destination or transport with
// many values cannot grow the edges without bound.
func TestMaxEdgesOverflowEdgesAreLimited(t *testing.T) {
	root := metrics.New()
	g := newGraph(root.Scope(), zap.NewNop(), NewNopContextExtractor(), nil, nil)
	g.setMaxEdges(2)

	call := func(dest string) {
		req := procedureRequest("p")
		req.Service = dest
		g.begin(context.Background(), transport.Unary, _directionOutbound, req).endStats(time.Millisecond, callResult{})
	}

	call("d1") // edge
	call("d2") // edge
	call("d3") // overflow edge
	call("d4") // overflow edge
	call("d5") // no room: collapsed
	call("d6") // no room: collapsed
	call("d3") // still its own overflow edge

	got := map[string]int64{}
	for _, c := range root.Snapshot().Counters {
		if c.Name == "calls" {
			got[c.Tags[_procedure]+"/"+c.Tags[_dest]] += c.Value
		}
	}
	assert.Equal(t, map[string]int64{
		"p/d1":                    1,
		"p/d2":                    1,
		_overflowTagValue + "/d3": 2,
		_overflowTagValue + "/d4": 1,
		_overflowTagValue + "/" + _overflowTagValue: 2,
	}, got)
	assert.Equal(t, 2, g.edges.len())
	assert.Equal(t, 2, g.overflow.len())
}

// A blocklisted tag is not a reason to have more overflow edges, which would
// only have the same metrics registered twice.
func TestMaxEdgesOverflowHonoursTagsBlocklist(t *testing.T) {
	root := metrics.New()
	core, logs := observer.New(zap.ErrorLevel)
	g := newGraph(root.Scope(), zap.New(core), NewNopContextExtractor(),
		[]string{"dest", "transport", "direction", "rpc_type"}, nil)
	g.setMaxEdges(1)

	g.begin(context.Background(), transport.Unary, _directionInbound, procedureRequest("owns-the-only-edge")).
		endStats(time.Millisecond, callResult{})

	for _, dest := range []string{"a", "b", "c"} {
		for _, direction := range []directionName{_directionInbound, _directionOutbound} {
			req := procedureRequest("p")
			req.Service = dest
			g.begin(context.Background(), transport.Unary, direction, req).endStats(time.Millisecond, callResult{})
		}
	}

	assert.Zero(t, logs.Len(), "no metric may be registered twice: %v", logs.All())
	assert.Equal(t, 1, g.overflow.len(), "the blocklisted tags no longer tell calls apart")
	assert.Equal(t, map[string]int64{"owns-the-only-edge": 1, _overflowTagValue: 6}, callsByProcedure(root))
}
