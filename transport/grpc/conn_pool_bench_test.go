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

package grpc

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Helpers
// ============================================================================

// benchPeer builds a grpcPeer with n active connections pre-populated.
// Uses makeConn so no real network is required.
func benchPeer(b *testing.B, n int, streamsPerConn int32) *grpcPeer {
	b.Helper()
	p := peerForPool(b)
	conns := make([]*grpcClientConnWrapper, n)
	for i := range conns {
		conns[i] = makeConn(connStateActive, streamsPerConn)
	}
	p.wmu.Lock()
	p.storeConns(conns)
	p.wmu.Unlock()
	return p
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
	w := makeConn(connStateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.incStreamCount()
	}
}

// BenchmarkDecStreamCount measures the cost of atomically decrementing the
// stream counter — runs on every RPC completion (typically via defer).
func BenchmarkDecStreamCount(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateActive, int32(b.N))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.decStreamCount()
	}
}

// BenchmarkGetStreamCount measures atomic load of the stream counter.
func BenchmarkGetStreamCount(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateActive, 42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.getStreamCount()
	}
}

// BenchmarkIncDecStreamCount measures the paired inc+dec that wraps every RPC.
func BenchmarkIncDecStreamCount(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.incStreamCount()
		w.decStreamCount()
	}
}

// BenchmarkIncDecStreamCountParallel measures concurrent inc/dec — models
// many goroutines driving the same connection simultaneously.
func BenchmarkIncDecStreamCountParallel(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateActive, 0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w.incStreamCount()
			w.decStreamCount()
		}
	})
}

// ============================================================================
// Connection state transitions
// ============================================================================

// BenchmarkGetState measures the atomic load of a connection's state.
func BenchmarkGetState(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.getState()
	}
}

// BenchmarkSetState measures the atomic store of a connection's state.
func BenchmarkSetState(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.setState(connStateDraining)
		w.setState(connStateActive)
	}
}

// BenchmarkIsActive measures the isActive convenience check.
func BenchmarkIsActive(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateActive, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.isActive()
	}
}

// ============================================================================
// pickConn — hot path, called on every outbound RPC
// ============================================================================

// BenchmarkPickConn measures connection selection across varying pool sizes.
// The selected connection has the fewest streams; this exercises the full
// linear scan under a read lock.
func BenchmarkPickConn(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 10)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.pickConn()
			}
		})
	}
}

// BenchmarkPickConnUniformLoad benchmarks pickConn when all connections carry
// the same stream count (no clear winner — scans full list every time).
func BenchmarkPickConnUniformLoad(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 50)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.pickConn()
			}
		})
	}
}

// BenchmarkPickConnSkewedLoad benchmarks pickConn when load is heavily skewed:
// the least-loaded conn is always the last one, forcing the full scan to run.
func BenchmarkPickConnSkewedLoad(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := peerForPool(b)
			conns := make([]*grpcClientConnWrapper, n)
			for i := range conns {
				streams := int32(100 - i) // first conn is busiest, last is least
				conns[i] = makeConn(connStateActive, streams)
			}
			p.wmu.Lock()
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.pickConn()
			}
		})
	}
}

// BenchmarkPickConnParallel measures pickConn throughput under high concurrency.
// This is the most important benchmark: it models many goroutines competing for
// the read lock to select a connection.
func BenchmarkPickConnParallel(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 10)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = p.pickConn()
				}
			})
		})
	}
}

// BenchmarkPickConnMixedStates benchmarks pickConn when the pool contains
// active, draining, and idle connections — models real steady-state.
func BenchmarkPickConnMixedStates(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("total_conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := peerForPool(b)
			conns := make([]*grpcClientConnWrapper, n)
			for i := range conns {
				switch i % 3 {
				case 0:
					conns[i] = makeConn(connStateActive, int32(i*5))
				case 1:
					conns[i] = makeConn(connStateDraining, int32(i*2))
				case 2:
					conns[i] = makeConn(connStateIdle, 0)
				}
			}
			p.wmu.Lock()
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.pickConn()
			}
		})
	}
}

// ============================================================================
// tryScaleUp — hot path, called on every outbound RPC after pickConn
// ============================================================================

// BenchmarkTryScaleUpDisabled measures the early-return cost when dynamic
// scaling is off. This is the baseline: every service that doesn't enable
// dynamic scaling pays this overhead per RPC.
func BenchmarkTryScaleUpDisabled(b *testing.B) {
	b.ReportAllocs()
	p := peerForPool(b)
	p.poolCfg.dynamicScalingEnabled = false
	conn := makeConn(connStateActive, 90) // above threshold
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.tryScaleUp(conn)
	}
}

