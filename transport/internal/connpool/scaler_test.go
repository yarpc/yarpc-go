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

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestScaler_TryScaleUpDialsWhenAboveThreshold(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	conns[0].IncStreamCount() // push above 0.8*250 is not enough with 1 stream; force threshold instead
	p.Config.MaxConcurrentStreams = 1
	p.Config.ScaleUpThreshold = 0.5 // threshold = 0 -> immediately eligible

	p.TryScaleUp(conns[0])

	require.Eventually(t, func() bool {
		return len(p.LoadConns()) == 2
	}, time.Second, time.Millisecond)
}

func TestScaler_TryScaleUpNoopBelowThreshold(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	p.TryScaleUp(conns[0]) // zero load, well below 0.8*250 threshold

	time.Sleep(20 * time.Millisecond)
	assert.Len(t, p.LoadConns(), 1)
}

func TestScaler_TryScaleUpNoopWhenDynamicScalingDisabled(t *testing.T) {
	cfg := baseConfig()
	cfg.DynamicScalingEnabled = false
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	for range 300 {
		conns[0].IncStreamCount()
	}
	p.TryScaleUp(conns[0])

	time.Sleep(20 * time.Millisecond)
	assert.Len(t, p.LoadConns(), 1)
}

func TestScaler_TryScaleUpRespectsMaxConnections(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConnections = 2
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	p.TryScaleUp(conns[0])

	time.Sleep(20 * time.Millisecond)
	assert.Len(t, p.LoadConns(), 2, "must not exceed MaxConnections")
}

func TestScaler_TryScaleUpReactivatesIdleBeforeDialing(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	idle := conns[1]
	require.True(t, idle.TransitionState(StateActive, StateDraining))
	require.True(t, idle.TransitionState(StateDraining, StateIdle))
	idle.setIdleNow()

	dialedBefore := nextID.Load()
	p.TryScaleUp(conns[0])

	require.Eventually(t, func() bool {
		return idle.GetState() == StateActive
	}, time.Second, time.Millisecond)
	assert.Len(t, p.LoadConns(), 2, "reactivation should not dial a new connection")
	assert.Equal(t, dialedBefore, nextID.Load())
}

func TestScaler_MaybeScaleDownHysteresis(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	cfg.ScaleUpThreshold = 0.8
	cfg.ScaleDownGap = 0.1
	cfg.MinConnections = 1
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	// scaleDownThreshold = 100*(0.8-0.1) = 70. capacityAfterDrain with 1
	// remaining conn = 70. Put enough load that draining would exceed 70.
	conns[0].IncStreamCount()
	for range 75 {
		conns[1].IncStreamCount()
	}

	p.MaybeScaleDown()
	assert.True(t, conns[0].IsActive())
	assert.True(t, conns[1].IsActive())
	assert.Equal(t, 2, len(p.LoadConns()))
}

func TestScaler_MaybeScaleDownDrainsWhenLoadFits(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	cfg.ScaleUpThreshold = 0.8
	cfg.ScaleDownGap = 0.1
	cfg.MinConnections = 1
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	// Total load well under the 70-stream capacity of a single connection.
	conns[0].IncStreamCount()

	p.MaybeScaleDown()

	drainedCount := 0
	for _, c := range p.LoadConns() {
		if c.GetState() == StateDraining {
			drainedCount++
		}
	}
	assert.Equal(t, 1, drainedCount)
}

// TestScaler_MaybeScaleDownBoundaryEqualCapacityNoDrain ports grpc's boundary
// check: when total load is exactly equal to capacityAfterDrain, the >=
// comparison must NOT drain (only strictly-less load may drain).
func TestScaler_MaybeScaleDownBoundaryEqualCapacityNoDrain(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	cfg.ScaleUpThreshold = 0.8
	cfg.ScaleDownGap = 0
	cfg.MinConnections = 1
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(3))
	defer func() { p.Stop(); p.Wait() }()

	// scaleDownThreshold = 100*0.8 = 80; capacityAfterDrain (2 remaining) = 160.
	conns := p.LoadConns()
	for range 54 {
		conns[0].IncStreamCount()
	}
	for range 53 {
		conns[1].IncStreamCount()
	}
	for range 53 {
		conns[2].IncStreamCount()
	}

	p.MaybeScaleDown()

	for _, c := range conns {
		assert.True(t, c.IsActive(), "load exactly at the capacity boundary must not drain")
	}
}

