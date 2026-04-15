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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/peer/abstractpeer"
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
	return &grpcPeer{
		Peer:    abstractpeer.NewPeer(abstractpeer.PeerIdentifier("10.0.0.1:9000"), transport),
		t:       transport,
		ctx:     ctx,
		cancel:  cancel,
		conns:   conns,
		poolCfg: cfg,
	}
}

// defaultCfg is a pool config used across maybeScaleDown tests.
// threshold = int32(100 * 0.8) = 80.
var defaultScaleDownCfg = connPoolConfig{
	minConnections:      1,
	maxConcurrentStreams: 100,
	scaleUpThreshold:    0.8,
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
		build      func() ([]*grpcClientConnWrapper, []context.Context)
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
			// Idle past timeout: all conditions met → appended to toClose,
			// logger.Debug called, c.cancel() called.
			desc: "idle past timeout - cancel called",
			build: func() ([]*grpcClientConnWrapper, []context.Context) {
				pastTime := time.Now().Add(-10 * time.Minute)
				w, ctx := makeConnWithCancel(connStateIdle, 0, pastTime)
				return []*grpcClientConnWrapper{w}, []context.Context{ctx}
			},
			idleTimeout:   shortTimeout,
			wantStates:    []connState{connStateIdle},
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
				w2, ctx2 := makeConnWithCancel(connStateIdle, 0, pastTime) // → cancel
				return []*grpcClientConnWrapper{w0, w1, w2},
					[]context.Context{ctx0, ctx1, ctx2}
			},
			idleTimeout:   shortTimeout,
			wantStates:    []connState{connStateActive, connStateIdle, connStateIdle},
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

			require.Len(t, p.conns, len(tt.wantStates))
			for i, want := range tt.wantStates {
				assert.Equal(t, want, p.conns[i].getState(), "conn[%d] state", i)
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
				minConnections:      2,
				maxConcurrentStreams: 100,
				scaleUpThreshold:    0.8,
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
				minConnections:      1,
				maxConcurrentStreams: 100,
				scaleUpThreshold:    0.8, // threshold=80, capacity=80*2=160, total=85 < 160
			},
			wantStates: []connState{connStateDraining, connStateActive, connStateDraining, connStateActive},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			p := peerForScaleDown(t, tt.conns, tt.cfg)
			p.maybeScaleDown()

			require.Len(t, p.conns, len(tt.wantStates))
			for i, want := range tt.wantStates {
				assert.Equal(t, want, p.conns[i].getState(),
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
		p.mu.Lock()
		p.conns = conns
		p.mu.Unlock()
		p.tryScaleUp(overBudget)

		// atMax check now runs inside the goroutine; wait for it to finish.
		assert.Eventually(t, func() bool {
			return atomic.LoadInt32(&p.isScaling) == 0
		}, 2*time.Second, 10*time.Millisecond)

		p.mu.RLock()
		n := len(p.conns)
		p.mu.RUnlock()
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
		p.mu.RLock()
		n := len(p.conns)
		p.mu.RUnlock()
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
		p.mu.Lock()
		p.conns = append(p.conns, idleConn)
		p.mu.Unlock()

		p.tryScaleUp(overBudget)

		assert.Eventually(t, func() bool {
			return atomic.LoadInt32(&p.isScaling) == 0
		}, 2*time.Second, 10*time.Millisecond)

		// Pool size unchanged — reactivation, not a new dial.
		p.mu.RLock()
		n := len(p.conns)
		p.mu.RUnlock()
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
