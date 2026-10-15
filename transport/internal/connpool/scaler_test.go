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
	"go.uber.org/net/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestScaler_TryScaleUpDialsWhenAboveThreshold(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5 // threshold = 0 -> immediately eligible
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	p.TryScaleUp(conns[0])

	require.Eventually(t, func() bool {
		return len(p.LoadConns()) == 2
	}, time.Second, time.Millisecond)
}

func TestScaler_TryScaleUpNoopBelowThreshold(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1, false))
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
	require.NoError(t, p.Start(1, false))
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
	require.NoError(t, p.Start(2, false))
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
	require.NoError(t, p.Start(2, false))
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
	require.NoError(t, p.Start(2, false))
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
	require.NoError(t, p.Start(2, false))
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

func TestScaler_MaybeScaleDownNeverBelowMinConnections(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	p.MaybeScaleDown()

	for _, c := range p.LoadConns() {
		assert.True(t, c.IsActive())
	}
}

func TestScaler_CleanupIdleConnsAdvancesDrainedToIdle(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1, false))
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
	require.NoError(t, p.Start(2, false))
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

func TestScaler_CleanupIdleConnsSkippedWhileScaling(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = 1 * time.Millisecond
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2, false))
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
	require.NoError(t, p.Start(3, false))
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
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	require.True(t, conns[0].TransitionState(StateActive, StateDraining))

	ok := p.ReactivateIdleConn()
	require.True(t, ok)
	assert.Equal(t, StateActive, conns[0].GetState())
}

func TestScaler_ReactivateIdleConnFalseWhenNoneAvailable(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	assert.False(t, p.ReactivateIdleConn())
}

func TestScaler_RunScalingMonitorExitsOnStop(t *testing.T) {
	cfg := baseConfig()
	cfg.ScalingMonitorInterval = time.Hour // never ticks in this test's lifetime
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, true))

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
	require.NoError(t, p.Start(1, true))
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
	require.NoError(t, p.Start(1, true))
	defer func() { p.Stop(); p.Wait() }()
	// No assertion beyond "does not panic"; exercises the interval<=0 branch
	// that falls back to defaultScalingMonitorInterval.
}

func TestScaler_EvaluateScalingRunsCleanupThenScaleDown(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = 10 * time.Millisecond
	cfg.MinConnections = 1
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2, false))
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
	require.NoError(t, p.Start(1, false))
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
	require.NoError(t, p.Start(1, false))
	conn := p.LoadConns()[0]
	p.Stop()
	p.Wait()

	p.TryScaleUp(conn)
	assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling))
	assert.Equal(t, int32(0), p.addingCount.Load(), "the guard must be released")
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
	p := NewPool(context.Background(), fixedConfig(cfg), dial, zap.NewNop(), "test-pool", nil)
	attachFakeWatcher(p)
	require.NoError(t, p.Start(1, false))
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
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.atMaxConnectionsLogged.Store(true)
	p.RefreshMetrics()
	assert.False(t, p.atMaxConnectionsLogged.Load())
}

func newPoolWithDial(cfg func() Config, dial func(context.Context) (*fakeConn, error)) *Pool[*fakeConn] {
	p := NewPool(context.Background(), cfg, dial, zap.NewNop(), "test-pool", nil)
	attachFakeWatcher(p)
	return p
}

func TestScaleDownFloor(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		min, max      int
		wantTarget    int
		wantScaleDown int
	}{
		{name: "scaling off keeps one", enabled: false, min: 3, max: 5, wantTarget: 0, wantScaleDown: 1},
		{name: "zero min keeps one", enabled: true, min: 0, max: 5, wantTarget: 0, wantScaleDown: 1},
		{name: "min within max", enabled: true, min: 3, max: 5, wantTarget: 3, wantScaleDown: 3},
		{name: "min capped by max", enabled: true, min: 9, max: 5, wantTarget: 5, wantScaleDown: 5},
		{name: "unbounded max", enabled: true, min: 3, max: 0, wantTarget: 3, wantScaleDown: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{DynamicScalingEnabled: tt.enabled, MinConnections: tt.min, MaxConnections: tt.max}
			assert.Equal(t, tt.wantTarget, minConnectionTarget(cfg))
			assert.Equal(t, tt.wantScaleDown, ScaleDownFloor(cfg))
		})
	}
}

func TestScaler_MaybeScaleDownScalingOffDrainsLeastLoadedRegardlessOfLoad(t *testing.T) {
	cfg := baseConfig()
	cfg.DynamicScalingEnabled = false
	cfg.MaxConcurrentStreams = 100
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	for range 90 {
		conns[0].IncStreamCount()
	}
	for range 80 {
		conns[1].IncStreamCount()
	}

	p.MaybeScaleDown()
	assert.Equal(t, StateActive, conns[0].GetState())
	assert.Equal(t, StateDraining, conns[1].GetState(), "least-loaded extra drains even under heavy load")

	// Down to a single active connection: nothing further to drain.
	p.MaybeScaleDown()
	assert.Equal(t, StateActive, conns[0].GetState())
}

func TestScaler_EvaluateScalingFillsToMinConnections(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 3
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.EvaluateScaling()

	assert.Len(t, p.LoadConns(), 3)
	assert.Equal(t, 3, p.activeConnCount())
	assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling))
}

func TestScaler_EvaluateScalingDoesNotFillWhenScalingDisabled(t *testing.T) {
	cfg := baseConfig()
	cfg.DynamicScalingEnabled = false
	cfg.MinConnections = 3
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.EvaluateScaling()

	assert.Len(t, p.LoadConns(), 1)
}