// TestScaler_MaybeScaleDownDrainsMostLoadedRegardlessOfPosition ports grpc's
// check that the most-loaded selection scans the entire active set rather
// than tracking only the last connection seen.
func TestScaler_MaybeScaleDownDrainsMostLoadedRegardlessOfPosition(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	cfg.ScaleUpThreshold = 0.8
	cfg.MinConnections = 1
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(3))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	for range 30 {
		conns[0].IncStreamCount()
	}
	for range 5 {
		conns[1].IncStreamCount()
	}
	for range 25 {
		conns[2].IncStreamCount()
	}

	p.MaybeScaleDown()

	assert.Equal(t, StateDraining, conns[0].GetState(), "globally most-loaded conn (first in slice) must be drained")
	assert.True(t, conns[1].IsActive())
	assert.True(t, conns[2].IsActive())
}

// TestScaler_MaybeScaleDownTiesPickFirstActiveConn ports grpc's tie-break
// check: when all active conns carry equal load, the first one encountered
// is selected (mostLoaded == nil on the first iteration; no subsequent equal
// conn satisfies the strict > comparison).
func TestScaler_MaybeScaleDownTiesPickFirstActiveConn(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	cfg.ScaleUpThreshold = 0.8
	cfg.MinConnections = 1
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(3))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	for _, c := range conns {
		for range 10 {
			c.IncStreamCount()
		}
	}

	p.MaybeScaleDown()

	assert.Equal(t, StateDraining, conns[0].GetState(), "first active conn wins the tie")
	assert.True(t, conns[1].IsActive())
	assert.True(t, conns[2].IsActive())
}

// TestScaler_MaybeScaleDownExcludesDrainingFromCandidatesAndSizeCheck ports
// grpc's check that a pre-existing Draining connection is excluded both from
// the MinConnections size check and from scale-down candidate selection: it
// must never be touched by MaybeScaleDown itself.
func TestScaler_MaybeScaleDownExcludesDrainingFromCandidatesAndSizeCheck(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	cfg.ScaleUpThreshold = 0.8
	cfg.MinConnections = 1
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(4))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	require.True(t, conns[0].TransitionState(StateActive, StateDraining))
	conns[0].IncStreamCount() // give it load; it must still be ignored

	for range 40 {
		conns[2].IncStreamCount()
	}
	for range 40 {
		conns[3].IncStreamCount()
	}
	// conns[1] stays at zero load.

	p.MaybeScaleDown()

	assert.Equal(t, StateDraining, conns[0].GetState(), "pre-existing draining conn is untouched")
	drained := 0
	for _, c := range conns[1:] {
		if c.GetState() == StateDraining {
			drained++
		}
	}
	assert.Equal(t, 1, drained, "exactly one of the active conns should be drained")
}

func TestScaler_MaybeScaleDownNeverBelowMinConnections(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	p.MaybeScaleDown()

	for _, c := range p.LoadConns() {
		assert.True(t, c.IsActive())
	}
}

func TestScaler_CleanupIdleConnsAdvancesDrainedToIdle(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()

	c := p.LoadConns()[0]
	require.True(t, c.TransitionState(StateActive, StateDraining))

	p.CleanupIdleConns()
	assert.Equal(t, StateIdle, c.GetState())
	assert.False(t, c.IdleSince().IsZero())
}

