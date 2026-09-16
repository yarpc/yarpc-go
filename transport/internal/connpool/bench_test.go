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

package connpool

// This file ports the benchmark suite that used to live in
// transport/grpc/conn_pool_bench_test.go (branch
// feature-grpc-connection-pooling-benchmarking-atomic, commit 959644d3)
// forward onto the transport-agnostic Pool[T]/Wrapper[T] types this package
// extracted the connection-pooling logic into (see conn.go/pool.go/
// scaler.go doc comments). The old benchmarks targeted transport/grpc's
// private grpcClientConnWrapper/grpcPeer types, which no longer exist after
// the extraction; the scenarios themselves (stream-count atomics, pickConn
// scan cost, scale-up/down, cleanup/reactivation, background monitor) are
// unchanged and are reproduced here against Pool[*fakeConn].
//
// BenchmarkConnPoolConfigConstruction from the old file is intentionally not
// ported: it benchmarked transport/grpc's own option-parsing into
// connPoolConfig, which is unaffected by this extraction and still lives
// (and is still benchmarked/tested) in transport/grpc.

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ============================================================================
// Helpers
// ============================================================================

// newBenchPool builds a Pool[*fakeConn] with a no-op logger and fake watcher,
// without requiring a *testing.T (unlike newTestPool in pool_test.go, which
// this mirrors).
func newBenchPool(cfg Config) *Pool[*fakeConn] {
	var nextID atomic.Int32
	dial := func(_ context.Context) (*fakeConn, error) {
		return &fakeConn{id: int(nextID.Add(1))}, nil
	}
	p := NewPool(context.Background(), cfg, dial, zap.NewNop(), "bench-pool", nil)
	attachFakeWatcher(p)
	return p
}

// benchPool builds a Pool with n active connections, each pre-loaded with
// streamsPerConn in-flight streams. It registers b.Cleanup to stop and wait
// for the pool's background goroutines (fake watchers, scaling monitor) so
// the package's goleak.VerifyTestMain check (leak_test.go) does not flag
// benchmark-only pools left running across the whole `go test -bench` binary.
func benchPool(b *testing.B, n int, streamsPerConn int32) *Pool[*fakeConn] {
	b.Helper()
	cfg := baseConfig()
	cfg.MaxConnections = n + 1
	p := newBenchPool(cfg)
	if err := p.Start(n); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { p.Stop(); p.Wait() })
	for _, c := range p.LoadConns() {
		for i := int32(0); i < streamsPerConn; i++ {
			c.IncStreamCount()
		}
	}
	return p
}

// startBenchPool wraps newBenchPool+Start with the same cleanup registration
// as benchPool, for benchmarks that need to seed connections themselves
// rather than via benchPool's uniform streamsPerConn.
func startBenchPool(b *testing.B, cfg Config, initialConnCount int) *Pool[*fakeConn] {
	b.Helper()
	p := newBenchPool(cfg)
	require1(b, p.Start(initialConnCount))
	b.Cleanup(func() { p.Stop(); p.Wait() })
	return p
}

// benchWrapper builds a standalone Wrapper[*fakeConn], bypassing Pool
// entirely, for benchmarks that only exercise Wrapper-level atomics.
func benchWrapper(state State, streamCount int32) *Wrapper[*fakeConn] {
	w := newWrapper(context.Background(), &fakeConn{})
	w.setState(state)
	for i := int32(0); i < streamCount; i++ {
		w.IncStreamCount()
	}
	return w
}

// poolSizes is the canonical set of pool sizes used across sub-benchmarks.
var poolSizes = []int{1, 2, 4, 8, 16, 32}

// ============================================================================
// Stream count atomics
// ============================================================================

// BenchmarkIncStreamCount measures the cost of atomically incrementing the
// stream counter on a connection wrapper — runs on every inbound RPC.
func BenchmarkIncStreamCount(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.IncStreamCount()
	}
}

// BenchmarkDecStreamCount measures the cost of atomically decrementing the
// stream counter — runs on every RPC completion (typically via defer).
func BenchmarkDecStreamCount(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateActive, int32(b.N))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.DecStreamCount()
	}
}