func TestScaler_FillReactivatesIdleBeforeDialing(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	idle := p.LoadConns()[1]
	require.True(t, idle.TransitionState(StateActive, StateDraining))
	require.True(t, idle.TransitionState(StateDraining, StateIdle))
	idle.setIdleNow()

	dialedBefore := nextID.Load()
	p.ensureMinConnections()

	assert.Equal(t, StateActive, idle.GetState())
	assert.Equal(t, dialedBefore, nextID.Load(), "must reuse the idle connection instead of dialing")
	assert.Len(t, p.LoadConns(), 2)
}

func TestScaler_FillCapsAtMaxConnections(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 3
	cfg.MaxConnections = 2
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.ensureMinConnections()
	assert.Len(t, p.LoadConns(), 2, "target is MinConnections capped by MaxConnections")
}

func TestScaler_FillStopsAtMaxWhenNothingReactivatable(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	cfg.MaxConnections = 2
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	// Closing is terminal, so these cannot be reactivated, yet they still
	// count against MaxConnections until their watchers remove them.
	for _, c := range p.LoadConns() {
		require.True(t, c.TransitionState(StateActive, StateDraining))
		require.True(t, c.TransitionState(StateDraining, StateIdle))
		require.True(t, c.TransitionState(StateIdle, StateClosing))
	}

	dialedBefore := nextID.Load()
	p.fillToMinConnections()
	assert.Equal(t, dialedBefore, nextID.Load())
}

func TestScaler_FillReturnsOnDialError(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 3
	var calls atomic.Int32
	p := newPoolWithDial(fixedConfig(cfg), func(context.Context) (*fakeConn, error) {
		if calls.Add(1) > 1 {
			return nil, errors.New("dial failed")
		}
		return &fakeConn{id: 1}, nil
	})
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.ensureMinConnections()

	assert.Len(t, p.LoadConns(), 1)
	assert.Equal(t, int32(2), calls.Load(), "fill gives up after the first failed dial")
	assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling))
}

func TestScaler_EnsureMinConnectionsNoopCases(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()
	dialedBefore := nextID.Load()

	atomic.StoreInt32(&p.isScaling, 1)
	p.ensureMinConnections()
	atomic.StoreInt32(&p.isScaling, 0)
	assert.Equal(t, dialedBefore, nextID.Load(), "must not fill while another scale-up is in flight")

	p.shutdownStarted.Store(true)
	p.ensureMinConnections()
	p.shutdownStarted.Store(false)
	assert.Equal(t, dialedBefore, nextID.Load(), "must not fill once shutdown has started")
	assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling))
}

func TestScaler_TryScaleUpFillsMinConnectionsWithoutLoad(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.TryScaleUp(nil)

	require.Eventually(t, func() bool {
		return len(p.LoadConns()) == 2
	}, time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	assert.Len(t, p.LoadConns(), 2, "no load-driven dial once the minimum is met")
}

func TestScaler_TryScaleUpStopsWhenScalingDisabledDuringFill(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5 // threshold 0: load check would otherwise dial again
	lc := newLiveConfig(cfg)

	var calls atomic.Int32
	p := newPoolWithDial(lc.get, func(context.Context) (*fakeConn, error) {
		if calls.Add(1) == 2 {
			off := cfg
			off.DynamicScalingEnabled = false
			lc.set(off)
		}
		return &fakeConn{id: int(calls.Load())}, nil
	})
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.TryScaleUp(nil)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&p.isScaling) == 0 && calls.Load() == 2
	}, time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(2), calls.Load())
}

func TestScaler_TryScaleUpReturnsWhenNoActiveConnAfterFill(t *testing.T) {
	cfg := baseConfig()
	var calls atomic.Int32
	p := newPoolWithDial(fixedConfig(cfg), func(context.Context) (*fakeConn, error) {
		if calls.Add(1) > 1 {
			return nil, errors.New("dial failed")
		}
		return &fakeConn{id: 1}, nil
	})
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	c := p.LoadConns()[0]
	require.True(t, c.TransitionState(StateActive, StateDraining))
	require.True(t, c.TransitionState(StateDraining, StateIdle))
	require.True(t, c.TransitionState(StateIdle, StateClosing))

	p.TryScaleUp(nil)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&p.isScaling) == 0 && calls.Load() == 2
	}, time.Second, time.Millisecond)
	assert.Len(t, p.LoadConns(), 1)
}

func TestScaler_TryScaleUpDoesNotStartMonitor(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.TryScaleUp(p.LoadConns()[0])
	assert.False(t, p.monitorStarted.Load(), "the monitor is started by Start, not by TryScaleUp")
}

func TestScaler_StartScalingMonitorStartsOnce(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1, true))
	assert.True(t, p.monitorStarted.Load())

	p.startScalingMonitor() // already started: must not add another goroutine

	p.Stop()
	p.Wait()

	p.monitorStarted.Store(false)
	p.startScalingMonitor()
	assert.False(t, p.monitorStarted.Load(), "must not start a monitor after shutdown")
}

func TestScaler_StartScalingMonitorSkippedOnceShutdownStarted(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.shutdownStarted.Store(true)
	p.startScalingMonitor()
	p.shutdownStarted.Store(false)

	assert.False(t, p.monitorStarted.Load())
	assert.Equal(t, int32(0), p.addingCount.Load(), "the guard must be released")
}

func TestScaler_MonitorEvaluatesImmediatelyOnStart(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	cfg.ScalingMonitorInterval = time.Hour
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, true))
	defer func() { p.Stop(); p.Wait() }()

	require.Eventually(t, func() bool {
		return len(p.LoadConns()) == 2
	}, time.Second, time.Millisecond, "first pass must not wait for the interval")
}