func TestScaler_CleanupIdleConnsCancelsTimedOutConn(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = 10 * time.Millisecond
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	c := p.LoadConns()[0]
	require.True(t, c.TransitionState(StateActive, StateDraining))
	require.True(t, c.TransitionState(StateDraining, StateIdle))
	c.setIdleNow()

	time.Sleep(20 * time.Millisecond)
	p.CleanupIdleConns()

	// CleanupIdleConns only cancels the wrapper's context; the actual
	// close/remove happens asynchronously in the per-connection watcher once
	// it observes the cancellation.
	assert.Equal(t, StateClosing, c.GetState())
	require.Eventually(t, func() bool {
		return len(p.LoadConns()) == 1
	}, time.Second, time.Millisecond)
	assert.True(t, c.Conn.closed.Load())
}

// TestScaler_CleanupIdleConnsPreservesNonEligibleStates ports grpc's
// exhaustive per-branch coverage of cleanupIdleConns (transport/grpc's
// former conn_pool_scaler_test.go TestCleanupIdleConns), which exercised
// several branches not otherwise covered above: a Draining connection with
// in-flight load must not advance to Idle, and an Idle connection must not
// be collected for closure either while its IdleSince is still zero (the
// defensive uninitialized-timestamp guard) or while it is within
// IdleTimeout.
func TestScaler_CleanupIdleConnsPreservesNonEligibleStates(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = time.Hour
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(4))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()

	// conns[0] stays Active: neither branch of CleanupIdleConns applies.

	// conns[1]: Draining with in-flight load must NOT advance to Idle.
	require.True(t, conns[1].TransitionState(StateActive, StateDraining))
	conns[1].IncStreamCount()

	// conns[2]: Idle but never had setIdleNow called, so IdleSince is zero.
	// The defensive !IdleSince().IsZero() guard must exclude it from closure.
	require.True(t, conns[2].TransitionState(StateActive, StateDraining))
	require.True(t, conns[2].TransitionState(StateDraining, StateIdle))

	// conns[3]: Idle, set just now -- within the (1 hour) IdleTimeout.
	require.True(t, conns[3].TransitionState(StateActive, StateDraining))
	require.True(t, conns[3].TransitionState(StateDraining, StateIdle))
	conns[3].setIdleNow()

	p.CleanupIdleConns()

	assert.Equal(t, StateActive, conns[0].GetState())
	assert.Equal(t, StateDraining, conns[1].GetState(), "draining conn with in-flight load must stay draining")
	assert.Equal(t, StateIdle, conns[2].GetState(), "idle conn with zero IdleSince must not be closed")
	assert.Equal(t, StateIdle, conns[3].GetState(), "idle conn within timeout must not be closed")
	assert.Len(t, p.LoadConns(), 4)
}

func TestScaler_CleanupIdleConnsSkippedWhileScaling(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = 1 * time.Millisecond
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	c := p.LoadConns()[0]
	require.True(t, c.TransitionState(StateActive, StateDraining))
	require.True(t, c.TransitionState(StateDraining, StateIdle))
	c.setIdleNow()
	time.Sleep(5 * time.Millisecond)

	atomic.StoreInt32(&p.isScaling, 1)
	p.CleanupIdleConns()
	atomic.StoreInt32(&p.isScaling, 0)

	assert.Len(t, p.LoadConns(), 2, "cleanup should be skipped while a scale-up is in flight")
}

func TestScaler_ReactivateIdleConnPrefersIdleOverDraining(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(3))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	draining, idle := conns[0], conns[1]
	require.True(t, draining.TransitionState(StateActive, StateDraining))
	require.True(t, idle.TransitionState(StateActive, StateDraining))
	require.True(t, idle.TransitionState(StateDraining, StateIdle))

	ok := p.ReactivateIdleConn()
	require.True(t, ok)
	assert.Equal(t, StateActive, idle.GetState())
	assert.Equal(t, StateDraining, draining.GetState())
}