// BenchmarkTryScaleUpBelowThreshold measures tryScaleUp when the stream count
// is below the scale-up threshold. This is the common case: no scale-up fires,
// just a threshold comparison and return.
func BenchmarkTryScaleUpBelowThreshold(b *testing.B) {
	b.ReportAllocs()
	p := peerForPool(b)
	// threshold = 100 * 0.8 = 80; use 50 to be well below
	conn := makeConn(connStateActive, 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.tryScaleUp(conn)
	}
}

// BenchmarkTryScaleUpCASContention measures tryScaleUp when the CAS fails
// because another goroutine is already scaling. This path exits immediately
// after a single atomic CAS — models the contention case at very high RPS.
func BenchmarkTryScaleUpCASContention(b *testing.B) {
	b.ReportAllocs()
	p := peerForPool(b)
	atomic.StoreInt32(&p.isScaling, 1) // pretend a scale-up goroutine is running
	conn := makeConn(connStateActive, 90)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.tryScaleUp(conn)
	}
}

// BenchmarkTryScaleUpBelowThresholdParallel measures the below-threshold fast
// path under concurrent callers — models thousands of goroutines each calling
// tryScaleUp per RPC when pool is not yet saturated.
func BenchmarkTryScaleUpBelowThresholdParallel(b *testing.B) {
	b.ReportAllocs()
	p := peerForPool(b)
	conn := makeConn(connStateActive, 50) // below threshold
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p.tryScaleUp(conn)
		}
	})
}

// ============================================================================
// Full request-path simulation
// ============================================================================

// BenchmarkFullRequestPath simulates the complete per-request hot path:
// pickConn → incStreamCount → tryScaleUp → decStreamCount.
// This is the end-to-end cost added by the connection pool per RPC.
func BenchmarkFullRequestPath(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 10)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				conn := p.pickConn()
				if conn == nil {
					continue
				}
				conn.incStreamCount()
				p.tryScaleUp(conn)
				conn.decStreamCount()
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
			p := benchPeer(b, n, 10)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					conn := p.pickConn()
					if conn == nil {
						continue
					}
					conn.incStreamCount()
					p.tryScaleUp(conn)
					conn.decStreamCount()
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
			p := benchPeer(b, n, 10)
			p.poolCfg.dynamicScalingEnabled = false
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				conn := p.pickConn()
				if conn == nil {
					continue
				}
				conn.incStreamCount()
				p.tryScaleUp(conn)
				conn.decStreamCount()
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
			p := benchPeer(b, n, 10)
			p.poolCfg.dynamicScalingEnabled = false
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					conn := p.pickConn()
					if conn == nil {
						continue
					}
					conn.incStreamCount()
					p.tryScaleUp(conn)
					conn.decStreamCount()
				}
			})
		})
	}
}

// ============================================================================
// maybeScaleDown — runs every scalingMonitorInterval
// ============================================================================

// BenchmarkMaybeScaleDownAtMinConnections measures the early-return path when
// the pool is at minConnections — the common steady-state for low-traffic services.
func BenchmarkMaybeScaleDownAtMinConnections(b *testing.B) {
	b.ReportAllocs()
	p := peerForPool(b)
	p.poolCfg.minConnections = 1
	p.wmu.Lock()
	p.storeConns([]*grpcClientConnWrapper{makeConn(connStateActive, 10)})
	p.wmu.Unlock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.maybeScaleDown()
	}
}

// BenchmarkMaybeScaleDownHighLoad measures the path where scale-down is
// considered but rejected because streams are too high — common during load.
func BenchmarkMaybeScaleDownHighLoad(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := peerForPool(b)
			// threshold=80, each conn has 90 streams → totalStreams always > capacityAfterDrain
			p.wmu.Lock()
			conns := make([]*grpcClientConnWrapper, n)
			for i := range conns {
				conns[i] = makeConn(connStateActive, 90)
			}
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.maybeScaleDown()
			}
		})
	}
}

// BenchmarkMaybeScaleDownTriggered measures the path where scale-down actually
// marks a connection for draining. Re-arms connections each iteration using
// cheap atomic stores (no allocs) so the timer-stopped section is negligible.
func BenchmarkMaybeScaleDownTriggered(b *testing.B) {
	for _, n := range poolSizes {
		if n < 2 {
			continue
		}
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := peerForPool(b)
			// Build connections once; reset state cheaply each iteration.
			conns := make([]*grpcClientConnWrapper, n)
			for j := range conns {
				conns[j] = makeConn(connStateActive, 1)
			}
			p.wmu.Lock()
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Reset state with cheap atomic stores (no alloc).
				for _, c := range conns {
					c.setState(connStateActive)
				}
				p.maybeScaleDown()
			}
		})
	}
}

