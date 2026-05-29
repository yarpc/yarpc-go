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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// newTestPeer returns a minimal grpcPeer sufficient for exercising the scaling
// monitor goroutine lifecycle. Methods that reach p.t will panic, so only use
// this for tests that exercise early-return paths.
func newTestPeer(ctx context.Context, cancel context.CancelFunc) *grpcPeer {
	return &grpcPeer{
		ctx:    ctx,
		cancel: cancel,
	}
}

// makeConn creates a grpcClientConnWrapper with the given state and stream
// count. Intended for use in unit tests only.
func makeConn(state connState, streams int32) *grpcClientConnWrapper {
	return &grpcClientConnWrapper{
		state:       state,
		streamCount: streams,
	}
}

// peerForScaleDown builds a grpcPeer suitable for maybeScaleDown tests. It
// wires up a real Transport (for its nop logger) and a cancellable context.
func peerForScaleDown(t *testing.T, conns []*grpcClientConnWrapper, cfg connPoolConfig) *grpcPeer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	transport := NewTransport()
	p := &grpcPeer{
		Peer:    abstractpeer.NewPeer(abstractpeer.PeerIdentifier("10.0.0.1:9000"), transport),
		t:       transport,
		ctx:     ctx,
		cancel:  cancel,
		poolCfg: cfg,
	}
	p.storeConns(conns)
	return p
}

// defaultCfg is a pool config used across maybeScaleDown tests.
// threshold = int32(100 * 0.8) = 80.
var defaultScaleDownCfg = connPoolConfig{
	minConnections:       1,
	maxConcurrentStreams: 100,
	scaleUpThreshold:     0.8,
}

// makeConnWithCancel creates a grpcClientConnWrapper with a real context so
// that c.cancel() can be observed in tests.  Returns the wrapper and the
// context so callers can assert it was cancelled.
func makeConnWithCancel(state connState, streams int32, idleAt time.Time) (*grpcClientConnWrapper, context.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &grpcClientConnWrapper{
		state:  state,
		cancel: cancel,
	}
	if streams > 0 {
		w.streamCount = streams
	}
	if !idleAt.IsZero() {
		w.lastIdleAtNano = idleAt.UnixNano()
	}
	return w, ctx
}

// TestCleanupIdleConns covers every line/branch of cleanupIdleConns.
func TestCleanupIdleConns(t *testing.T) {
	t.Parallel()

	const shortTimeout = 100 * time.Millisecond

	tests := []struct {
		desc string
		// build returns the conns slice and a slice of contexts corresponding
		// to wrappers whose cancellation should be verified.
		build       func() ([]*grpcClientConnWrapper, []context.Context)
		idleTimeout time.Duration
		// wantStates is the expected state of each conn after the call (index-aligned).
		wantStates []connState
		// wantCancelled is the index set of contexts that must be cancelled after the call.
		wantCancelled []int
	}{
		{
			// Empty pool: lock/unlock + empty loop, nothing happens.
			desc: "empty pool - no-op",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				return nil, nil
			},
			idleTimeout: time.Minute,
			wantStates:  nil,
		},
		{
			// Active connection: neither if-branch fires, state unchanged.
			desc: "active connection - no state change",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				w, ctx := makeConnWithCancel(connStateActive, 5, time.Time{})
				return []*grpcClientConnWrapper{w}, []context.Context{ctx}
			},
			idleTimeout:   time.Minute,
			wantStates:    []connState{connStateActive},
			wantCancelled: nil,
		},
		{
			// Draining with streams > 0: first if is false (streams != 0),
			// second if is false (state != idle) → stays draining.
			desc: "draining with active streams - stays draining",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				w, ctx := makeConnWithCancel(connStateDraining, 3, time.Time{})
				return []*grpcClientConnWrapper{w}, []context.Context{ctx}
			},
			idleTimeout:   time.Minute,
			wantStates:    []connState{connStateDraining},
			wantCancelled: nil,
		},
		{
			// Draining with zero streams: first if fires → setState(idle) + setIdleNow().
			// Second if then checks: state is idle, but idleSince was just set so
			// duration < idleTimeout → not collected for closing.
			desc: "draining with zero streams - advances to idle, not yet timed out",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				w, ctx := makeConnWithCancel(connStateDraining, 0, time.Time{})
				return []*grpcClientConnWrapper{w}, []context.Context{ctx}
			},
			idleTimeout:   time.Hour,
			wantStates:    []connState{connStateIdle},
			wantCancelled: nil,
		},
		{
			// Already idle but lastIdleAtNano == 0 (defensive guard):
			// second if short-circuits on !c.idleSince().IsZero() → not collected.
			desc: "idle with zero idleSince - not collected",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				w, ctx := makeConnWithCancel(connStateIdle, 0, time.Time{}) // lastIdleAtNano stays 0
				return []*grpcClientConnWrapper{w}, []context.Context{ctx}
			},
			idleTimeout:   shortTimeout,
			wantStates:    []connState{connStateIdle},
			wantCancelled: nil,
		},
		{
			// Idle within timeout: all conditions true except duration < timeout → not collected.
			desc: "idle within timeout - not collected",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				w, ctx := makeConnWithCancel(connStateIdle, 0, time.Now()) // idleSince = now
				return []*grpcClientConnWrapper{w}, []context.Context{ctx}
			},
			idleTimeout:   time.Hour,
			wantStates:    []connState{connStateIdle},
			wantCancelled: nil,
		},
		{
			// Idle past timeout: all conditions met → CAS idle→closing succeeds,
			// logger.Debug called, c.cancel() called.
			desc: "idle past timeout - cancel called",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				pastTime := time.Now().Add(-10 * time.Minute)
				w, ctx := makeConnWithCancel(connStateIdle, 0, pastTime)
				return []*grpcClientConnWrapper{w}, []context.Context{ctx}
			},
			idleTimeout:   shortTimeout,
			wantStates:    []connState{connStateClosing},
			wantCancelled: []int{0},
		},
		{
			// Mixed pool: one draining→idle transition, one past-timeout cancel,
			// one active unchanged. Exercises all branches in a single call.
			desc: "mixed pool - correct per-conn behavior",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				w0, ctx0 := makeConnWithCancel(connStateActive, 10, time.Time{})
				w1, ctx1 := makeConnWithCancel(connStateDraining, 0, time.Time{}) // → idle
				pastTime := time.Now().Add(-5 * time.Minute)
				w2, ctx2 := makeConnWithCancel(connStateIdle, 0, pastTime) // → closing + cancel
				return []*grpcClientConnWrapper{w0, w1, w2},
					[]context.Context{ctx0, ctx1, ctx2}
			},
			idleTimeout:   shortTimeout,
			wantStates:    []connState{connStateActive, connStateIdle, connStateClosing},
			wantCancelled: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			conns, ctxs := tt.build()
			cfg := connPoolConfig{idleTimeout: tt.idleTimeout}
			p := peerForScaleDown(t, conns, cfg)

			p.cleanupIdleConns()

			require.Len(t, p.loadConns(), len(tt.wantStates))
			for i, want := range tt.wantStates {
				assert.Equal(t, want, p.loadConns()[i].getState(), "conn[%d] state", i)
			}

			cancelledSet := make(map[int]bool)
			for _, i := range tt.wantCancelled {
				cancelledSet[i] = true
			}
			for i, ctx := range ctxs {
				if cancelledSet[i] {
					assert.Error(t, ctx.Err(), "conn[%d] context should be cancelled", i)
				} else {
					assert.NoError(t, ctx.Err(), "conn[%d] context should not be cancelled", i)
				}
			}
		})
	}
}

func TestMaybeScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		// conns is the initial pool state.
		conns []*grpcClientConnWrapper
		cfg   connPoolConfig
		// afterCall maps each conn (by index) to its expected state after the call.
		wantStates []connState
	}{
		{
			// Empty pool: active slice is empty → len(active)=0 <= minConnections=1 → return.
			desc:       "empty pool - returns at minConnections guard",
			conns:      nil,
			cfg:        defaultScaleDownCfg,
			wantStates: nil,
		},
		{
			// One active conn, minConnections=1: len(active)=1 <= 1 → return, no drain.
			desc: "active equals minConnections - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateActive},
		},
		{
			// One active conn, minConnections=2: len(active)=1 <= 2 → return, no drain.
			desc: "active below minConnections - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
			},
			cfg: connPoolConfig{
				minConnections:       2,
				maxConcurrentStreams: 100,
				scaleUpThreshold:     0.8,
			},
			wantStates: []connState{connStateActive},
		},
		{
			// Only draining conns: active slice is empty → returns at minConnections guard.
			// The existing draining connections are not modified.
			desc: "only draining conns - active=0, no change",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateDraining, 5),
				makeConn(connStateDraining, 5),
			},
			cfg:        defaultScaleDownCfg, // minConnections=1, active=0
			wantStates: []connState{connStateDraining, connStateDraining},
		},
		{
			// Mix of active and draining. Only active conns count toward the
			// pool size, but the pool is at minConnections after filtering.
			desc: "mixed pool at minConnections - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
				makeConn(connStateDraining, 20),
			},
			cfg:        defaultScaleDownCfg, // minConnections=1, active=1
			wantStates: []connState{connStateActive, connStateDraining},
		},
		{
			// 3 active conns, load too high: threshold=80, capacityAfterDrain=80*2=160,
			// totalStreams=180 >= 160 → return without draining.
			desc: "total streams exceed capacity after drain - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 60),
				makeConn(connStateActive, 60),
				makeConn(connStateActive, 60),
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateActive, connStateActive, connStateActive},
		},
		{
			// 3 active conns, load exactly equal to capacityAfterDrain:
			// totalStreams=160 >= 160 → return (boundary check).
			desc: "total streams exactly equal to capacity after drain - no drain",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 54),
				makeConn(connStateActive, 53),
				makeConn(connStateActive, 53),
			},
			cfg:        defaultScaleDownCfg, // threshold=80, capacity=160, total=160
			wantStates: []connState{connStateActive, connStateActive, connStateActive},
		},
		{
			// 3 active conns, low load: threshold=80, capacityAfterDrain=160,
			// totalStreams=60 < 160 → drain the most-loaded (last conn, 30 streams)
			// to maximise residual capacity in the surviving connections.
			desc: "low load - drains most-loaded connection",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
				makeConn(connStateActive, 20),
				makeConn(connStateActive, 30), // most loaded → drained
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateActive, connStateActive, connStateDraining},
		},
		{
			// Most-loaded is the first conn. Verifies the comparison loop
			// picks the globally largest stream count, not just the last.
			desc: "most-loaded is first - correct conn drained",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 30), // most loaded → drained
				makeConn(connStateActive, 5),
				makeConn(connStateActive, 25),
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateDraining, connStateActive, connStateActive},
		},
		{
			// All active conns have equal stream counts. The first one is selected
			// (mostLoaded == nil on first iteration, then no subsequent conn wins
			// because equal counts don't satisfy >).
			desc: "equal stream counts - first active conn drained",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10), // selected (first, tied)
				makeConn(connStateActive, 10),
				makeConn(connStateActive, 10),
			},
			cfg:        defaultScaleDownCfg,
			wantStates: []connState{connStateDraining, connStateActive, connStateActive},
		},
		{
			// Draining conn interspersed: draining conn is excluded from active
			// list and from scale-down candidate selection.
			desc: "draining conn excluded from candidate selection",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateDraining, 1), // excluded from active
				makeConn(connStateActive, 5),
				makeConn(connStateActive, 40), // most loaded active → drained
				makeConn(connStateActive, 40),
			},
			cfg: connPoolConfig{
				minConnections:       1,
				maxConcurrentStreams: 100,
				scaleUpThreshold:     0.8, // threshold=80, capacity=80*2=160, total=85 < 160
			},
			wantStates: []connState{connStateDraining, connStateActive, connStateDraining, connStateActive},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			p := peerForScaleDown(t, tt.conns, tt.cfg)
			p.maybeScaleDown()

			require.Len(t, p.loadConns(), len(tt.wantStates))
			for i, want := range tt.wantStates {
				assert.Equal(t, want, p.loadConns()[i].getState(),
					"conn[%d] state mismatch", i)
			}
		})
	}
}