func TestScaler_ScalingMonitorIntervalFollowsLiveConfig(t *testing.T) {
	cfg := baseConfig()
	lc := newLiveConfig(cfg)
	p, _ := newTestPoolLive(t, lc)

	set := func(d time.Duration) {
		c := cfg
		c.ScalingMonitorInterval = d
		lc.set(c)
	}

	set(0)
	assert.Equal(t, defaultScalingMonitorInterval, p.scalingMonitorInterval())
	assert.False(t, p.intervalClampWarned.Load())

	set(time.Second)
	assert.Equal(t, defaultScalingMonitorInterval, p.scalingMonitorInterval())
	assert.True(t, p.intervalClampWarned.Load(), "clamp warns once")
	assert.Equal(t, defaultScalingMonitorInterval, p.scalingMonitorInterval())
	assert.True(t, p.intervalClampWarned.Load())

	set(time.Minute)
	assert.Equal(t, time.Minute, p.scalingMonitorInterval())
	assert.False(t, p.intervalClampWarned.Load(), "a valid interval re-arms the warning")
}

func TestScaler_CleanupIdleConnsUsesLiveIdleTimeout(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = time.Hour
	lc := newLiveConfig(cfg)
	p, _ := newTestPoolLive(t, lc)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	c := p.LoadConns()[0]
	require.True(t, c.TransitionState(StateActive, StateDraining))
	require.True(t, c.TransitionState(StateDraining, StateIdle))
	c.setIdleNow()
	time.Sleep(5 * time.Millisecond)

	p.CleanupIdleConns()
	assert.Equal(t, StateIdle, c.GetState())

	short := cfg
	short.IdleTimeout = time.Millisecond
	lc.set(short)
	p.CleanupIdleConns()
	assert.Equal(t, StateClosing, c.GetState())
}

func TestScaler_RefreshMetricsUsesLiveMaxConnections(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConnections = 1
	lc := newLiveConfig(cfg)
	p, _ := newTestPoolLive(t, lc)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.atMaxConnectionsLogged.Store(true)
	p.RefreshMetrics()
	assert.True(t, p.atMaxConnectionsLogged.Load(), "still at the cap")

	raised := cfg
	raised.MaxConnections = 5
	lc.set(raised)
	p.RefreshMetrics()
	assert.False(t, p.atMaxConnectionsLogged.Load())
}

func TestScaler_LogMessagesCarryPrefixAndPeerField(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	cfg := baseConfig()
	cfg.DynamicScalingEnabled = false
	p := newPoolWithDial(fixedConfig(cfg), func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil })
	p.Logger = zap.New(core)
	p.LogPrefix = "grpc"
	p.ID = "host:1234"
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	p.MaybeScaleDown()

	entries := logs.FilterMessage("grpc: scaling down connection pool; marked connection for draining").All()
	require.Len(t, entries, 1)
	assert.Equal(t, "host:1234", entries[0].ContextMap()["peer"])
}

func TestScaler_NilLoggerSkipsFirstEvaluationAndClampWarning(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	cfg.ScalingMonitorInterval = time.Second // below the minimum
	p := newPoolWithDial(fixedConfig(cfg), func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil })
	p.Logger = nil
	require.NoError(t, p.Start(1, true))
	defer func() { p.Stop(); p.Wait() }()

	time.Sleep(30 * time.Millisecond)
	assert.Len(t, p.LoadConns(), 1, "no immediate evaluation without a logger")

	assert.Equal(t, defaultScalingMonitorInterval, p.scalingMonitorInterval())
	assert.False(t, p.intervalClampWarned.Load(), "no clamp warning without a logger")
}

// --- ported from the former transport/grpc pool/scaler tests ---

// makeWrapper builds a standalone wrapper in the given state carrying the given
// in-flight stream count.
func makeWrapper(state State, streams int32) *Wrapper[*fakeConn] {
	w := newWrapper(context.Background(), &fakeConn{})
	w.setState(state)
	atomic.StoreInt32(&w.streamCount, streams)
	return w
}

// makeIdleWrapper builds an idle wrapper whose idle timestamp is idleAt (the
// zero time leaves it unset).
func makeIdleWrapper(idleAt time.Time) *Wrapper[*fakeConn] {
	w := makeWrapper(StateIdle, 0)
	if !idleAt.IsZero() {
		atomic.StoreInt64(&w.lastIdleAtNano, idleAt.UnixNano())
	}
	return w
}

// newBarePool builds a Pool holding exactly the given hand-built wrappers. It
// is never started, so there are no background goroutines and no watchers:
// state transitions on the wrappers are the only moving parts. Use it for
// decision-logic tests that never dial.
func newBarePool(t *testing.T, cfg Config, reporter *Reporter, conns ...*Wrapper[*fakeConn]) *Pool[*fakeConn] {
	t.Helper()
	dial := func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil }
	p := NewPool(context.Background(), fixedConfig(cfg), dial, zap.NewNop(), "test-pool", reporter)
	t.Cleanup(p.Stop)
	snap := append([]*Wrapper[*fakeConn](nil), conns...)
	p.connsPtr.Store(&snap)
	p.connCount.Store(int32(len(snap)))
	return p
}

func newTestReporter() (*Metrics, *Reporter) {
	shared := NewMetrics(MetricsParams{Meter: metrics.New().Scope(), Transport: "grpc"})
	return shared, NewReporter(shared)
}

// reportedGauges reads the shared gauges with atomic loads. Tests must not use
// Root.Snapshot() while pool goroutines may still be adding to the gauges.
func reportedGauges(m *Metrics) (active, draining, idle int64) {
	return m.connectionCount.Load(), m.drainingConnectionCount.Load(), m.idleConnectionCount.Load()
}