// BenchmarkGetStreamCount measures atomic load of the stream counter.
func BenchmarkGetStreamCount(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateActive, 42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.StreamCount()
	}
}

// BenchmarkIncDecStreamCount measures the paired inc+dec that wraps every RPC.
func BenchmarkIncDecStreamCount(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.IncStreamCount()
		w.DecStreamCount()
	}
}

// BenchmarkIncDecStreamCountParallel measures concurrent inc/dec — models
// many goroutines driving the same connection simultaneously.
func BenchmarkIncDecStreamCountParallel(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateActive, 0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w.IncStreamCount()
			w.DecStreamCount()
		}
	})
}

// ============================================================================
// Connection state transitions
// ============================================================================

// BenchmarkGetState measures the atomic load of a connection's state.
func BenchmarkGetState(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.GetState()
	}
}

// BenchmarkSetState measures the atomic store of a connection's state.
func BenchmarkSetState(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.setState(StateDraining)
		w.setState(StateActive)
	}
}

// BenchmarkIsActive measures the IsActive convenience check.
func BenchmarkIsActive(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.IsActive()
	}
}

// ============================================================================
// PickConn — hot path, called on every outbound RPC
// ============================================================================

// BenchmarkPickConn measures connection selection across varying pool sizes.
// The selected connection has the fewest streams; this exercises the full
// linear scan.
func BenchmarkPickConn(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.PickConn()
			}
		})
	}
}

// BenchmarkPickConnUniformLoad benchmarks PickConn when all connections carry
// the same stream count (no clear winner — scans full list every time).
func BenchmarkPickConnUniformLoad(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 50)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.PickConn()
			}
		})
	}
}

// BenchmarkPickConnSkewedLoad benchmarks PickConn when load is heavily skewed:
// the least-loaded conn is always the last one, forcing the full scan to run.
func BenchmarkPickConnSkewedLoad(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 0)
			conns := p.LoadConns()
			for i, c := range conns {
				for j := int32(0); j < int32(100-i); j++ { // first conn is busiest, last is least
					c.IncStreamCount()
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.PickConn()
			}
		})
	}
}

// BenchmarkPickConnParallel measures PickConn throughput under high
// concurrency. This is the most important benchmark: it models many
// goroutines competing to select a connection.
func BenchmarkPickConnParallel(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = p.PickConn()
				}
			})
		})
	}
}

// BenchmarkPickConnMixedStates benchmarks PickConn when the pool contains
// active, draining, and idle connections — models real steady-state.
func BenchmarkPickConnMixedStates(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("total_conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 0)
			conns := p.LoadConns()
			for i, c := range conns {
				switch i % 3 {
				case 0:
					for j := 0; j < i*5; j++ {
						c.IncStreamCount()
					}
				case 1:
					c.setState(StateDraining)
					for j := 0; j < i*2; j++ {
						c.IncStreamCount()
					}
				case 2:
					c.setState(StateIdle)
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.PickConn()
			}
		})
	}
}

// BenchmarkPickConnWriteContention models the case where a writer (scale-up
// or state transition) competes with concurrent readers (PickConn). Pool
// membership is a lock-free copy-on-write slice, so the "write" here is a
// plain atomic state store on an existing wrapper.
func BenchmarkPickConnWriteContention(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)

			stop := make(chan struct{})
			go func() {
				for {
					select {
					case <-stop:
						return
					default:
						if conns := p.LoadConns(); len(conns) > 0 {
							conns[0].setState(StateActive)
						}
					}
				}
			}()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = p.PickConn()
				}
			})
			b.StopTimer()
			close(stop)
		})
	}
}

// ============================================================================
// TryScaleUp — hot path, called on every outbound RPC after PickConn
// ============================================================================

// BenchmarkTryScaleUpDisabled measures the early-return cost when dynamic
// scaling is off. This is the baseline: every pool that doesn't enable
// dynamic scaling pays this overhead per RPC.
func BenchmarkTryScaleUpDisabled(b *testing.B) {
	b.ReportAllocs()
	cfg := baseConfig()
	cfg.DynamicScalingEnabled = false
	p := startBenchPool(b, cfg, 1)
	conn := p.LoadConns()[0]
	for i := int32(0); i < 90; i++ { // above threshold
		conn.IncStreamCount()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TryScaleUp(conn)
	}
}

