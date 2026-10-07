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
)

// fakeConn is a minimal Conn implementation for tests.
type fakeConn struct {
	id     int
	closed atomic.Bool
}

func (c *fakeConn) Close() error {
	c.closed.Store(true)
	return nil
}

// attachFakeWatcher wires a fake per-connection watcher goroutine onto
// OnConnAdded: it waits for the wrapper's context to be cancelled, then
// closes the connection, removes it from the pool, refreshes metrics, closes
// StoppedC, and finally reports itself done to the pool's connWg. Without
// this, Stop/Wait would hang forever, since AddConn's connWg.Add(1) is only
// ever balanced by a watcher calling ConnDone.
func attachFakeWatcher(p *Pool[*fakeConn]) {
	p.OnConnAdded = func(w *Wrapper[*fakeConn]) {
		<-w.Context().Done()
		_ = w.Conn.Close()
		p.Remove(w)
		p.RefreshMetrics()
		close(w.StoppedC)
		p.ConnDone()
	}
}

// liveConfig is a Config source whose value can be swapped while a pool is
// running, to exercise config changes taking effect on the next decision.
type liveConfig struct{ v atomic.Pointer[Config] }

func newLiveConfig(c Config) *liveConfig {
	l := &liveConfig{}
	l.set(c)
	return l
}

func (l *liveConfig) get() Config  { return *l.v.Load() }
func (l *liveConfig) set(c Config) { l.v.Store(&c) }

// fixedConfig returns a Config accessor that always yields cfg.
func fixedConfig(cfg Config) func() Config {
	return func() Config { return cfg }
}

// newTestPool builds a Pool with a no-op logger and a fake watcher wired via
// attachFakeWatcher.
func newTestPool(t *testing.T, cfg Config) (*Pool[*fakeConn], *atomic.Int32) {
	t.Helper()
	return newTestPoolLive(t, newLiveConfig(cfg))
}

// newTestPoolLive is newTestPool with a config the test can change later.
func newTestPoolLive(t *testing.T, lc *liveConfig) (*Pool[*fakeConn], *atomic.Int32) {
	t.Helper()
	var nextID atomic.Int32
	dial := func(_ context.Context) (*fakeConn, error) {
		return &fakeConn{id: int(nextID.Add(1))}, nil
	}
	p := NewPool(context.Background(), lc.get, dial, zap.NewNop(), "test-pool", nil)
	attachFakeWatcher(p)
	return p, &nextID
}

func baseConfig() Config {
	return Config{
		DynamicScalingEnabled:  true,
		MaxConcurrentStreams:   250,
		ScaleUpThreshold:       0.8,
		ScaleDownGap:           0.1,
		MinConnections:         1,
		MaxConnections:         5,
		IdleTimeout:            15 * time.Minute,
		ScalingMonitorInterval: 30 * time.Second,
	}
}

func TestPool_StartDialsInitialConns(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(3, false))
	defer func() { p.Stop(); p.Wait() }()

	assert.Len(t, p.LoadConns(), 3)
	assert.EqualValues(t, 3, p.connCount.Load())
}

func TestPool_StartPropagatesDialError(t *testing.T) {
	dial := func(_ context.Context) (*fakeConn, error) {
		return nil, errors.New("dial failed")
	}
	p := NewPool(context.Background(), fixedConfig(baseConfig()), dial, zap.NewNop(), "test-pool", nil)
	err := p.Start(1, false)
	assert.Error(t, err)
	assert.ErrorIs(t, p.ctx.Err(), context.Canceled, "a failed initial dial must cancel the pool")
}