// --- scaling monitor lifecycle tests ---

// TestRunScalingMonitorExitsOnContextCancel verifies that runScalingMonitor
// returns promptly when the peer's context is cancelled.
func TestRunScalingMonitorExitsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	p := newTestPeer(ctx, cancel)

	done := make(chan struct{})
	go func() {
		p.runScalingMonitor()
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// expected – monitor exited after context cancellation
	case <-time.After(2 * time.Second):
		t.Fatal("runScalingMonitor did not exit after context cancellation")
	}
}

// TestRunScalingMonitorTicksEvaluateScaling verifies that the monitor exits
// cleanly when its context is cancelled.
func TestRunScalingMonitorTicksEvaluateScaling(t *testing.T) {
	t.Parallel()

	// Cancel after a short fixed window rather than a multiple of
	// _scalingMonitorInterval, which is now 30s and would make the test too slow.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	p := newTestPeer(ctx, cancel)

	// runScalingMonitor blocks until the context expires; run it inline so the
	// test waits for it naturally.
	p.runScalingMonitor()

	// If we reach here the monitor exited cleanly on context cancellation – success.
}

// TestEvaluateScalingDoesNotPanic ensures that evaluateScaling (and its
// constituent helpers) can be called on a peer without panicking.
func TestEvaluateScalingDoesNotPanic(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := newTestPeer(ctx, cancel)

	assert.NotPanics(t, p.evaluateScaling)
}

// TestScalingHelperMethodsDoNotPanic verifies individual scaling helpers
// independently to make future implementation easier to validate.
func TestScalingHelperMethodsDoNotPanic(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := newTestPeer(ctx, cancel)

	assert.NotPanics(t, p.cleanupIdleConns)
	assert.NotPanics(t, p.maybeScaleDown)
}

