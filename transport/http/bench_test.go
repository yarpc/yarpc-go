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

// Benchmark suite for HTTP/2 transport performance exploration.
//
// Run all benchmarks:
//   go test -bench=. -benchmem -benchtime=5s ./transport/http/
//
// Run a specific area:
//   go test -bench=BenchmarkPayload -benchmem -benchtime=5s ./transport/http/
//
// With CPU profiling:
//   go test -bench=BenchmarkHeaderAlloc -benchmem -cpuprofile=cpu.prof ./transport/http/
//   go tool pprof cpu.prof
//
// With memory profiling:
//   go test -bench=BenchmarkHeaderAlloc -benchmem -memprofile=mem.prof ./transport/http/
//   go tool pprof mem.prof

package http

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// ── Payload sizes used across benchmarks ──────────────────────────────────────

var (
	payload1KB  = bytes.Repeat([]byte("x"), 1<<10)
	payload64KB = bytes.Repeat([]byte("x"), 64<<10)
	payload1MB  = bytes.Repeat([]byte("x"), 1<<20)
	payload4MB  = bytes.Repeat([]byte("x"), 4<<20)
)

// ── Shared server helpers ─────────────────────────────────────────────────────

// echoHandler reads the request body and writes it back unchanged.
// Used for all round-trip benchmarks.
func echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Rpc-Status", "success")
		_, _ = w.Write(body)
	}
}

// newH2CServer creates an HTTP/2 cleartext server backed by echoHandler.
func newH2CServer(b testing.TB) *httptest.Server {
	b.Helper()
	h2s := &http2.Server{IdleTimeout: 15 * time.Minute}
	srv := httptest.NewServer(h2c.NewHandler(echoHandler(), h2s))
	b.Cleanup(srv.Close)
	return srv
}

// newH1Server creates a plain HTTP/1.1 server backed by echoHandler.
func newH1Server(b testing.TB) *httptest.Server {
	b.Helper()
	srv := httptest.NewServer(echoHandler())
	b.Cleanup(srv.Close)
	return srv
}

// newOutbound creates a started outbound to the given URL.
func newOutbound(b testing.TB, url string, opts ...OutboundOption) *Outbound {
	b.Helper()
	tr := NewTransport()
	out := tr.NewSingleOutbound(url, opts...)
	if err := out.Start(); err != nil {
		b.Fatalf("start outbound: %v", err)
	}
	b.Cleanup(func() {
		_ = out.Stop()
		_ = tr.Stop()
	})
	return out
}

// call issues one unary RPC and discards the response.
// Returns true on success. Non-fatal on error so concurrency benchmarks
// can count failures rather than abort.
func call(b testing.TB, out *Outbound, body []byte) bool {
	b.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := out.Call(ctx, &transport.Request{
		Caller:    "bench-caller",
		Service:   "bench-service",
		Encoding:  raw.Encoding,
		Procedure: "echo",
		Body:      bytes.NewReader(body),
	})
	if err != nil {
		return false
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area 1: HTTP/1.1 vs HTTP/2 — baseline transport comparison
//
// Goal: establish the overhead difference between the two transports.
// Expect HTTP/2 to be slightly higher latency for small payloads (framing
// overhead) but significantly better for large payloads and concurrency.
// ═══════════════════════════════════════════════════════════════════════════════

func BenchmarkBaseline_HTTP1_1KB(b *testing.B) {
	srv := newH1Server(b)
	out := newOutbound(b, srv.URL)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		call(b, out, payload1KB)
	}
}

func BenchmarkBaseline_HTTP2_1KB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		call(b, out, payload1KB)
	}
}

func BenchmarkBaseline_HTTP1_64KB(b *testing.B) {
	srv := newH1Server(b)
	out := newOutbound(b, srv.URL)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		call(b, out, payload64KB)
	}
}

func BenchmarkBaseline_HTTP2_64KB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		call(b, out, payload64KB)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area 2: Flow control window size impact
//
// Goal: show the throughput cliff when payload exceeds the default 65 KB
// HTTP/2 flow control window. These benchmarks expose the need for a larger
// InitialStreamWindowSize or a BDP estimator.
//
// The 1 MB and 4 MB cases will stall waiting for WINDOW_UPDATE frames when
// the default 65 KB window is exhausted.
// ═══════════════════════════════════════════════════════════════════════════════

func BenchmarkWindowSize_Default_1MB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload1MB)))
	for i := 0; i < b.N; i++ {
		call(b, out, payload1MB)
	}
}

func BenchmarkWindowSize_Default_4MB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload4MB)))
	for i := 0; i < b.N; i++ {
		call(b, out, payload4MB)
	}
}