// BenchmarkTryScaleUpBelowThreshold measures TryScaleUp when the stream
// count is below the scale-up threshold (threshold = 100*0.8 = 80; use 50 to
// be well below). This is the common case: no scale-up fires, just a
// threshold comparison and return.
func BenchmarkTryScaleUpBelowThreshold(b *testing.B) {
	b.ReportAllocs()
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	p := startBenchPool(b, cfg, 1)
	conn := p.LoadConns()[0]
	for i := int32(0); i < 50; i++ {
		conn.IncStreamCount()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TryScaleUp(conn)
	}
}

// BenchmarkTryScaleUpCASContention measures TryScaleUp when the CAS fails
// because another goroutine is already scaling. This path exits immediately
// after a single atomic CAS — models the contention case at very high RPS.
func BenchmarkTryScaleUpCASContention(b *testing.B) {
	b.ReportAllocs()
	p := startBenchPool(b, baseConfig(), 1)
	atomic.StoreInt32(&p.isScaling, 1) // pretend a scale-up goroutine is running
	conn := p.LoadConns()[0]
	for i := int32(0); i < 90; i++ {
		conn.IncStreamCount()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TryScaleUp(conn)
	}
}

// BenchmarkTryScaleUpBelowThresholdParallel measures the below-threshold fast
// path under concurrent callers — models thousands of goroutines each calling
// TryScaleUp per RPC when the pool is not yet saturated.
func BenchmarkTryScaleUpBelowThresholdParallel(b *testing.B) {
	b.ReportAllocs()
	p := startBenchPool(b, baseConfig(), 1)
	conn := p.LoadConns()[0]
	for i := int32(0); i < 50; i++ { // below threshold
		conn.IncStreamCount()
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p.TryScaleUp(conn)
		}
	})
}

// ============================================================================
// Full request-path simulation
// ============================================================================

// BenchmarkFullRequestPath simulates the complete per-request hot path:
// PickConn → IncStreamCount → TryScaleUp → DecStreamCount. This is the
// end-to-end cost the connection pool adds per RPC.
func BenchmarkFullRequestPath(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				conn := p.PickConn()
				if conn == nil {
					continue
				}
				conn.IncStreamCount()
				p.TryScaleUp(conn)
				conn.DecStreamCount()
			}
		})
	}
}

// BenchmarkFullRequestPathParallel is the concurrent version of
// BenchmarkFullRequestPath — the most realistic model of production traffic.
func BenchmarkFullRequestPathParallel(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					conn := p.PickConn()
					if conn == nil {
						continue
					}
					conn.IncStreamCount()
					p.TryScaleUp(conn)
					conn.DecStreamCount()
				}
			})
		})
	}
}

// BenchmarkFullRequestPathScalingDisabled is the baseline: same hot path but
// with dynamic scaling turned off. Diff against BenchmarkFullRequestPath to
// isolate the overhead of the scaling feature.
func BenchmarkFullRequestPathScalingDisabled(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			p.Config.DynamicScalingEnabled = false
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				conn := p.PickConn()
				if conn == nil {
					continue
				}
				conn.IncStreamCount()
				p.TryScaleUp(conn)
				conn.DecStreamCount()
			}
		})
	}
}

// BenchmarkFullRequestPathScalingDisabledParallel is the concurrent baseline.
func BenchmarkFullRequestPathScalingDisabledParallel(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			p.Config.DynamicScalingEnabled = false
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					conn := p.PickConn()
					if conn == nil {
						continue
					}
					conn.IncStreamCount()
					p.TryScaleUp(conn)
					conn.DecStreamCount()
				}
			})
		})
	}
}

// ============================================================================
// MaybeScaleDown — runs every ScalingMonitorInterval
// ============================================================================