func TestPool_PickConnPicksLeastLoaded(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(3, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	conns[0].IncStreamCount()
	conns[0].IncStreamCount()
	conns[1].IncStreamCount()
	// conns[2] has zero load; PickConn should choose it.

	picked := p.PickConn()
	require.NotNil(t, picked)
	assert.Same(t, conns[2], picked)
}

func TestPool_PickConnIgnoresInactive(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	require.True(t, conns[0].TransitionState(StateActive, StateDraining))

	picked := p.PickConn()
	require.NotNil(t, picked)
	assert.Same(t, conns[1], picked)
}

func TestPool_PickConnNilWhenEmpty(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(0, false))
	defer func() { p.Stop(); p.Wait() }()

	assert.Nil(t, p.PickConn())
}

func TestPool_CancelDrivesWatcherToCloseAndDropFromSlice(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(2, false))
	defer func() { p.Stop(); p.Wait() }()

	conns := p.LoadConns()
	toRemove := conns[0]
	toRemove.Cancel()
	<-toRemove.StoppedC

	assert.True(t, toRemove.Conn.closed.Load())
	assert.Len(t, p.LoadConns(), 1)
	assert.NotContains(t, p.LoadConns(), toRemove)
	assert.EqualValues(t, 1, p.connCount.Load())

	// Removing twice (once via the watcher above, once directly) is a safe
	// no-op.
	p.Remove(toRemove)
	assert.Len(t, p.LoadConns(), 1)
}

func TestPool_RemoveIsPureCASWithNoSideEffects(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	p.OnConnAdded = nil // no watcher: verify Remove itself has no side effects
	require.NoError(t, p.Start(2, false))
	// AddConn's connWg.Add(1) for each connection is normally balanced by a
	// watcher spawned from OnConnAdded calling ConnDone; simulate that here
	// since this test intentionally has no watcher, so Stop/Wait below can
	// still terminate cleanly.
	defer func() {
		p.ConnDone()
		p.ConnDone()
		p.Stop()
		p.Wait()
	}()

	conns := p.LoadConns()
	toRemove := conns[0]
	p.Remove(toRemove)

	assert.Len(t, p.LoadConns(), 1)
	assert.NotContains(t, p.LoadConns(), toRemove)
	assert.False(t, toRemove.Conn.closed.Load(), "Remove must not close the connection")
	select {
	case <-toRemove.StoppedC:
		t.Fatal("Remove must not close StoppedC")
	default:
	}

	// Removing a connection not present in the pool is a safe no-op.
	p.Remove(toRemove)
	assert.Len(t, p.LoadConns(), 1)
}

func TestPool_StopClosesAllConnsAndUnblocksWait(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(3, false))

	conns := p.LoadConns()
	p.Stop()
	p.Wait()

	for _, c := range conns {
		assert.True(t, c.Conn.closed.Load())
	}
	assert.Empty(t, p.LoadConns())
}

func TestPool_AddConnAfterStopFails(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(1, false))
	p.Stop()
	p.Wait()

	_, err := p.AddConn()
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPool_ZeroValueLoadConnsAndRemoveAreNilSafe(t *testing.T) {
	var p Pool[*fakeConn]
	assert.Nil(t, p.LoadConns())
	// Must not panic even though connsPtr was never initialized via NewPool.
	p.Remove(&Wrapper[*fakeConn]{})
}

func TestPool_StartTeardownZeroesRealMetricsReporter(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), Transport: "http2"})
	reporter := NewReporter(shared)

	var nextID atomic.Int32
	dial := func(_ context.Context) (*fakeConn, error) {
		return &fakeConn{id: int(nextID.Add(1))}, nil
	}
	p := NewPool(context.Background(), fixedConfig(baseConfig()), dial, zap.NewNop(), "test-pool", reporter)
	attachFakeWatcher(p)

	require.NoError(t, p.Start(2, false))
	p.Stop()
	p.Wait()

	snap := root.Snapshot()
	for _, g := range snap.Gauges {
		if g.Name == "conn_pool_active_connections" {
			assert.EqualValues(t, 0, g.Value)
		}
	}
}