// BenchmarkWindowSize_Large_* — PENDING implementation of two things:
//
//  1. H2Transport(t *http2.Transport) TransportOption  — expose a custom
//     http2.Transport to the yarpc Transport so we can override settings.
//
//  2. golang.org/x/net/http2.Transport does NOT expose InitialStreamWindowSize
//     directly. The only way to bump the client window is to patch the
//     unexported http2.Transport.initialWindowSize field via the ConnPool
//     interface, or to wait for upstream to expose it. This is itself a key
//     finding: x/net/http2 is less configurable than grpc-go's transport.
//
// Until those are implemented, the Default_1MB / Default_4MB benchmarks above
// serve as the baseline showing WHERE the window cliff hurts.
//
// TODO: add BenchmarkWindowSize_Large_1MB and _4MB once H2Transport option
// and client-side window tuning are implemented.

// ═══════════════════════════════════════════════════════════════════════════════
// Area 3: Concurrency — stream multiplexing
//
// Goal: measure how well HTTP/2 multiplexes concurrent streams vs HTTP/1.1
// which opens a new connection per request (or queues behind the 2-conn pool).
//
// HTTP/1.1 will be throttled by MaxIdleConnsPerHost=2 (default).
// HTTP/2 multiplexes all goroutines over one connection.
// ═══════════════════════════════════════════════════════════════════════════════

func benchmarkConcurrent(b *testing.B, out *Outbound, body []byte, concurrency int) {
	b.Helper()
	b.SetParallelism(concurrency)
	b.ResetTimer()
	b.ReportAllocs()
	var errors int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if !call(b, out, body) {
				errors++
			}
		}
	})
	if errors > 0 {
		b.Logf("errors: %d (%.1f%%)", errors, float64(errors)/float64(b.N)*100)
	}
}

func BenchmarkConcurrency_HTTP1_10goroutines_1KB(b *testing.B) {
	srv := newH1Server(b)
	out := newOutbound(b, srv.URL)
	benchmarkConcurrent(b, out, payload1KB, 10)
}

func BenchmarkConcurrency_HTTP2_10goroutines_1KB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	benchmarkConcurrent(b, out, payload1KB, 10)
}

func BenchmarkConcurrency_HTTP1_100goroutines_1KB(b *testing.B) {
	srv := newH1Server(b)
	out := newOutbound(b, srv.URL)
	benchmarkConcurrent(b, out, payload1KB, 100)
}

func BenchmarkConcurrency_HTTP2_100goroutines_1KB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	benchmarkConcurrent(b, out, payload1KB, 100)
}

func BenchmarkConcurrency_HTTP2_100goroutines_64KB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	benchmarkConcurrent(b, out, payload64KB, 100)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area 4: Header allocation hot path
//
// Goal: isolate and measure the per-request allocation cost of header
// processing in headerMapper.ToHTTPHeaders(). This shows how much we'd
// gain from pooling the http.Header map.
//
// Run with -memprofile to see allocation sites.
// ═══════════════════════════════════════════════════════════════════════════════

// BenchmarkHeaderAlloc_CurrentImpl measures the current implementation:
// makes a new http.Header every call.
func BenchmarkHeaderAlloc_CurrentImpl(b *testing.B) {
	appHeaders := transport.NewHeaders().
		With("x-request-id", "abc123").
		With("x-uber-source", "bench").
		With("x-trace-id", "0000ffff")

	m := headerMapper{ApplicationHeaderPrefix}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := make(http.Header, appHeaders.Len())
		m.ToHTTPHeaders(appHeaders, h)
		_ = h
	}
}

// BenchmarkHeaderAlloc_Pooled demonstrates the allocation saving from a
// sync.Pool of pre-allocated http.Header maps.
// This is the target implementation we want to reach.
func BenchmarkHeaderAlloc_Pooled(b *testing.B) {
	appHeaders := transport.NewHeaders().
		With("x-request-id", "abc123").
		With("x-uber-source", "bench").
		With("x-trace-id", "0000ffff")

	pool := sync.Pool{
		New: func() any {
			return make(http.Header, 16) // pre-sized for typical RPC headers
		},
	}

	m := headerMapper{ApplicationHeaderPrefix}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := pool.Get().(http.Header)
		m.ToHTTPHeaders(appHeaders, h)
		_ = h
		// Clear before returning to pool — cheap O(k) where k = num headers.
		for k := range h {
			delete(h, k)
		}
		pool.Put(h)
	}
}

// BenchmarkHeaderAlloc_EndToEnd measures header alloc cost as part of a full
// round-trip so we can see it as a fraction of total request cost.
func BenchmarkHeaderAlloc_EndToEnd_NoAppHeaders(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		call(b, out, payload1KB)
	}
}