func stateCount(conns []*Wrapper[*fakeConn], s State) int {
	n := 0
	for _, c := range conns {
		if c.GetState() == s {
			n++
		}
	}
	return n
}

func TestScaler_CleanupIdleConnsPerConnBranches(t *testing.T) {
	const shortTimeout = 100 * time.Millisecond

	tests := []struct {
		desc          string
		build         func() []*Wrapper[*fakeConn]
		idleTimeout   time.Duration
		wantStates    []State
		wantCancelled []int
	}{
		{
			desc:        "empty pool is a no-op",
			build:       func() []*Wrapper[*fakeConn] { return nil },
			idleTimeout: time.Minute,
		},
		{
			desc: "active connection is untouched",
			build: func() []*Wrapper[*fakeConn] {
				return []*Wrapper[*fakeConn]{makeWrapper(StateActive, 5)}
			},
			idleTimeout: time.Minute,
			wantStates:  []State{StateActive},
		},
		{
			desc: "draining with in-flight streams stays draining",
			build: func() []*Wrapper[*fakeConn] {
				return []*Wrapper[*fakeConn]{makeWrapper(StateDraining, 3)}
			},
			idleTimeout: time.Minute,
			wantStates:  []State{StateDraining},
		},
		{
			desc: "draining with zero streams advances to idle but is not yet timed out",
			build: func() []*Wrapper[*fakeConn] {
				return []*Wrapper[*fakeConn]{makeWrapper(StateDraining, 0)}
			},
			idleTimeout: time.Hour,
			wantStates:  []State{StateIdle},
		},
		{
			desc: "idle with unset idle timestamp is never collected",
			build: func() []*Wrapper[*fakeConn] {
				return []*Wrapper[*fakeConn]{makeIdleWrapper(time.Time{})}
			},
			idleTimeout: shortTimeout,
			wantStates:  []State{StateIdle},
		},
		{
			desc: "idle within timeout is not collected",
			build: func() []*Wrapper[*fakeConn] {
				return []*Wrapper[*fakeConn]{makeIdleWrapper(time.Now())}
			},
			idleTimeout: time.Hour,
			wantStates:  []State{StateIdle},
		},
		{
			desc: "idle past timeout is claimed and cancelled",
			build: func() []*Wrapper[*fakeConn] {
				return []*Wrapper[*fakeConn]{makeIdleWrapper(time.Now().Add(-10 * time.Minute))}
			},
			idleTimeout:   shortTimeout,
			wantStates:    []State{StateClosing},
			wantCancelled: []int{0},
		},
		{
			desc: "mixed pool applies the right behavior per connection",
			build: func() []*Wrapper[*fakeConn] {
				return []*Wrapper[*fakeConn]{
					makeWrapper(StateActive, 10),
					makeWrapper(StateDraining, 0),                     // -> idle
					makeIdleWrapper(time.Now().Add(-5 * time.Minute)), // -> closing + cancel
				}
			},
			idleTimeout:   shortTimeout,
			wantStates:    []State{StateActive, StateIdle, StateClosing},
			wantCancelled: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			conns := tt.build()
			cfg := baseConfig()
			cfg.IdleTimeout = tt.idleTimeout
			p := newBarePool(t, cfg, nil, conns...)

			p.CleanupIdleConns()

			require.Len(t, p.LoadConns(), len(tt.wantStates))
			for i, want := range tt.wantStates {
				assert.Equal(t, want, conns[i].GetState(), "conn[%d] state", i)
			}
			cancelled := make(map[int]bool)
			for _, i := range tt.wantCancelled {
				cancelled[i] = true
			}
			for i, c := range conns {
				if cancelled[i] {
					assert.Error(t, c.Context().Err(), "conn[%d] context should be cancelled", i)
				} else {
					assert.NoError(t, c.Context().Err(), "conn[%d] context should not be cancelled", i)
				}
			}
		})
	}
}

func TestScaler_CleanupIdleConnsSkippedWhileScalingKeepsConnIntact(t *testing.T) {
	cfg := baseConfig()
	cfg.IdleTimeout = time.Second
	w := makeIdleWrapper(time.Now().Add(-10 * time.Minute))
	p := newBarePool(t, cfg, nil, w)

	atomic.StoreInt32(&p.isScaling, 1)
	p.CleanupIdleConns()

	assert.NoError(t, w.Context().Err(), "idle connection must not be cancelled while scaling")
	assert.Equal(t, StateIdle, w.GetState(), "connection state must be unchanged")
}

func TestScaler_CleanupIdleConnsIgnoresFormerDrainingConnThatIsNowActive(t *testing.T) {
	// The connection looks like a draining/zero-stream candidate at the start
	// of the pass; here it has already been reactivated, so cleanup must
	// neither move it to idle nor cancel it.
	w := makeWrapper(StateDraining, 0)
	w.setState(StateActive)
	cfg := baseConfig()
	cfg.IdleTimeout = time.Hour
	p := newBarePool(t, cfg, nil, w)

	p.CleanupIdleConns()

	assert.Equal(t, StateActive, w.GetState())
	assert.NoError(t, w.Context().Err(), "active connection must not be cancelled")
}