// BenchmarkMaybeScaleDownAtMinConns measures the early-return path when the
// pool is at MinConnections — the common steady-state for low-traffic peers.
func BenchmarkMaybeScaleDownAtMinConns(b *testing.B) {
	b.ReportAllocs()
	cfg := baseConfig()
	cfg.MinConnections = 1
	p := startBenchPool(b, cfg, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.MaybeScaleDown()
	}
}

// BenchmarkMaybeScaleDownHighLoad measures the path where scale-down is
// considered but rejected because streams are too high — common during load.
func BenchmarkMaybeScaleDownHighLoad(b *testing.B) {
	for _, n := range poolSizes {
		if n < 2 {
			continue
		}
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			// threshold=80, each conn has 90 streams -> totalStreams always > capacityAfterDrain.
			p := benchPool(b, n, 90)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.MaybeScaleDown()
			}
		})
	}
}

// BenchmarkMaybeScaleDownTriggered measures the path where scale-down
// actually marks a connection for draining. Connections are re-armed to
// StateActive each iteration via cheap atomic stores (no allocs).
func BenchmarkMaybeScaleDownTriggered(b *testing.B) {
	for _, n := range poolSizes {
		if n < 2 {
			continue
		}
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 1)
			conns := p.LoadConns()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, c := range conns {
					c.setState(StateActive)
				}
				p.MaybeScaleDown()
			}
		})
	}
}

// ============================================================================
// CleanupIdleConns — runs every ScalingMonitorInterval
// ============================================================================

// BenchmarkCleanupIdleConnsEmpty measures the empty-pool fast path.
func BenchmarkCleanupIdleConnsEmpty(b *testing.B) {
	b.ReportAllocs()
	p := benchPool(b, 0, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.CleanupIdleConns()
	}
}

// BenchmarkCleanupIdleConnsAllActive measures the case where all connections
// are active — no transitions needed, just a scan and early return.
func BenchmarkCleanupIdleConnsAllActive(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			p.Config.IdleTimeout = time.Hour
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.CleanupIdleConns()
			}
		})
	}
}

// BenchmarkCleanupIdleConnsDraining measures the draining->idle transition of
// zero-stream connections. State is reset each iteration with a cheap atomic
// store to avoid allocs in the hot path.
func BenchmarkCleanupIdleConnsDraining(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 0)
			p.Config.IdleTimeout = time.Hour
			conns := p.LoadConns()
			for _, c := range conns {
				c.setState(StateDraining)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, c := range conns {
					c.setState(StateDraining)
				}
				p.CleanupIdleConns()
			}
		})
	}
}

// BenchmarkCleanupIdleConnsMixed benchmarks a realistic mixed pool: some
// active, some draining with streams, some draining without streams, some
// idle past timeout. Draining states are reset per iteration.
func BenchmarkCleanupIdleConnsMixed(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("total=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 0)
			p.Config.IdleTimeout = 5 * time.Minute
			pastTime := time.Now().Add(-10 * time.Minute)
			conns := p.LoadConns()
			for j, c := range conns {
				switch j % 4 {
				case 0:
					for k := 0; k < 10; k++ {
						c.IncStreamCount()
					}
				case 1:
					c.setState(StateDraining)
					for k := 0; k < 5; k++ {
						c.IncStreamCount()
					}
				case 2:
					c.setState(StateDraining)
				case 3:
					c.setState(StateIdle)
					atomic.StoreInt64(&c.lastIdleAtNano, pastTime.UnixNano())
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j, c := range conns {
					if j%4 == 1 || j%4 == 2 {
						c.setState(StateDraining)
					}
				}
				p.CleanupIdleConns()
			}
		})
	}
}

// ============================================================================
// ReactivateIdleConn
// ============================================================================

// BenchmarkReactivateIdleConnNoIdle measures the scan-and-miss case where
// there are no idle connections — returns false after scanning all conns.
func BenchmarkReactivateIdleConnNoIdle(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10) // all active
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.ReactivateIdleConn()
			}
		})
	}
}