func BenchmarkHeaderAlloc_EndToEnd_10AppHeaders(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())

	// Pre-build request with 10 application headers to stress the mapper.
	req := &transport.Request{
		Caller:    "bench-caller",
		Service:   "bench-service",
		Encoding:  raw.Encoding,
		Procedure: "echo",
		Headers: transport.NewHeaders().
			With("x-request-id", "abc123").
			With("x-uber-source", "bench").
			With("x-region", "us-east-1").
			With("x-datacenter", "dc1").
			With("x-env", "production").
			With("x-feature-flag", "enabled").
			With("x-tenant", "uber").
			With("x-client-version", "1.2.3").
			With("x-session-id", "sess-xyz").
			With("x-trace-flags", "01"),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req.Body = bytes.NewReader(payload1KB)
		res, err := out.Call(ctx, req)
		cancel()
		if err != nil {
			b.Fatalf("call: %v", err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area 5: Write coalescing
//
// Goal: measure whether batching small writes before flush reduces syscall
// overhead. We simulate the grpc-go loopyWriter pattern: yield once with
// runtime.Gosched() when buffer < 1 KB before flushing.
//
// This area benchmarks the effect rather than the implementation, by
// measuring throughput under varying burst sizes sent concurrently.
// The burst creates the natural batching opportunity.
// ═══════════════════════════════════════════════════════════════════════════════

// BenchmarkCoalescing_SingleStream: no coalescing opportunity (sequential).
func BenchmarkCoalescing_SingleStream_1KB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		call(b, out, payload1KB)
	}
}

// BenchmarkCoalescing_BurstStreams: concurrent small writes — here grpc-go's
// loopyWriter would coalesce them. yarpc sends each immediately.
// Compare this ns/op against single-stream to see the goroutine scheduling cost.
func BenchmarkCoalescing_BurstStreams_1KB(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())
	const burst = 10

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg sync.WaitGroup
			wg.Add(burst)
			for j := 0; j < burst; j++ {
				go func() {
					defer wg.Done()
					call(b, out, payload1KB)
				}()
			}
			wg.Wait()
		}
	})
}

// ══════════════════════════════════════════════════���════════════════════════════
// Area 6: Health check — TCP dial vs PING
//
// Goal: quantify the cost of yarpc's current health-check approach (full TCP
// dial + immediate close) vs a single HTTP/2 PING on an existing connection.
//
// We benchmark two approaches:
//   a) Current: net.Dial → check → close   (what peer.isAvailable does today)
//   b) Proposed: send HTTP/2 PING frame on live conn
//
// We use the round-trip latency of an empty-body request as a proxy for PING
// RTT since net/http2 doesn't expose raw PING directly.
// ═══════════════════════════════════════════════════════════════════════════════

// BenchmarkHealthCheck_TCPDial measures the cost of opening a new TCP
// connection per health probe (current behaviour).
func BenchmarkHealthCheck_TCPDial(b *testing.B) {
	srv := newH2CServer(b)
	addr := strings.TrimPrefix(srv.URL, "http://")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Replicate what peer.isAvailable() does today.
		probeConn(addr, 500*time.Millisecond)
	}
}

// BenchmarkHealthCheck_H2Probe measures the optimized health-check path:
// H2Probing() TransportOption routes probes through the HTTP/2 connection pool
// instead of dialing a new TCP socket on every check.
// Compare directly against BenchmarkHealthCheck_TCPDial.
func BenchmarkHealthCheck_H2Probe(b *testing.B) {
	srv := newH2CServer(b)
	// Enable H2Probing so isAvailable() uses probeH2() instead of probeTCP().
	tr := NewTransport(H2Probing())
	out := tr.NewSingleOutbound(srv.URL, UseHTTP2())
	if err := out.Start(); err != nil {
		b.Fatalf("start: %v", err)
	}
	b.Cleanup(func() { _ = out.Stop(); _ = tr.Stop() })

	// Warm up: ensure connection is pooled before timing probes.
	call(b, out, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call(b, out, nil)
	}
}

// BenchmarkHealthCheck_PingViaEmptyRequest measures health-check latency
// when reusing an existing HTTP/2 connection (zero-body round-trip).
// This approximates the cost of a PING frame on a live connection.
func BenchmarkHealthCheck_PingViaEmptyRequest(b *testing.B) {
	srv := newH2CServer(b)
	out := newOutbound(b, srv.URL, UseHTTP2())

	// Warm up the connection.
	call(b, out, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call(b, out, nil) // empty body — minimal framing cost
	}
}

// probeConn simulates what peer.go MaintainConn does: dial + close.
// This is the baseline for Area 6 health-check benchmarks.
func probeConn(addr string, timeout time.Duration) {
	dialer := &net.Dialer{Timeout: timeout}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return
	}
	_ = conn.Close()
}