func TestScaler_ReactivateIdleConnFallsBackToDraining(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	require.True(t, conns[0].TransitionState(StateActive, StateDraining))

	ok := p.ReactivateIdleConn()
	require.True(t, ok)
	assert.Equal(t, StateActive, conns[0].GetState())
}

func TestScaler_ReactivateIdleConnFalseWhenNoneAvailable(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	assert.False(t, p.ReactivateIdleConn())
}

// TestScaler_ReactivateIdleConnSkipsCancelledDraining ports grpc's "skips
// draining conn with cancelled context" case: ReactivateIdleConn's second
// pass (draining fallback) must not reactivate a draining connection whose
// context has already been cancelled (e.g. by a concurrent CleanupIdleConns
// claiming it for closure elsewhere). No watcher is attached, so cancelling
// the connection's context has no side effect other than what this test
// observes directly.
func TestScaler_ReactivateIdleConnSkipsCancelledDraining(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	p.OnConnAdded = nil // no watcher: avoid a race with async removal on Cancel
	require.NoError(t, p.Start(1))
	defer func() {
		p.ConnDone() // balance the connWg.Add(1) a watcher would normally close
		p.Stop()
		p.Wait()
	}()

	c := p.LoadConns()[0]
	require.True(t, c.TransitionState(StateActive, StateDraining))
	c.Cancel()

	assert.False(t, p.ReactivateIdleConn())
	assert.Equal(t, StateDraining, c.GetState())
}

func TestScaler_RunScalingMonitorExitsOnStop(t *testing.T) {
	cfg := baseConfig()
	cfg.ScalingMonitorInterval = time.Hour // never ticks in this test's lifetime
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1))

	done := make(chan struct{})
	go func() {
		p.Wait()
		close(done)
	}()

	p.Stop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scaling monitor did not exit after Stop")
	}
}

func TestScaler_RunScalingMonitorClampsSmallInterval(t *testing.T) {
	cfg := baseConfig()
	cfg.ScalingMonitorInterval = time.Millisecond
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()
	// No assertion beyond "does not panic and does not tick every
	// millisecond"; the clamp is exercised, correctness is that Stop()
	// above still terminates promptly (checked implicitly by not timing
	// out the surrounding test).
}

func TestScaler_RunScalingMonitorZeroIntervalUsesDefault(t *testing.T) {
	cfg := baseConfig()
	cfg.ScalingMonitorInterval = 0
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()
	// No assertion beyond "does not panic"; exercises the interval<=0 branch
	// that falls back to defaultScalingMonitorInterval.
}

func TestScaler_EvaluateScalingRunsCleanupThenScaleDown(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = 10 * time.Millisecond
	cfg.MinConnections = 1
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2))
	defer func() { p.Stop(); p.Wait() }()

	idle := p.LoadConns()[0]
	require.True(t, idle.TransitionState(StateActive, StateDraining))
	require.True(t, idle.TransitionState(StateDraining, StateIdle))
	idle.setIdleNow()
	time.Sleep(20 * time.Millisecond)

	p.EvaluateScaling()

	assert.Equal(t, StateClosing, idle.GetState())
}

func TestScaler_TryScaleUpNoopWhenAlreadyScaling(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()

	atomic.StoreInt32(&p.isScaling, 1)
	defer atomic.StoreInt32(&p.isScaling, 0)

	dialedBefore := nextID.Load()
	p.TryScaleUp(p.LoadConns()[0])

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, dialedBefore, nextID.Load(), "must not dial while another scale-up is in flight")
}

func TestScaler_TryScaleUpNoopAfterShutdown(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1))
	conn := p.LoadConns()[0]
	p.Stop()
	p.Wait()

	p.TryScaleUp(conn)
	assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling))
}