// TestTryScaleUp covers all branches of tryScaleUp.
func TestTryScaleUp(t *testing.T) {
	t.Parallel()

	// threshold = int32(100 * 0.8) = 80
	overBudget := makeConn(connStateActive, 85)
	underBudget := makeConn(connStateActive, 50)

	t.Run("flag disabled - no scale-up", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)
		p.poolCfg.dynamicScalingEnabled = false
		p.tryScaleUp(overBudget)
		assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling))
	})

	t.Run("under threshold - no scale-up", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)
		p.tryScaleUp(underBudget)
		assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling))
	})

	t.Run("at max connections - no scale-up", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)
		conns := make([]*grpcClientConnWrapper, p.poolCfg.maxConnections)
		for i := range conns {
			conns[i] = makeConn(connStateActive, 0)
		}
		p.storeConns(conns)
		p.tryScaleUp(overBudget)

		// atMax check now runs inside the goroutine; wait for it to finish.
		assert.Eventually(t, func() bool {
			return atomic.LoadInt32(&p.isScaling) == 0
		}, 2*time.Second, 10*time.Millisecond)

		n := len(p.loadConns())
		assert.Equal(t, p.poolCfg.maxConnections, n, "pool should not grow beyond max")
	})

	t.Run("already scaling - no second goroutine launched", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)
		atomic.StoreInt32(&p.isScaling, 1)
		p.tryScaleUp(overBudget)
		assert.Equal(t, int32(1), atomic.LoadInt32(&p.isScaling))
	})

	t.Run("triggers goroutine and adds connection", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)
		p.tryScaleUp(overBudget)

		// isScaling resets to 0 once the goroutine completes.
		assert.Eventually(t, func() bool {
			return atomic.LoadInt32(&p.isScaling) == 0
		}, 2*time.Second, 10*time.Millisecond, "isScaling should reset to 0 after goroutine completes")

		// One connection was added to the pool.
		n := len(p.loadConns())
		assert.Equal(t, 1, n, "one connection should be added to the pool")
	})

	t.Run("reactivates idle connection instead of dialing", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)

		// Seed an idle connection with a live context.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		idleConn := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		idleConn.setState(connStateIdle)
		idleConn.setIdleNow()
		p.storeConns(append(p.loadConns(), idleConn))

		p.tryScaleUp(overBudget)

		assert.Eventually(t, func() bool {
			return atomic.LoadInt32(&p.isScaling) == 0
		}, 2*time.Second, 10*time.Millisecond)

		// Pool size unchanged — reactivation, not a new dial.
		n := len(p.loadConns())
		assert.Equal(t, 1, n, "pool size should not grow on reactivation")
		assert.Equal(t, connStateActive, idleConn.getState(), "idle conn should be active after reactivation")
		assert.True(t, idleConn.idleSince().IsZero(), "idle timestamp should be cleared")
	})
}

// TestReactivateIdleConn covers all branches of reactivateIdleConn.
func TestReactivateIdleConn(t *testing.T) {
	t.Parallel()

	t.Run("empty pool returns false", func(t *testing.T) {
		t.Parallel()
		p := peerForScaleDown(t, nil, connPoolConfig{})
		assert.False(t, p.reactivateIdleConn())
	})

	t.Run("no idle connections returns false", func(t *testing.T) {
		t.Parallel()
		conns := []*grpcClientConnWrapper{
			makeConn(connStateActive, 10),
			makeConn(connStateDraining, 5),
		}
		p := peerForScaleDown(t, conns, connPoolConfig{})
		assert.False(t, p.reactivateIdleConn())
	})

	t.Run("idle with cancelled context is skipped", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already cancelled
		w := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		w.setState(connStateIdle)
		p := peerForScaleDown(t, []*grpcClientConnWrapper{w}, connPoolConfig{})
		assert.False(t, p.reactivateIdleConn())
		assert.Equal(t, connStateIdle, w.getState(), "cancelled idle conn must not be reactivated")
	})

	t.Run("reactivates idle connection with live context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		w := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		w.setState(connStateIdle)
		w.setIdleNow()
		p := peerForScaleDown(t, []*grpcClientConnWrapper{w}, connPoolConfig{})

		assert.True(t, p.reactivateIdleConn())
		assert.Equal(t, connStateActive, w.getState())
		assert.True(t, w.idleSince().IsZero(), "idle timestamp should be cleared after reactivation")
	})

	t.Run("skips cancelled idle picks first live idle", func(t *testing.T) {
		t.Parallel()
		cancelledCtx, cancelledCancel := context.WithCancel(context.Background())
		cancelledCancel()
		cancelled := &grpcClientConnWrapper{ctx: cancelledCtx, cancel: cancelledCancel}
		cancelled.setState(connStateIdle)

		liveCtx, liveCancel := context.WithCancel(context.Background())
		defer liveCancel()
		live := &grpcClientConnWrapper{ctx: liveCtx, cancel: liveCancel}
		live.setState(connStateIdle)

		p := peerForScaleDown(t, []*grpcClientConnWrapper{cancelled, live}, connPoolConfig{})

		assert.True(t, p.reactivateIdleConn())
		assert.Equal(t, connStateIdle, cancelled.getState(), "cancelled conn should remain idle")
		assert.Equal(t, connStateActive, live.getState(), "live idle conn should be reactivated")
	})
}