// ============================================================================
// cleanupIdleConns — runs every scalingMonitorInterval
// ============================================================================

// BenchmarkCleanupIdleConnsEmpty measures the empty-pool fast path.
func BenchmarkCleanupIdleConnsEmpty(b *testing.B) {
	b.ReportAllocs()
	p := peerForPool(b)
	p.wmu.Lock()
	p.storeConns(nil)
	p.wmu.Unlock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.cleanupIdleConns()
	}
}

// BenchmarkCleanupIdleConnsAllActive measures the case where all connections
// are active — no transitions needed, just a scan and early return.
func BenchmarkCleanupIdleConnsAllActive(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 10)
			p.poolCfg.idleTimeout = time.Hour
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.cleanupIdleConns()
			}
		})
	}
}

// BenchmarkCleanupIdleConnsDraining measures transition of draining→idle
// connections with zero streams. Conns are built once; state is reset per
// iteration with a cheap atomic store to avoid alloc in the hot path.
func BenchmarkCleanupIdleConnsDraining(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := peerForPool(b)
			p.poolCfg.idleTimeout = time.Hour
			conns := make([]*grpcClientConnWrapper, n)
			for j := range conns {
				conns[j] = makeConn(connStateDraining, 0)
			}
			p.wmu.Lock()
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, c := range conns {
					c.setState(connStateDraining)
				}
				p.cleanupIdleConns()
			}
		})
	}
}

// BenchmarkCleanupIdleConnsMixed benchmarks a realistic mixed pool:
// some active, some draining with streams, some draining without streams,
// some idle within timeout, some idle past timeout.
// Conns are built once; draining states are reset per iteration. Past-time idle
// conns are cancelled on every iteration (cancel is idempotent) so the toClose
// path is exercised on each pass without any allocs in the loop.
func BenchmarkCleanupIdleConnsMixed(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("total=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := peerForPool(b)
			p.poolCfg.idleTimeout = 5 * time.Minute
			pastTime := time.Now().Add(-10 * time.Minute)
			conns := make([]*grpcClientConnWrapper, n)
			for j := range conns {
				switch j % 4 {
				case 0:
					conns[j] = makeConn(connStateActive, 10)
				case 1:
					conns[j] = makeConn(connStateDraining, 5)
				case 2:
					conns[j] = makeConn(connStateDraining, 0)
				case 3:
					w, _ := makeConnWithCancel(connStateIdle, 0, pastTime)
					conns[j] = w
				}
			}
			p.wmu.Lock()
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Reset draining conns back to their original state.
				for j, c := range conns {
					switch j % 4 {
					case 1:
						c.setState(connStateDraining)
					case 2:
						c.setState(connStateDraining)
					}
				}
				p.cleanupIdleConns()
			}
		})
	}
}

// ============================================================================
// reactivateIdleConn
// ============================================================================

// BenchmarkReactivateIdleConnNoIdle measures the scan-and-miss case where
// there are no idle connections — returns false after scanning all conns.
func BenchmarkReactivateIdleConnNoIdle(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 10) // all active
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.reactivateIdleConn()
			}
		})
	}
}

// BenchmarkReactivateIdleConnSuccess measures the case where the first conn
// is idle and gets reactivated — best case, exits after first element.
// The idle conn's state is reset atomically each iteration; no allocs in loop.
func BenchmarkReactivateIdleConnSuccess(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := peerForPool(b)
			conns := make([]*grpcClientConnWrapper, n)
			// idle conn at position 0 with a background context (never cancelled).
			idleConn := &grpcClientConnWrapper{ctx: context.Background()}
			idleConn.setState(connStateIdle)
			conns[0] = idleConn
			for j := 1; j < n; j++ {
				conns[j] = makeConn(connStateActive, 10)
			}
			p.wmu.Lock()
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idleConn.setState(connStateIdle)
				p.reactivateIdleConn()
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
			p := peerForPool(b)
			conns := make([]*grpcClientConnWrapper, n)
			for j := 0; j < n-1; j++ {
				conns[j] = makeConn(connStateActive, 10)
			}
			idleConn := &grpcClientConnWrapper{ctx: context.Background()}
			idleConn.setState(connStateIdle)
			conns[n-1] = idleConn
			p.wmu.Lock()
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idleConn.setState(connStateIdle)
				p.reactivateIdleConn()
			}
		})
	}
}

// ============================================================================
// evaluateScaling — full monitor tick
// ============================================================================