func TestScaler_MaybeScaleDownPerConnBranches(t *testing.T) {
	// threshold = int32(100 * 0.8) = 80
	scaleDownCfg := func() Config {
		return Config{
			DynamicScalingEnabled: true,
			MinConnections:        1,
			MaxConnections:        50,
			MaxConcurrentStreams:  100,
			ScaleUpThreshold:      0.8,
		}
	}
	minTwo := scaleDownCfg()
	minTwo.MinConnections = 2
	minTwo.MaxConnections = 0

	tests := []struct {
		desc       string
		conns      []*Wrapper[*fakeConn]
		cfg        Config
		wantStates []State
	}{
		{
			desc: "empty pool returns at the floor guard",
			cfg:  scaleDownCfg(),
		},
		{
			desc:       "active equals the floor so nothing drains",
			conns:      []*Wrapper[*fakeConn]{makeWrapper(StateActive, 10)},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateActive},
		},
		{
			desc:       "active below the floor so nothing drains",
			conns:      []*Wrapper[*fakeConn]{makeWrapper(StateActive, 10)},
			cfg:        minTwo,
			wantStates: []State{StateActive},
		},
		{
			desc: "only draining conns leaves them unchanged",
			conns: []*Wrapper[*fakeConn]{
				makeWrapper(StateDraining, 5),
				makeWrapper(StateDraining, 5),
			},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateDraining, StateDraining},
		},
		{
			desc: "mixed pool whose active count is at the floor does not drain",
			conns: []*Wrapper[*fakeConn]{
				makeWrapper(StateActive, 10),
				makeWrapper(StateDraining, 20),
			},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateActive, StateDraining},
		},
		{
			desc: "total streams above capacity after drain does not drain",
			conns: []*Wrapper[*fakeConn]{
				makeWrapper(StateActive, 60),
				makeWrapper(StateActive, 60),
				makeWrapper(StateActive, 60),
			},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateActive, StateActive, StateActive},
		},
		{
			// capacityAfterDrain = 80*2 = 160; total == 160 must NOT drain.
			desc: "total streams exactly equal to capacity after drain does not drain",
			conns: []*Wrapper[*fakeConn]{
				makeWrapper(StateActive, 54),
				makeWrapper(StateActive, 53),
				makeWrapper(StateActive, 53),
			},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateActive, StateActive, StateActive},
		},
		{
			desc: "low load drains the most-loaded connection",
			conns: []*Wrapper[*fakeConn]{
				makeWrapper(StateActive, 10),
				makeWrapper(StateActive, 20),
				makeWrapper(StateActive, 30),
			},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateActive, StateActive, StateDraining},
		},
		{
			desc: "most-loaded being first still selects the global maximum",
			conns: []*Wrapper[*fakeConn]{
				makeWrapper(StateActive, 30),
				makeWrapper(StateActive, 5),
				makeWrapper(StateActive, 25),
			},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateDraining, StateActive, StateActive},
		},
		{
			desc: "equal stream counts drain the first active connection",
			conns: []*Wrapper[*fakeConn]{
				makeWrapper(StateActive, 10),
				makeWrapper(StateActive, 10),
				makeWrapper(StateActive, 10),
			},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateDraining, StateActive, StateActive},
		},
		{
			desc: "an already-draining conn is excluded from candidate selection",
			conns: []*Wrapper[*fakeConn]{
				makeWrapper(StateDraining, 1),
				makeWrapper(StateActive, 5),
				makeWrapper(StateActive, 40),
				makeWrapper(StateActive, 40),
			},
			cfg:        scaleDownCfg(),
			wantStates: []State{StateDraining, StateActive, StateDraining, StateActive},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			p := newBarePool(t, tt.cfg, nil, tt.conns...)

			p.MaybeScaleDown()

			require.Len(t, p.LoadConns(), len(tt.wantStates))
			for i, want := range tt.wantStates {
				assert.Equal(t, want, tt.conns[i].GetState(), "conn[%d] state", i)
			}
		})
	}
}

func TestScaler_MaybeScaleDownLeavesPreExistingDrainingConnAlone(t *testing.T) {
	conns := []*Wrapper[*fakeConn]{
		makeWrapper(StateActive, 5),
		makeWrapper(StateActive, 5),
		makeWrapper(StateDraining, 30),
	}
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	cfg.ScaleDownGap = 0
	p := newBarePool(t, cfg, nil, conns...)

	p.MaybeScaleDown()

	assert.Equal(t, StateDraining, conns[2].GetState(), "pre-existing draining conn must not change state")
	assert.Equal(t, 1, stateCount(conns[:2], StateDraining), "exactly one active conn should be drained")
}

func TestScaler_MaybeScaleDownFloorIsMinCappedByMax(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 8
	cfg.MaxConnections = 2
	cfg.MaxConcurrentStreams = 100
	conns := []*Wrapper[*fakeConn]{
		makeWrapper(StateActive, 5),
		makeWrapper(StateActive, 5),
		makeWrapper(StateActive, 5),
	}
	p := newBarePool(t, cfg, nil, conns...)

	p.MaybeScaleDown()

	assert.Equal(t, 2, p.activeConnCount(), "scale-down floor must be min capped by max")
}

func TestScaler_MaybeScaleDownScalingOffDrainsWithHighLoadAcrossThree(t *testing.T) {
	cfg := baseConfig()
	cfg.DynamicScalingEnabled = false
	cfg.MaxConcurrentStreams = 100
	conns := []*Wrapper[*fakeConn]{
		makeWrapper(StateActive, 80),
		makeWrapper(StateActive, 80),
		makeWrapper(StateActive, 80),
	}
	p := newBarePool(t, cfg, nil, conns...)

	p.MaybeScaleDown()

	assert.Equal(t, 1, stateCount(conns, StateDraining), "scaling off must drain an extra even when load is high")
	assert.Equal(t, 2, p.activeConnCount())
}