// BenchmarkReactivateIdleConnSuccess measures the case where the first conn
// is idle and gets reactivated — best case, exits after the first element.
func BenchmarkReactivateIdleConnSuccess(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			idleConn := p.LoadConns()[0]
			idleConn.setState(StateIdle)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idleConn.setState(StateIdle)
				p.ReactivateIdleConn()
			}
		})
	}
}

// BenchmarkReactivateIdleConnLastPosition measures the worst case: the only
// idle connection is the last element — must scan all active conns first.
func BenchmarkReactivateIdleConnLastPosition(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			conns := p.LoadConns()
			idleConn := conns[n-1]
			idleConn.setState(StateIdle)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idleConn.setState(StateIdle)
				p.ReactivateIdleConn()
			}
		})
	}
}

// ============================================================================
// EvaluateScaling — full monitor tick
// ============================================================================

// BenchmarkEvaluateScaling measures the full per-tick cost of the scaling
// monitor: CleanupIdleConns + MaybeScaleDown.
func BenchmarkEvaluateScaling(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 10)
			p.Config.IdleTimeout = time.Hour // prevent actual close side-effects
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.EvaluateScaling()
			}
		})
	}
}

// BenchmarkEvaluateScalingWithDraining measures a tick that includes
// draining-to-idle transitions — models post-scale-down state.
func BenchmarkEvaluateScalingWithDraining(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 0)
			p.Config.IdleTimeout = time.Hour
			p.Config.MinConnections = 1
			p.Config.MaxConcurrentStreams = 100
			p.Config.ScaleUpThreshold = 0.8

			conns := p.LoadConns()
			for j, c := range conns {
				if j%2 == 0 {
					for k := 0; k < 5; k++ {
						c.IncStreamCount()
					}
				} else {
					c.setState(StateDraining) // will transition to idle
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				for j := range conns {
					if j%2 != 0 {
						conns[j].setState(StateDraining)
						atomic.StoreInt64(&conns[j].lastIdleAtNano, 0)
					}
				}
				b.StartTimer()
				p.EvaluateScaling()
			}
		})
	}
}

// ============================================================================
// RefreshMetrics
// ============================================================================

// BenchmarkRefreshMetrics measures the cost of the gauge update that runs
// after every pool state transition (scale-up, scale-down, idle transition).
func BenchmarkRefreshMetrics(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPool(b, n, 0)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.RefreshMetrics()
			}
		})
	}
}

// ============================================================================
// Idle timeout tracking
// ============================================================================

// BenchmarkSetIdleNow measures the atomic write of the idle timestamp.
func BenchmarkSetIdleNow(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateIdle, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.setIdleNow()
	}
}

// BenchmarkIdleSince measures the atomic read plus time reconstruction.
func BenchmarkIdleSince(b *testing.B) {
	b.ReportAllocs()
	w := benchWrapper(StateIdle, 0)
	w.setIdleNow()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.IdleSince()
	}
}

// ============================================================================
// ScalingMonitorInterval — background monitor overhead
// ============================================================================

// BenchmarkRunScalingMonitorOverhead measures the goroutine overhead of the
// scaling monitor: how quickly it exits after context cancellation.
func BenchmarkRunScalingMonitorOverhead(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := newBenchPool(baseConfig())
		p.connWg.Add(1)
		done := make(chan struct{})
		go func() {
			p.RunScalingMonitor()
			close(done)
		}()
		p.cancel()
		<-done
	}
}

// ============================================================================
// Scale-up threshold comparison (inline)
// ============================================================================

// BenchmarkThresholdComputation measures the threshold calculation performed
// on every TryScaleUp call — int32 cast + float multiply.
func BenchmarkThresholdComputation(b *testing.B) {
	b.ReportAllocs()
	cfg := Config{
		MaxConcurrentStreams: 250,
		ScaleUpThreshold:     0.8,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = int32(float64(cfg.MaxConcurrentStreams) * cfg.ScaleUpThreshold)
	}
}

// require1 fails the benchmark immediately if err is non-nil, mirroring
// require.NoError without pulling testify into a benchmark-only path.
func require1(b *testing.B, err error) {
	b.Helper()
	if err != nil {
		b.Fatal(err)
	}
}