// TestCleanupIdleConnsSkippedWhileScaling verifies that cleanupIdleConns
// returns early without closing any connections when a scale-up is in progress,
// so that idle connections remain available for reactivation by tryScaleUp.
func TestCleanupIdleConnsSkippedWhileScaling(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build an idle connection past its timeout that would normally be closed.
	w := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
	w.setState(connStateIdle)
	atomic.StoreInt64(&w.lastIdleAtNano, time.Now().Add(-10*time.Minute).UnixNano())

	cfg := connPoolConfig{idleTimeout: time.Second}
	p := peerForScaleDown(t, []*grpcClientConnWrapper{w}, cfg)

	// Simulate a scale-up goroutine running.
	atomic.StoreInt32(&p.isScaling, 1)

	p.cleanupIdleConns()

	// Connection must not have been cancelled — isScaling caused early return.
	assert.NoError(t, ctx.Err(), "idle connection must not be cancelled while scaling")
	assert.Equal(t, connStateIdle, w.getState(), "connection state must be unchanged")
}

// --- metrics wiring ---

// peerWithMetrics creates a peerForPool with a real metrics scope attached.
func peerWithMetrics(t *testing.T) (*grpcPeer, *metrics.Root) {
	t.Helper()
	root := metrics.New()
	p := peerForPool(t)
	p.metrics = newConnPoolMetrics(connPoolMetricsParams{
		Meter:       root.Scope(),
		Logger:      zap.NewNop(),
		ServiceName: "test-svc",
		Peer:        p.Peer.Identifier(),
	})
	return p, root
}

func gaugesFromSnapshot(snap *metrics.RootSnapshot) map[string]int64 {
	m := make(map[string]int64, len(snap.Gauges))
	for _, g := range snap.Gauges {
		m[g.Name] = g.Value
	}
	return m
}

func countersFromSnapshot(snap *metrics.RootSnapshot) map[string]int64 {
	m := make(map[string]int64, len(snap.Counters))
	for _, c := range snap.Counters {
		m[c.Name] = c.Value
	}
	return m
}

// TestRefreshPoolMetrics verifies that refreshPoolMetrics correctly derives
// active/draining/idle counts from the pool and publishes them as gauges.
func TestRefreshPoolMetrics(t *testing.T) {
	t.Parallel()

	t.Run("empty pool sets all gauges to zero", func(t *testing.T) {
		t.Parallel()
		p, root := peerWithMetrics(t)
		p.refreshPoolMetrics()
		g := gaugesFromSnapshot(root.Snapshot())
		assert.Equal(t, int64(0), g["conn_pool_active_connections"])
		assert.Equal(t, int64(0), g["conn_pool_draining_connections"])
		assert.Equal(t, int64(0), g["conn_pool_idle_connections"])
	})

	t.Run("mixed pool reports correct counts", func(t *testing.T) {
		t.Parallel()
		p, root := peerWithMetrics(t)
		p.storeConns([]*grpcClientConnWrapper{
			makeConn(connStateActive, 0),
			makeConn(connStateActive, 0),
			makeConn(connStateDraining, 0),
			makeConn(connStateIdle, 0),
		})

		p.refreshPoolMetrics()

		g := gaugesFromSnapshot(root.Snapshot())
		assert.Equal(t, int64(2), g["conn_pool_active_connections"])
		assert.Equal(t, int64(1), g["conn_pool_draining_connections"])
		assert.Equal(t, int64(1), g["conn_pool_idle_connections"])
	})
}

// TestMaybeScaleDownMetrics verifies that maybeScaleDown increments the
// scale-down counter and refreshes the pool gauges when draining a connection.
func TestMaybeScaleDownMetrics(t *testing.T) {
	t.Parallel()

	p, root := peerWithMetrics(t)
	p.poolCfg = connPoolConfig{
		minConnections:       1,
		maxConcurrentStreams: 100,
		scaleUpThreshold:     0.8, // threshold = 80
	}
	p.storeConns([]*grpcClientConnWrapper{
		makeConn(connStateActive, 10),
		makeConn(connStateActive, 10),
		makeConn(connStateActive, 10), // total 30 < capacityAfterDrain=160 → drain
	})

	p.maybeScaleDown()

	c := countersFromSnapshot(root.Snapshot())
	assert.Equal(t, int64(1), c["conn_pool_scale_down_total"], "scale-down counter should increment")

	g := gaugesFromSnapshot(root.Snapshot())
	assert.Equal(t, int64(2), g["conn_pool_active_connections"])
	assert.Equal(t, int64(1), g["conn_pool_draining_connections"])
}

// TestTryScaleUpDialMetrics verifies that tryScaleUp increments the scale-up
// counter when it opens a new connection.
func TestTryScaleUpDialMetrics(t *testing.T) {
	t.Parallel()

	p, root := peerWithMetrics(t)
	overBudget := makeConn(connStateActive, 85) // threshold=80

	p.tryScaleUp(overBudget)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&p.isScaling) == 0
	}, 2*time.Second, 10*time.Millisecond)

	c := countersFromSnapshot(root.Snapshot())
	assert.Equal(t, int64(1), c["conn_pool_scale_up_total"])

	// Gauge must reflect the newly added connection.
	g := gaugesFromSnapshot(root.Snapshot())
	assert.Equal(t, int64(1), g["conn_pool_active_connections"])
}