func TestScaler_EvaluateScalingScalingOffDrainsTowardOne(t *testing.T) {
	cfg := baseConfig()
	cfg.DynamicScalingEnabled = false
	cfg.IdleTimeout = time.Hour
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(3, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	c1, c2, c3 := conns[0], conns[1], conns[2]
	atomic.StoreInt32(&c1.streamCount, 30)
	atomic.StoreInt32(&c2.streamCount, 10)
	atomic.StoreInt32(&c3.streamCount, 1)

	p.EvaluateScaling()
	assert.Equal(t, 2, p.activeConnCount())
	assert.Equal(t, StateDraining, c3.GetState(), "least-loaded extra drains first")

	p.EvaluateScaling()
	assert.Equal(t, 1, p.activeConnCount())
	assert.Equal(t, StateActive, c1.GetState())

	p.EvaluateScaling()
	assert.Equal(t, 1, p.activeConnCount(), "must not drain the last connection")
}

func TestScaler_ReactivateIdleConnEdgeCases(t *testing.T) {
	t.Run("empty pool returns false", func(t *testing.T) {
		p := newBarePool(t, baseConfig(), nil)
		assert.False(t, p.ReactivateIdleConn())
	})

	t.Run("active and context-less draining conns are not reactivatable", func(t *testing.T) {
		// A draining wrapper that carries no context cannot be proven live, so
		// the draining pass must skip it rather than dereference it.
		noCtx := &Wrapper[*fakeConn]{state: int32(StateDraining)}
		p := newBarePool(t, baseConfig(), nil, makeWrapper(StateActive, 10), noCtx)
		assert.False(t, p.ReactivateIdleConn())
		assert.Equal(t, StateDraining, noCtx.GetState())
	})

	t.Run("idle with cancelled context is skipped", func(t *testing.T) {
		w := makeWrapper(StateIdle, 0)
		w.Cancel()
		p := newBarePool(t, baseConfig(), nil, w)
		assert.False(t, p.ReactivateIdleConn())
		assert.Equal(t, StateIdle, w.GetState(), "cancelled idle conn must not be reactivated")
	})

	t.Run("reactivating an idle conn clears its idle timestamp", func(t *testing.T) {
		w := makeIdleWrapper(time.Now())
		require.False(t, w.IdleSince().IsZero())
		p := newBarePool(t, baseConfig(), nil, w)

		assert.True(t, p.ReactivateIdleConn())
		assert.Equal(t, StateActive, w.GetState())
		assert.True(t, w.IdleSince().IsZero(), "idle timestamp should be cleared after reactivation")
	})

	t.Run("skips cancelled idle and picks the first live idle", func(t *testing.T) {
		cancelled := makeWrapper(StateIdle, 0)
		cancelled.Cancel()
		live := makeWrapper(StateIdle, 0)
		p := newBarePool(t, baseConfig(), nil, cancelled, live)

		assert.True(t, p.ReactivateIdleConn())
		assert.Equal(t, StateIdle, cancelled.GetState(), "cancelled conn should remain idle")
		assert.Equal(t, StateActive, live.GetState(), "live idle conn should be reactivated")
	})

	t.Run("reactivated draining conn keeps its in-flight streams", func(t *testing.T) {
		draining := makeWrapper(StateDraining, 5)
		p := newBarePool(t, baseConfig(), nil, draining)

		assert.True(t, p.ReactivateIdleConn(), "should reactivate draining conn")
		assert.Equal(t, StateActive, draining.GetState())
		assert.EqualValues(t, 5, draining.StreamCount())
	})

	t.Run("draining conn with cancelled context is skipped", func(t *testing.T) {
		draining := makeWrapper(StateDraining, 0)
		draining.Cancel()
		p := newBarePool(t, baseConfig(), nil, draining)

		assert.False(t, p.ReactivateIdleConn(), "cancelled draining conn must not be reactivated")
		assert.Equal(t, StateDraining, draining.GetState())
	})
}

func TestScaler_TryScaleUpReactivationClearsIdleTimestamp(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	p, _ := newTestPool(t, cfg)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	idle := conns[1]
	require.True(t, idle.TransitionState(StateActive, StateDraining))
	require.True(t, idle.TransitionState(StateDraining, StateIdle))
	idle.setIdleNow()

	p.TryScaleUp(conns[0])

	require.Eventually(t, func() bool {
		return idle.GetState() == StateActive && atomic.LoadInt32(&p.isScaling) == 0
	}, time.Second, time.Millisecond)
	assert.True(t, idle.IdleSince().IsZero(), "idle timestamp should be cleared")
}

func TestScaler_TryScaleUpNilConnNoopWhenMinMet(t *testing.T) {
	p, nextID := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()
	dialedBefore := nextID.Load()

	assert.NotPanics(t, func() { p.TryScaleUp(nil) })

	assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling))
	assert.Len(t, p.LoadConns(), 1)
	assert.Equal(t, dialedBefore, nextID.Load())
}

func TestScaler_TryScaleUpAtMaxLogsOncePerSaturation(t *testing.T) {
	const atMaxMsg = "grpc: cannot scale up connection pool; at max connections"

	cfg := baseConfig()
	cfg.MaxConnections = 2
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5 // threshold 0: always over budget
	lc := newLiveConfig(cfg)
	core, logs := observer.New(zap.InfoLevel)
	p, _ := newTestPoolLive(t, lc)
	p.Logger = zap.New(core)
	p.LogPrefix = "grpc"
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	scaleUpOnce := func() {
		p.TryScaleUp(p.LoadConns()[0])
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&p.isScaling) == 0
		}, 2*time.Second, time.Millisecond)
	}

	for range 10 {
		scaleUpOnce()
	}
	assert.Equal(t, 1, logs.FilterMessage(atMaxMsg).Len(), "at-max log should fire once per saturation episode")
	assert.Len(t, p.LoadConns(), 2, "pool must not grow beyond max")

	// Leaving saturation re-arms the log; re-entering it logs again, once.
	raised := cfg
	raised.MaxConnections = 5
	lc.set(raised)
	p.RefreshMetrics()
	lc.set(cfg)
	for range 3 {
		scaleUpOnce()
	}
	assert.Equal(t, 2, logs.FilterMessage(atMaxMsg).Len(), "a new saturation episode logs once more")
}

