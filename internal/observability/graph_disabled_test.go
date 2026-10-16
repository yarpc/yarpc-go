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
	"sort"
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

// unaryHistograms are all the histograms of a unary edge.
var unaryHistograms = []string{
	"success_latency_ms",
	"caller_failure_latency_ms",
	"server_failure_latency_ms",
	"ttl_ms",
	"timeout_ttl_ms",
	"request_payload_size_bytes",
	"response_payload_size_bytes",
}

var streamHistograms = []string{
	"stream_duration_ms",
	"stream_request_payload_size_bytes",
	"stream_response_payload_size_bytes",
}

func histogramNames(root *metrics.Root) []string {
	var names []string
	for _, h := range root.Snapshot().Histograms {
		names = append(names, h.Name)
	}
	sort.Strings(names)
	return names
}

func sorted(names ...string) []string {
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out
}

func without(names []string, drop ...string) []string {
	var out []string
	for _, n := range names {
		keep := true
		for _, d := range drop {
			keep = keep && n != d
		}
		if keep {
			out = append(out, n)
		}
	}
	return sorted(out...)
}

// TestDisableableMetricsAreAllHistograms makes sure the documented list of
// names is the full set of histograms YARPC creates and nothing else.
func TestDisableableMetricsAreAllHistograms(t *testing.T) {
	assert.Equal(t,
		sorted(append(append([]string(nil), unaryHistograms...), streamHistograms...)...),
		sorted(DisableableMetrics...))
}

func TestDisabledMetricsUnary(t *testing.T) {
	tests := []struct {
		desc     string
		disabled []string
		want     []string
	}{
		{desc: "nothing disabled", want: sorted(unaryHistograms...)},
		{
			desc:     "ttl and payload size",
			disabled: []string{"ttl_ms", "timeout_ttl_ms", "request_payload_size_bytes", "response_payload_size_bytes"},
			want:     without(unaryHistograms, "ttl_ms", "timeout_ttl_ms", "request_payload_size_bytes", "response_payload_size_bytes"),
		},
		{
			desc:     "failure latencies",
			disabled: []string{"caller_failure_latency_ms", "server_failure_latency_ms"},
			want:     without(unaryHistograms, "caller_failure_latency_ms", "server_failure_latency_ms"),
		},
		{
			desc:     "success latency",
			disabled: []string{"success_latency_ms"},
			want:     without(unaryHistograms, "success_latency_ms"),
		},
		{
			desc:     "stream names do not affect unary edges",
			disabled: []string{"stream_duration_ms"},
			want:     sorted(unaryHistograms...),
		},
		{desc: "everything", disabled: unaryHistograms, want: nil},
		{desc: "duplicates", disabled: []string{"ttl_ms", "ttl_ms"}, want: without(unaryHistograms, "ttl_ms")},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			root := metrics.New()
			g := newGraph(root.Scope(), zap.NewNop(), nil, nil, tt.disabled)
			req := &transport.Request{Caller: "caller", Service: "service", Encoding: "raw", Procedure: "p"}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// Exercise every recording path: a success, a caller failure, a
			// server failure and a timeout. None may panic on a disabled
			// (nil) histogram.
			for _, res := range []callResult{
				{requestSize: 10, responseSize: 20},
				{err: yarpcerrors.InvalidArgumentErrorf("bad"), requestSize: 10},
				{err: yarpcerrors.InternalErrorf("oops"), requestSize: 10},
				{err: yarpcerrors.DeadlineExceededErrorf("slow"), requestSize: 10},
			} {
				c := g.begin(ctx, transport.Unary, _directionInbound, req)
				c.endStats(5*time.Millisecond, res)
			}

			assert.Equal(t, tt.want, histogramNames(root))

			// Counters keep working whatever is disabled.
			edge := g.getEdge(mustKey(req, string(_directionInbound), transport.Unary))
			require.NotNil(t, edge)
			assert.EqualValues(t, 4, edge.calls.Load())
			assert.EqualValues(t, 1, edge.successes.Load())
		})
	}
}

func TestDisabledMetricsStream(t *testing.T) {
	tests := []struct {
		desc     string
		disabled []string
		want     []string
	}{
		{desc: "nothing disabled", want: sorted(streamHistograms...)},
		{
			desc:     "payload sizes",
			disabled: []string{"stream_request_payload_size_bytes", "stream_response_payload_size_bytes"},
			want:     []string{"stream_duration_ms"},
		},
		{desc: "everything", disabled: streamHistograms, want: nil},
		{
			desc:     "unary names do not affect stream edges",
			disabled: []string{"ttl_ms"},
			want:     sorted(streamHistograms...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			root := metrics.New()
			g := newGraph(root.Scope(), zap.NewNop(), NewNopContextExtractor(), nil, tt.disabled)
			req := &transport.Request{Caller: "caller", Service: "service", Encoding: "raw", Procedure: "p"}

			c := g.begin(context.Background(), transport.Streaming, _directionInbound, req)
			c.EndStreamHandshake()
			c.EndStream(nil)

			assert.Equal(t, tt.want, histogramNames(root))
		})
	}
}

func TestDisabledMetricsUnknownName(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	g := newGraph(metrics.New().Scope(), zap.New(core), nil, nil,
		[]string{"ttl_ms", "no_such_metric", "calls"})

	assert.True(t, g.disabledMetrics.ttl, "known names still apply")
	entries := logs.All()
	require.Len(t, entries, 2, "one warning for each unknown name, including counters")
	assert.Equal(t, "no_such_metric", entries[0].ContextMap()["metric"])
	assert.Equal(t, "calls", entries[1].ContextMap()["metric"])
}

func TestDisabledMetricsSkipDeadlineLookup(t *testing.T) {
	root := metrics.New()
	g := newGraph(root.Scope(), zap.NewNop(), nil, nil, []string{"ttl_ms", "timeout_ttl_ms"})
	req := &transport.Request{Caller: "caller", Service: "service", Encoding: "raw", Procedure: "p"}

	// A context that fails the test if its deadline is consulted.
	ctx := deadlineSpy{Context: context.Background(), t: t}
	c := g.begin(ctx, transport.Unary, _directionInbound, req)
	c.endStats(time.Millisecond, callResult{err: yarpcerrors.DeadlineExceededErrorf("slow")})
}

type deadlineSpy struct {
	context.Context
	t *testing.T
}

func (d deadlineSpy) Deadline() (time.Time, bool) {
	d.t.Error("the deadline must not be read when both ttl histograms are disabled")
	return time.Time{}, false
}

func mustKey(req *transport.Request, direction string, rpcType transport.Type) []byte {
	key, free := getKey(req, direction, rpcType)
	defer free()
	return append([]byte(nil), key...)
}