// TestTryScaleUpReactivationMetrics verifies that tryScaleUp increments the
// idle-reactivation counter and updates gauges when reactivating an idle conn.
func TestTryScaleUpReactivationMetrics(t *testing.T) {
	t.Parallel()

	p, root := peerWithMetrics(t)

	// Seed an idle connection with a live context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	idleConn := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
	idleConn.setState(connStateIdle)
	p.storeConns(append(p.loadConns(), idleConn))

	overBudget := makeConn(connStateActive, 85)
	p.tryScaleUp(overBudget)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&p.isScaling) == 0
	}, 2*time.Second, 10*time.Millisecond)

	c := countersFromSnapshot(root.Snapshot())
	assert.Equal(t, int64(1), c["conn_pool_idle_reactivation_total"])
	assert.Equal(t, int64(0), c["conn_pool_scale_up_total"], "no new dial should happen")

	g := gaugesFromSnapshot(root.Snapshot())
	assert.Equal(t, int64(1), g["conn_pool_active_connections"])
	assert.Equal(t, int64(0), g["conn_pool_idle_connections"])
}

// TestCleanupIdleConnsMetrics verifies that cleanupIdleConns updates gauges
// after advancing draining connections to idle.
func TestCleanupIdleConnsMetrics(t *testing.T) {
	t.Parallel()

	p, root := peerWithMetrics(t)
	p.poolCfg = connPoolConfig{idleTimeout: time.Hour}
	p.storeConns([]*grpcClientConnWrapper{
		makeConn(connStateActive, 5),
		makeConn(connStateDraining, 0), // 0 streams → advances to idle
	})

	p.cleanupIdleConns()

	g := gaugesFromSnapshot(root.Snapshot())
	assert.Equal(t, int64(1), g["conn_pool_active_connections"])
	assert.Equal(t, int64(0), g["conn_pool_draining_connections"])
	assert.Equal(t, int64(1), g["conn_pool_idle_connections"])
}

// TestMaybeScaleDownWithExistingDrainingConn verifies that when some
// connections are already draining, maybeScaleDown only considers active
// connections for the pool size check and candidate selection.  The
// already-draining connection is left untouched; one of the active ones is
// drained if load is low enough.
func TestMaybeScaleDownWithExistingDrainingConn(t *testing.T) {
	t.Parallel()
	// 2 active + 1 already-draining. active=2 > minConnections=1, low load → drain one active.
	conns := []*grpcClientConnWrapper{
		makeConn(connStateActive, 5),
		makeConn(connStateActive, 5),
		makeConn(connStateDraining, 30), // already draining — excluded from active pool
	}
	p := peerForScaleDown(t, conns, defaultScaleDownCfg)
	p.maybeScaleDown()

	// conns[2] must remain draining — it was never in the active selection pool.
	assert.Equal(t, connStateDraining, conns[2].getState(), "pre-existing draining conn must not change state")

	// Exactly one of the active conns must now be draining.
	drained := 0
	for _, c := range conns[:2] {
		if c.getState() == connStateDraining {
			drained++
		}
	}
	assert.Equal(t, 1, drained, "exactly one active conn should be drained")
}

// TestCleanupIdleConnsDrainingCASFailure verifies that cleanupIdleConns skips
// the draining→idle transition when another goroutine has already changed the
// connection state away from draining (CAS failure path).
func TestCleanupIdleConnsDrainingCASFailure(t *testing.T) {
	t.Parallel()

	// Connection starts draining with zero streams — would normally advance to idle.
	w, ctx := makeConnWithCancel(connStateDraining, 0, time.Time{})

	// Simulate a concurrent reactivation: state is changed to active before
	// cleanupIdleConns gets to the CAS.
	w.setState(connStateActive)

	cfg := connPoolConfig{idleTimeout: time.Hour}
	p := peerForScaleDown(t, []*grpcClientConnWrapper{w}, cfg)
	p.cleanupIdleConns()

	// CAS draining→idle failed (state was active) — connection stays active.
	assert.Equal(t, connStateActive, w.getState())
	assert.NoError(t, ctx.Err(), "active connection must not be cancelled")
}

// TestRunScalingMonitorUsesConfiguredInterval verifies that a non-zero
// scalingMonitorInterval from poolCfg is used instead of the default,
// and that a value below the 30s minimum is clamped (with a warning) rather
// than causing a panic or hard error.
func TestRunScalingMonitorUsesConfiguredInterval(t *testing.T) {
	t.Parallel()
	// peerForPool provides a transport with a nop logger so the clamping warning
	// in runScalingMonitor does not panic.
	p := peerForPool(t)
	// 5s is below the 30s minimum → will be clamped; the monitor still exits
	// promptly when the context is cancelled.
	p.poolCfg.scalingMonitorInterval = 5 * time.Second

	done := make(chan struct{})
	go func() {
		p.runScalingMonitor()
		close(done)
	}()

	p.cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runScalingMonitor did not exit after context cancellation with clamped interval")
	}
}