func TestScaler_TryScaleUpDoesNotStartWorkerOnceShutdownStarted(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()
	dialedBefore := nextID.Load()

	p.shutdownStarted.Store(true)
	p.TryScaleUp(p.LoadConns()[0])
	p.shutdownStarted.Store(false)

	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(0), atomic.LoadInt32(&p.isScaling), "the scale-up slot must be released")
	assert.Equal(t, int32(0), p.addingCount.Load(), "the guard must be released")
	assert.Equal(t, dialedBefore, nextID.Load(), "no worker may run once shutdown has started")
}

func TestScaler_FillReactivatesDrainingBeforeDialing(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 1
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	c := p.LoadConns()[0]
	require.True(t, c.TransitionState(StateActive, StateDraining))
	dialedBefore := nextID.Load()

	p.EvaluateScaling()

	assert.Equal(t, StateActive, c.GetState())
	assert.Equal(t, dialedBefore, nextID.Load(), "must not dial when a draining conn can meet the minimum")
	assert.Len(t, p.LoadConns(), 1)
}

func TestScaler_FillReactivatesIdleThenDials(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	idle := p.LoadConns()[0]
	require.True(t, idle.TransitionState(StateActive, StateDraining))
	require.True(t, idle.TransitionState(StateDraining, StateIdle))
	idle.setIdleNow()
	dialedBefore := nextID.Load()

	p.EvaluateScaling()

	assert.Equal(t, StateActive, idle.GetState())
	assert.Len(t, p.LoadConns(), 2)
	assert.Equal(t, 2, p.activeConnCount())
	assert.Equal(t, dialedBefore+1, nextID.Load(), "exactly one dial tops up after the reactivation")
}

func TestScaler_TryScaleUpReactivatesIdleForMinWithoutLoad(t *testing.T) {
	cfg := baseConfig()
	cfg.MinConnections = 2
	p, nextID := newTestPool(t, cfg)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	idle := conns[1]
	require.True(t, idle.TransitionState(StateActive, StateDraining))
	require.True(t, idle.TransitionState(StateDraining, StateIdle))
	idle.setIdleNow()
	dialedBefore := nextID.Load()

	p.TryScaleUp(conns[0]) // zero load: only the minimum drives this

	require.Eventually(t, func() bool {
		return idle.GetState() == StateActive && atomic.LoadInt32(&p.isScaling) == 0
	}, time.Second, time.Millisecond)
	assert.Len(t, p.LoadConns(), 2, "must not dial when idle can meet the minimum")
	assert.Equal(t, dialedBefore, nextID.Load())
}

func TestScaler_EnsureAndFillNoPanicOnEmptyPool(t *testing.T) {
	disabled := baseConfig()
	disabled.DynamicScalingEnabled = false
	zeroMin := baseConfig()
	zeroMin.MinConnections = 0

	for name, cfg := range map[string]Config{"scaling disabled": disabled, "zero minimum": zeroMin} {
		t.Run(name, func(t *testing.T) {
			p, _ := newTestPool(t, cfg)
			require.NoError(t, p.Start(0, false))
			defer func() { p.Stop(); p.Wait() }()

			assert.NotPanics(t, p.EvaluateScaling)
			assert.NotPanics(t, p.CleanupIdleConns)
			assert.NotPanics(t, p.MaybeScaleDown)
			assert.NotPanics(t, p.ensureMinConnections)
			assert.NotPanics(t, p.fillToMinConnections)
			assert.NotPanics(t, func() { p.TryScaleUp(nil) })
			assert.Empty(t, p.LoadConns(), "nothing to fill or drain on an empty pool")
		})
	}
}

func TestScaler_RunScalingMonitorExitsOnParentContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := baseConfig()
	cfg.ScalingMonitorInterval = time.Hour
	dial := func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil }
	p := NewPool(ctx, fixedConfig(cfg), dial, zap.NewNop(), "test-pool", nil)
	attachFakeWatcher(p)
	require.NoError(t, p.Start(1, true))

	cancel()

	done := make(chan struct{})
	go func() { p.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scaling monitor did not exit after the parent context was cancelled")
	}
}

func TestScaler_RunScalingMonitorClampWarnsOnce(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	cfg := baseConfig()
	cfg.ScalingMonitorInterval = 5 * time.Second // below the minimum
	p := newPoolWithDial(fixedConfig(cfg), func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil })
	p.Logger = zap.New(core)
	p.LogPrefix = "grpc"
	p.ID = "host:1234"
	require.NoError(t, p.Start(1, true))

	p.Stop()
	p.Wait()

	require.Equal(t, 1, logs.Len(), "expected exactly one warning log")
	entry := logs.All()[0]
	assert.Contains(t, entry.Message, "scalingMonitorInterval")
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "host:1234", entry.ContextMap()["peer"])
}

func TestScaler_RefreshMetricsReportsStateCounts(t *testing.T) {
	shared, reporter := newTestReporter()
	p := newBarePool(t, baseConfig(), reporter)

	p.RefreshMetrics()
	active, draining, idle := reportedGauges(shared)
	assert.Zero(t, active)
	assert.Zero(t, draining)
	assert.Zero(t, idle)

	conns := []*Wrapper[*fakeConn]{
		makeWrapper(StateActive, 0),
		makeWrapper(StateActive, 0),
		makeWrapper(StateDraining, 0),
		makeWrapper(StateIdle, 0),
		makeWrapper(StateClosing, 0), // closing is not counted in any gauge
	}
	p.connsPtr.Store(&conns)
	p.connCount.Store(int32(len(conns)))
	p.RefreshMetrics()

	active, draining, idle = reportedGauges(shared)
	assert.EqualValues(t, 2, active)
	assert.EqualValues(t, 1, draining)
	assert.EqualValues(t, 1, idle)
}