func TestPool_ConcurrentPickAndStreamCounting(t *testing.T) {
	p, _ := newTestPool(t, baseConfig())
	require.NoError(t, p.Start(4, false))
	defer func() { p.Stop(); p.Wait() }()

	done := make(chan struct{})
	for range 20 {
		go func() {
			defer func() { done <- struct{}{} }()
			for range 50 {
				c := p.PickConn()
				if c == nil {
					continue
				}
				c.IncStreamCount()
				c.DecStreamCount()
			}
		}()
	}
	for range 20 {
		<-done
	}
}

// --- ported from the former transport/grpc metrics tests ---

func TestPool_TeardownZeroesAllSharedGaugesAndKeepsTags(t *testing.T) {
	root := metrics.New()
	shared := NewMetrics(MetricsParams{Meter: root.Scope(), ServiceName: "test-svc", Transport: "grpc"})
	p := NewPool(context.Background(), fixedConfig(baseConfig()),
		func(context.Context) (*fakeConn, error) { return &fakeConn{}, nil },
		zap.NewNop(), "test-pool", NewReporter(shared))
	attachFakeWatcher(p)
	require.NoError(t, p.Start(3, false))

	// Spread connections across all three gauges before stopping.
	conns := p.LoadConns()
	require.True(t, conns[1].TransitionState(StateActive, StateDraining))
	require.True(t, conns[2].TransitionState(StateActive, StateDraining))
	require.True(t, conns[2].TransitionState(StateDraining, StateIdle))
	p.RefreshMetrics()
	active, draining, idle := shared.connectionCount.Load(), shared.drainingConnectionCount.Load(), shared.idleConnectionCount.Load()
	require.EqualValues(t, 1, active)
	require.EqualValues(t, 1, draining)
	require.EqualValues(t, 1, idle)

	p.Stop()
	p.Wait()

	assert.EqualValues(t, 0, shared.connectionCount.Load())
	assert.EqualValues(t, 0, shared.drainingConnectionCount.Load())
	assert.EqualValues(t, 0, shared.idleConnectionCount.Load())
	snap := root.Snapshot()
	require.NotEmpty(t, snap.Gauges)
	for _, g := range snap.Gauges {
		assert.Equal(t, "yarpc", g.Tags["component"], "gauge %s", g.Name)
		assert.Equal(t, "test-svc", g.Tags["service"], "gauge %s", g.Name)
		assert.Equal(t, "grpc", g.Tags["transport"], "gauge %s", g.Name)
	}
}

// TestPool_DynamicScalingTeardownRace hammers the scale-up and scale-down
// paths while stopping a dynamically scaled pool and checks the shared gauges
// return to zero. Run with -race and a high -count to stress the teardown race.
func TestPool_DynamicScalingTeardownRace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping teardown race stress test in short mode")
	}

	cfg := baseConfig()
	cfg.MinConnections = 2
	cfg.MaxConnections = 4
	cfg.MaxConcurrentStreams = 100

	const iterations = 50
	for range iterations {
		shared := NewMetrics(MetricsParams{Meter: metrics.New().Scope(), Transport: "grpc"})
		var nextID atomic.Int32
		p := NewPool(context.Background(), fixedConfig(cfg),
			func(context.Context) (*fakeConn, error) { return &fakeConn{id: int(nextID.Add(1))}, nil },
			zap.NewNop(), "test-pool", NewReporter(shared))
		attachFakeWatcher(p)
		require.NoError(t, p.Start(2, true))

		var wg sync.WaitGroup
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 100 {
					conn := p.PickConn()
					if conn == nil {
						continue
					}
					p.TryScaleUp(conn)
					p.EvaluateScaling()
					atomic.StoreInt32(&conn.streamCount, 85)
				}
			}()
		}

		time.Sleep(5 * time.Millisecond)
		p.Stop()
		wg.Wait()
		p.Wait()

		assert.EqualValues(t, 0, shared.connectionCount.Load())
		assert.EqualValues(t, 0, shared.drainingConnectionCount.Load())
		assert.EqualValues(t, 0, shared.idleConnectionCount.Load())
	}
}