// TestRunScalingMonitorValidCustomInterval verifies that an interval at or
// above the 30s minimum is used as-is without clamping.
func TestRunScalingMonitorValidCustomInterval(t *testing.T) {
	t.Parallel()
	p := peerForPool(t)
	p.poolCfg.scalingMonitorInterval = 60 * time.Second // valid, above minimum

	done := make(chan struct{})
	go func() {
		p.runScalingMonitor()
		close(done)
	}()

	p.cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runScalingMonitor did not exit after context cancellation with 60s interval")
	}
}

// TestRunScalingMonitorClampsAndWarns verifies that a below-minimum interval
// emits a Warn log and the monitor still exits cleanly on context cancellation.
func TestRunScalingMonitorClampsAndWarns(t *testing.T) {
	t.Parallel()
	core, logs := observer.New(zap.WarnLevel)
	observedLogger := zap.New(core)

	tr := NewTransport(Logger(observedLogger))
	ctx, cancel := context.WithCancel(context.Background())
	p := &grpcPeer{
		Peer:    abstractpeer.NewPeer(abstractpeer.PeerIdentifier("10.0.0.1:9000"), tr),
		t:       tr,
		ctx:     ctx,
		cancel:  cancel,
		poolCfg: connPoolConfig{scalingMonitorInterval: 5 * time.Second},
	}
	t.Cleanup(cancel)

	done := make(chan struct{})
	go func() {
		p.runScalingMonitor()
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runScalingMonitor did not exit")
	}

	require.Equal(t, 1, logs.Len(), "expected exactly one warning log")
	assert.Contains(t, logs.All()[0].Message, "scalingMonitorInterval")
	assert.Equal(t, zap.WarnLevel, logs.All()[0].Level)
}

// TestTransportOptionDefaults verifies that newTransportOptions applies the
// correct default values for the two new pool config fields.
func TestTransportOptionDefaults(t *testing.T) {
	t.Parallel()
	opts := newTransportOptions(nil)
	assert.Equal(t, defaultClientConnPoolScaleDownGap, opts.clientConnPoolScaleDownGap,
		"scaleDownGap default should be %.2f", defaultClientConnPoolScaleDownGap)
	assert.Equal(t, defaultClientConnPoolScalingMonitorInterval, opts.clientConnPoolScalingMonitorInterval,
		"scalingMonitorInterval default should be %v", defaultClientConnPoolScalingMonitorInterval)
}

// between cleanupIdleConns (which cancels idle connections) and
// reactivateIdleConn (which transitions idle connections back to active).
// Run with -race to catch the race described in the review:
//
//	Time 1 cleanupIdleConns: reads isScaling==0, adds c to toClose
//	Time 2 reactivateIdleConn: CAS idle→active
//	Time 3 cleanupIdleConns: would cancel an active connection without CAS guard
//
// With CAS, only one of the two wins the idle→closing or idle→active transition.
func TestConcurrentCleanupAndReactivationRace(t *testing.T) {
	t.Parallel()

	const iterations = 500
	for i := 0; i < iterations; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		w := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		w.setState(connStateIdle)
		atomic.StoreInt64(&w.lastIdleAtNano, time.Now().Add(-10*time.Minute).UnixNano())

		cfg := connPoolConfig{idleTimeout: time.Second}
		p := peerForScaleDown(t, []*grpcClientConnWrapper{w}, cfg)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.cleanupIdleConns()
		}()
		go func() {
			defer wg.Done()
			p.reactivateIdleConn()
		}()
		wg.Wait()

		// Invariant: if the connection is active, its context must not be cancelled.
		// This would fire if cleanupIdleConns cancelled a connection that
		// reactivateIdleConn had already transitioned to active.
		if w.getState() == connStateActive {
			assert.NoError(t, ctx.Err(),
				"active connection must not have a cancelled context (iteration %d)", i)
		}
	}
}

// TestConcurrentScaleDownAndScaleUpRace verifies that concurrent maybeScaleDown
// and tryScaleUp calls on the same pool do not corrupt connection state.
// The race being guarded: both functions read the pool snapshot and evaluate the
// same connection; the CAS in each (active→draining for scaleDown, draining→active
// or dial for scaleUp) ensures only one winner per transition.
//
// Invariant: after each iteration no connection is simultaneously draining and
// receiving new streams (stream count on a draining conn must not increase after
// the CAS succeeds, because pickConn skips non-active connections).
// Run with -race to catch any unsynchronised reads/writes.
func TestConcurrentScaleDownAndScaleUpRace(t *testing.T) {
	t.Parallel()

	const iterations = 500
	for i := 0; i < iterations; i++ {
		ctx, cancel := context.WithCancel(context.Background())

		// Two active connections each at 80% load (scaleUpThreshold=0.8, streams=80/100).
		// maybeScaleDown: totalStreams=160, capacityAfterDrain=80*1=80 → 160>=80 → no drain.
		// Keep load high so tryScaleUp also fires (least-loaded is at threshold).
		cfg := connPoolConfig{
			minConnections:       1,
			maxConnections:       5,
			maxConcurrentStreams: 100,
			scaleUpThreshold:     0.8,
			scaleDownGap:         0.1,
		}
		c1 := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		c1.setState(connStateActive)
		atomic.StoreInt32(&c1.streamCount, 80)
		c2 := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		c2.setState(connStateActive)
		atomic.StoreInt32(&c2.streamCount, 80)

		p := peerForScaleDown(t, []*grpcClientConnWrapper{c1, c2}, cfg)
		t.Cleanup(cancel)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.maybeScaleDown()
		}()
		go func() {
			defer wg.Done()
			// tryScaleUp picks the least-loaded conn; both are at threshold.
			p.tryScaleUp(c1)
		}()
		wg.Wait()
		cancel() // stop any scale-up dial goroutine

		// Invariant: draining connections must not be treated as active by pickConn.
		// If a connection is draining its state must be consistent — no connection
		// should be both draining AND still selected as active by pickConn.
		picked := p.pickConn()
		for _, c := range p.loadConns() {
			if c.getState() == connStateDraining {
				assert.NotEqual(t, c, picked,
					"iteration %d: draining connection must not be picked as active", i)
			}
		}
	}
}