// BenchmarkEvaluateScaling measures the full per-tick cost of the scaling
// monitor: cleanupIdleConns + maybeScaleDown + pool state log.
func BenchmarkEvaluateScaling(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 10)
			p.poolCfg.idleTimeout = time.Hour // prevent actual close side-effects
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.evaluateScaling()
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
			p := peerForPool(b)
			p.poolCfg.idleTimeout = time.Hour
			p.poolCfg.minConnections = 1
			p.poolCfg.maxConcurrentStreams = 100
			p.poolCfg.scaleUpThreshold = 0.8

			conns := make([]*grpcClientConnWrapper, n)
			for j := range conns {
				if j%2 == 0 {
					conns[j] = makeConn(connStateActive, 5)
				} else {
					conns[j] = makeConn(connStateDraining, 0) // will transition to idle
				}
			}
			p.wmu.Lock()
			p.storeConns(conns)
			p.wmu.Unlock()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// reset draining conns
				for j := range conns {
					if j%2 != 0 {
						conns[j].setState(connStateDraining)
						atomic.StoreInt64(&conns[j].lastIdleAtNano, 0)
					}
				}
				b.StartTimer()
				p.evaluateScaling()
			}
		})
	}
}

// ============================================================================
// refreshPoolMetrics
// ============================================================================

// BenchmarkRefreshPoolMetrics measures the cost of the gauge update that runs
// after every pool state transition (scale-up, scale-down, idle transition).
func BenchmarkRefreshPoolMetrics(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 0)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.refreshPoolMetrics()
			}
		})
	}
}

// ============================================================================
// Read/write lock contention
// ============================================================================

// BenchmarkPickConnWriteContention models the case where a writer (scale-up
// or state transition) competes with concurrent readers (pickConn).
// Readers use RLock; the writer uses full Lock.
func BenchmarkPickConnWriteContention(b *testing.B) {
	for _, n := range poolSizes {
		n := n
		b.Run(fmt.Sprintf("conns=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			p := benchPeer(b, n, 10)

			// Background writer: simulates a scale-up goroutine changing conn state.
			stop := make(chan struct{})
			go func() {
				for {
					select {
					case <-stop:
						return
					default:
						p.wmu.Lock()
						if len(p.loadConns()) > 0 {
							p.loadConns()[0].setState(connStateActive)
						}
						p.wmu.Unlock()
					}
				}
			}()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = p.pickConn()
				}
			})
			b.StopTimer()
			close(stop)
		})
	}
}

// ============================================================================
// connPoolConfig construction
// ============================================================================

// BenchmarkConnPoolConfigConstruction measures the cost of building a
// connPoolConfig from transportOptions fields — happens once per peer creation.
func BenchmarkConnPoolConfigConstruction(b *testing.B) {
	b.ReportAllocs()
	opts := newTransportOptions(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = connPoolConfig{
			dynamicScalingEnabled: opts.clientConnPoolDynamicScalingEnabled,
			maxConcurrentStreams:   opts.clientConnPoolMaxConcurrentStreams,
			scaleUpThreshold:      opts.clientConnPoolScaleUpThreshold,
			minConnections:        opts.clientConnPoolMinConnections,
			maxConnections:        opts.clientConnPoolMaxConnections,
			idleTimeout:           opts.clientConnPoolIdleTimeout,
		}
	}
}

// ============================================================================
// Idle timeout tracking
// ============================================================================

// BenchmarkSetIdleNow measures the atomic write of the idle timestamp.
func BenchmarkSetIdleNow(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateIdle, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.setIdleNow()
	}
}

// BenchmarkIdleSince measures the atomic read + time reconstruction.
func BenchmarkIdleSince(b *testing.B) {
	b.ReportAllocs()
	w := makeConn(connStateIdle, 0)
	w.setIdleNow()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.idleSince()
	}
}

// ============================================================================
// ScalingMonitorInterval — configurable ticker
// ============================================================================

// BenchmarkRunScalingMonitorOverhead measures the goroutine overhead of the
// scaling monitor: how quickly it exits after context cancellation.
func BenchmarkRunScalingMonitorOverhead(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		p := newTestPeer(ctx, cancel)

		done := make(chan struct{})
		go func() {
			p.runScalingMonitor()
			close(done)
		}()
		cancel()
		<-done
	}
}

// ============================================================================
// Scale-up threshold comparison (inline)
// ============================================================================

// BenchmarkThresholdComputation measures the threshold calculation that runs
// on every tryScaleUp call — int32 cast + float multiply.
func BenchmarkThresholdComputation(b *testing.B) {
	b.ReportAllocs()
	cfg := connPoolConfig{
		maxConcurrentStreams: 250,
		scaleUpThreshold:    0.8,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = int32(float64(cfg.maxConcurrentStreams) * cfg.scaleUpThreshold)
	}
}