func TestScaler_TryScaleUpLogsWarnOnDialFailure(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5

	var calls atomic.Int32
	dial := func(_ context.Context) (*fakeConn, error) {
		if calls.Add(1) > 1 {
			return nil, errors.New("dial failed")
		}
		return &fakeConn{id: 1}, nil
	}
	p := NewPool(context.Background(), cfg, dial, zap.NewNop(), "test-pool", nil)
	attachFakeWatcher(p)
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()

	p.TryScaleUp(p.LoadConns()[0])

	require.Eventually(t, func() bool {
		return calls.Load() >= 2
	}, time.Second, time.Millisecond)
	assert.Len(t, p.LoadConns(), 1, "the failed scale-up dial must not add a connection")
}

func TestScaler_RefreshMetricsResetsMaxLoggedFlagBelowCap(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConnections = 2
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1))
	defer func() { p.Stop(); p.Wait() }()

	p.atMaxConnectionsLogged.Store(true)
	p.RefreshMetrics()
	assert.False(t, p.atMaxConnectionsLogged.Load())
}

// TestScaler_ConcurrentCleanupAndReactivationRace ports transport/grpc's
// dedicated stress test for the race between CleanupIdleConns (which cancels
// idle connections past their timeout) and ReactivateIdleConn (which
// transitions idle connections back to active):
//
//	Time 1 CleanupIdleConns:   reads isScaling==0, adds c to toClose
//	Time 2 ReactivateIdleConn: CAS idle->active
//	Time 3 CleanupIdleConns:   TransitionState(idle->closing) fails -> skipped
//
// Run with -race to catch any unsynchronised read/write; the invariant
// checked on every iteration is that an Active connection must never have a
// cancelled context.
func TestScaler_ConcurrentCleanupAndReactivationRace(t *testing.T) {
	const iterations = 500
	for i := 0; i < iterations; i++ {
		cfg := baseConfig()
		cfg.IdleTimeout = time.Second
		p, _ := newTestPool(t, cfg)
		require.NoError(t, p.Start(1))

		c := p.LoadConns()[0]
		require.True(t, c.TransitionState(StateActive, StateDraining))
		require.True(t, c.TransitionState(StateDraining, StateIdle))
		atomic.StoreInt64(&c.lastIdleAtNano, time.Now().Add(-10*time.Minute).UnixNano())

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.CleanupIdleConns()
		}()
		go func() {
			defer wg.Done()
			p.ReactivateIdleConn()
		}()
		wg.Wait()

		if c.GetState() == StateActive {
			assert.NoError(t, c.Context().Err(),
				"active connection must not have a cancelled context (iteration %d)", i)
		}

		p.Stop()
		p.Wait()
	}
}

// TestScaler_ConcurrentScaleDownAndScaleUpRace ports transport/grpc's
// dedicated stress test verifying that concurrent MaybeScaleDown and
// TryScaleUp calls on the same pool do not corrupt connection state: the CAS
// in each (active->draining for scale-down, draining->active or dial for
// scale-up) must ensure only one winner per transition, so a connection is
// never simultaneously draining and selectable by PickConn.
func TestScaler_ConcurrentScaleDownAndScaleUpRace(t *testing.T) {
	const iterations = 500
	for i := 0; i < iterations; i++ {
		cfg := baseConfig()
		cfg.MinConnections = 1
		cfg.MaxConnections = 5
		cfg.MaxConcurrentStreams = 100
		cfg.ScaleUpThreshold = 0.8
		cfg.ScaleDownGap = 0.1

		p, _ := newTestPool(t, cfg)
		require.NoError(t, p.Start(2))

		conns := p.LoadConns()
		for range 80 {
			conns[0].IncStreamCount()
		}
		for range 80 {
			conns[1].IncStreamCount()
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.MaybeScaleDown()
		}()
		go func() {
			defer wg.Done()
			p.TryScaleUp(conns[0])
		}()
		wg.Wait()

		picked := p.PickConn()
		for _, c := range p.LoadConns() {
			if c.GetState() == StateDraining {
				assert.NotSame(t, c, picked,
					"iteration %d: draining connection must not be picked as active", i)
			}
		}

		p.Stop()
		p.Wait()
	}
}
