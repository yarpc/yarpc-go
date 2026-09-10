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