// TestMaybeScaleDownHysteresis verifies that the scaleDownGap prevents draining
// when stream count is between the scale-down and scale-up thresholds.
func TestMaybeScaleDownHysteresis(t *testing.T) {
	t.Parallel()

	// scaleUpThreshold=0.8, scaleDownGap=0.1 → scaleDownThreshold=0.7
	// maxConcurrentStreams=100 → scaleDownThreshold=70
	// With 3 active conns: capacityAfterDrain = 70 * 2 = 140
	cfg := connPoolConfig{
		minConnections:       1,
		maxConcurrentStreams: 100,
		scaleUpThreshold:     0.8,
		scaleDownGap:         0.1,
	}

	t.Run("load between scale-down and scale-up thresholds - no drain", func(t *testing.T) {
		t.Parallel()
		// totalStreams=150: above scaleDownThreshold capacity (140) but below scaleUpThreshold capacity (160)
		// Without hysteresis this would drain; with gap it should not.
		conns := []*grpcClientConnWrapper{
			makeConn(connStateActive, 50),
			makeConn(connStateActive, 50),
			makeConn(connStateActive, 50), // total=150, capacityAfterDrain=140, 150>=140 → no drain
		}
		p := peerForScaleDown(t, conns, cfg)
		p.maybeScaleDown()
		for i, c := range p.loadConns() {
			assert.Equal(t, connStateActive, c.getState(), "conn[%d] should stay active", i)
		}
	})

	t.Run("load below scale-down threshold - drains", func(t *testing.T) {
		t.Parallel()
		// totalStreams=60: below capacityAfterDrain=140 → should drain
		conns := []*grpcClientConnWrapper{
			makeConn(connStateActive, 20),
			makeConn(connStateActive, 20),
			makeConn(connStateActive, 20), // total=60 < 140 → drain
		}
		p := peerForScaleDown(t, conns, cfg)
		p.maybeScaleDown()
		draining := 0
		for _, c := range p.loadConns() {
			if c.getState() == connStateDraining {
				draining++
			}
		}
		assert.Equal(t, 1, draining, "exactly one connection should be draining")
	})
}

// TestReactivateDrainingConn verifies that reactivateIdleConn falls back to
// reactivating a draining connection when no idle connection is available,
// preventing accumulation of stuck draining connections under sustained load.
func TestReactivateDrainingConn(t *testing.T) {
	t.Parallel()

	t.Run("reactivates draining connection when no idle available", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		draining := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		draining.setState(connStateDraining)
		draining.streamCount = 5 // has in-flight streams

		p := peerForScaleDown(t, []*grpcClientConnWrapper{draining}, connPoolConfig{})
		assert.True(t, p.reactivateIdleConn(), "should reactivate draining conn")
		assert.Equal(t, connStateActive, draining.getState())
	})

	t.Run("prefers idle over draining for reactivation", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		idle := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		idle.setState(connStateIdle)

		ctx2, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		draining := &grpcClientConnWrapper{ctx: ctx2, cancel: cancel2}
		draining.setState(connStateDraining)

		p := peerForScaleDown(t, []*grpcClientConnWrapper{draining, idle}, connPoolConfig{})
		assert.True(t, p.reactivateIdleConn())
		assert.Equal(t, connStateActive, idle.getState(), "idle conn should be reactivated first")
		assert.Equal(t, connStateDraining, draining.getState(), "draining conn should be untouched")
	})

	t.Run("skips draining conn with cancelled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already cancelled

		draining := &grpcClientConnWrapper{ctx: ctx, cancel: cancel}
		draining.setState(connStateDraining)

		p := peerForScaleDown(t, []*grpcClientConnWrapper{draining}, connPoolConfig{})
		assert.False(t, p.reactivateIdleConn(), "cancelled draining conn must not be reactivated")
		assert.Equal(t, connStateDraining, draining.getState())
	})
}