func TestScaler_MaybeScaleDownReportsMetrics(t *testing.T) {
	shared, reporter := newTestReporter()
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100
	cfg.ScaleDownGap = 0
	p := newBarePool(t, cfg, reporter,
		makeWrapper(StateActive, 10),
		makeWrapper(StateActive, 10),
		makeWrapper(StateActive, 10),
	)

	p.MaybeScaleDown()

	assert.EqualValues(t, 1, shared.scaleDownTotal.Load(), "scale-down counter should increment")
	active, draining, _ := reportedGauges(shared)
	assert.EqualValues(t, 2, active)
	assert.EqualValues(t, 1, draining)
}

func TestScaler_CleanupIdleConnsReportsMetrics(t *testing.T) {
	shared, reporter := newTestReporter()
	cfg := baseConfig()
	cfg.IdleTimeout = time.Hour
	p := newBarePool(t, cfg, reporter,
		makeWrapper(StateActive, 5),
		makeWrapper(StateDraining, 0), // zero streams: advances to idle
	)

	p.CleanupIdleConns()

	active, draining, idle := reportedGauges(shared)
	assert.EqualValues(t, 1, active)
	assert.EqualValues(t, 0, draining)
	assert.EqualValues(t, 1, idle)
}

func TestScaler_TryScaleUpDialReportsMetrics(t *testing.T) {
	shared, reporter := newTestReporter()
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	var nextID atomic.Int32
	dial := func(context.Context) (*fakeConn, error) { return &fakeConn{id: int(nextID.Add(1))}, nil }
	p := NewPool(context.Background(), fixedConfig(cfg), dial, zap.NewNop(), "test-pool", reporter)
	attachFakeWatcher(p)
	require.NoError(t, p.Start(1, false))
	defer func() { p.Stop(); p.Wait() }()

	p.TryScaleUp(p.LoadConns()[0])

	require.Eventually(t, func() bool {
		return len(p.LoadConns()) == 2 && atomic.LoadInt32(&p.isScaling) == 0
	}, 2*time.Second, time.Millisecond)
	assert.EqualValues(t, 1, shared.scaleUpTotal.Load())
	assert.EqualValues(t, 0, shared.idleReactivationTotal.Load())
	active, _, _ := reportedGauges(shared)
	assert.EqualValues(t, 2, active, "gauge reflects the seeded connection plus the new dial")
}

func TestScaler_TryScaleUpReactivationReportsMetrics(t *testing.T) {
	shared, reporter := newTestReporter()
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 1
	cfg.ScaleUpThreshold = 0.5
	var nextID atomic.Int32
	dial := func(context.Context) (*fakeConn, error) { return &fakeConn{id: int(nextID.Add(1))}, nil }
	p := NewPool(context.Background(), fixedConfig(cfg), dial, zap.NewNop(), "test-pool", reporter)
	attachFakeWatcher(p)
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	require.True(t, conns[1].TransitionState(StateActive, StateDraining))
	require.True(t, conns[1].TransitionState(StateDraining, StateIdle))
	conns[1].setIdleNow()
	p.RefreshMetrics()
	_, _, idle := reportedGauges(shared)
	require.EqualValues(t, 1, idle)

	p.TryScaleUp(conns[0])

	require.Eventually(t, func() bool {
		return conns[1].GetState() == StateActive && atomic.LoadInt32(&p.isScaling) == 0
	}, 2*time.Second, time.Millisecond)
	assert.EqualValues(t, 1, shared.idleReactivationTotal.Load())
	assert.EqualValues(t, 0, shared.scaleUpTotal.Load(), "no new dial should happen")
	active, _, idle := reportedGauges(shared)
	assert.EqualValues(t, 2, active)
	assert.EqualValues(t, 0, idle)
}

func TestScaler_ConcurrentCleanupAndReactivationRace(t *testing.T) {
	// Guards the race between CleanupIdleConns (cancels idle connections) and
	// ReactivateIdleConn (idle -> active):
	//
	//	T1 CleanupIdleConns:  reads isScaling==0, adds c to toClose
	//	T2 ReactivateIdleConn: CAS idle->active
	//	T3 CleanupIdleConns:  would cancel an active connection without the CAS
	//
	// With the CAS exactly one of idle->closing and idle->active wins.
	cfg := baseConfig()
	cfg.IdleTimeout = time.Second

	const iterations = 500
	for i := range iterations {
		p, _ := newTestPool(t, cfg)
		require.NoError(t, p.Start(1, false))

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

		// An active connection must never have a cancelled context.
		if c.GetState() == StateActive {
			assert.NoError(t, c.Context().Err(),
				"active connection must not have a cancelled context (iteration %d)", i)
		}
		p.Stop()
		p.Wait()
	}
}

func TestScaler_ConcurrentScaleDownAndScaleUpRace(t *testing.T) {
	// MaybeScaleDown and TryScaleUp read the same snapshot; their CASes must
	// leave every connection in a consistent state, and a draining connection
	// must never be handed out by PickConn.
	cfg := baseConfig()
	cfg.MaxConcurrentStreams = 100

	const iterations = 500
	for i := range iterations {
		p, _ := newTestPool(t, cfg)
		require.NoError(t, p.Start(2, false))
		conns := p.LoadConns()
		atomic.StoreInt32(&conns[0].streamCount, 80)
		atomic.StoreInt32(&conns[1].streamCount, 80)

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
		p.Wait() // also waits for any in-flight scale-up worker
	}
}
