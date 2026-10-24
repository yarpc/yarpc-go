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
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// defaultScalingMonitorInterval is both the default and the minimum allowed
// interval for the scaling monitor. Values below this would cause excessive
// pool churn and are clamped at runtime.
const defaultScalingMonitorInterval = 30 * time.Second

// runScalingMonitor runs as a background goroutine for the lifetime of the
// pool. It periodically evaluates whether connections should be added or
// removed. It exits when the pool's context is cancelled (via Stop).
//
// The caller has already accounted for this goroutine in connWg;
// runScalingMonitor calls connWg.Done on exit.
func (p *Pool[T]) runScalingMonitor() {
	defer p.connWg.Done()

	// Apply live MinConnections / idle cleanup without waiting for the first tick.
	if p.Logger != nil {
		p.EvaluateScaling()
	}

	timer := time.NewTimer(p.scalingMonitorInterval())
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			p.EvaluateScaling()
			timer.Reset(p.scalingMonitorInterval())
		case <-p.ctx.Done():
			return
		}
	}
}

// scalingMonitorInterval returns the wait until the next monitor pass.
// Changes to Config's ScalingMonitorInterval are picked up here; values under
// the minimum are still clamped so a config flap cannot thrash the pool.
func (p *Pool[T]) scalingMonitorInterval() time.Duration {
	interval := p.Config().ScalingMonitorInterval
	switch {
	case interval <= 0:
		return defaultScalingMonitorInterval
	case interval < defaultScalingMonitorInterval:
		if p.Logger != nil && p.intervalClampWarned.CompareAndSwap(false, true) {
			p.Logger.Warn(p.LogPrefix+": scalingMonitorInterval is below the minimum; clamping to avoid pool thrashing",
				zap.Duration("configured", interval),
				zap.Duration("effective", defaultScalingMonitorInterval),
				zap.String("peer", p.ID))
		}
		return defaultScalingMonitorInterval
	default:
		p.intervalClampWarned.Store(false)
		return interval
	}
}

// EvaluateScaling runs one monitor pass: fill to MinConnections (when scaling
// is enabled), idle cleanup, then a scale-down check.
func (p *Pool[T]) EvaluateScaling() {
	cfg := p.Config()
	if cfg.DynamicScalingEnabled {
		p.ensureMinConnections()
	}
	p.CleanupIdleConns()
	p.MaybeScaleDown()
}

// StartMonitor starts the background scale-down/idle/min-fill loop if it is
// not already running. It is for pools whose first connection is dialed
// lazily after Start (Start(0, false)): starting the monitor only once that
// connection exists keeps the monitor's immediate min-fill pass from dialing
// concurrently with the caller's own first dial. It is a no-op once the pool
// is stopping or the monitor has already started.
func (p *Pool[T]) StartMonitor() { p.startScalingMonitor() }

// startScalingMonitor starts the background scale-down/idle/min-fill loop once.
func (p *Pool[T]) startScalingMonitor() {
	p.addingCount.Add(1)
	defer p.addingCount.Add(-1)
	if p.ctx.Err() != nil || p.shutdownStarted.Load() {
		return
	}
	if !p.monitorStarted.CompareAndSwap(false, true) {
		return
	}
	p.connWg.Add(1)
	go p.runScalingMonitor()
}

// minConnectionTarget is the fill floor for active connections, capped by
// MaxConnections. It is 0 when scaling is off so fill/scale-up do not grow
// the pool.
func minConnectionTarget(cfg Config) int {
	if !cfg.DynamicScalingEnabled || cfg.MinConnections < 1 {
		return 0
	}
	if cfg.MaxConnections > 0 && cfg.MinConnections > cfg.MaxConnections {
		return cfg.MaxConnections
	}
	return cfg.MinConnections
}

// ScaleDownFloor is how many active connections MaybeScaleDown will keep.
// Scaling on uses MinConnections capped by MaxConnections. Scaling off, or a
// zero MinConnections, keeps a single connection so extras still wind down.
// It is also the natural number of connections to dial at startup.
func ScaleDownFloor(cfg Config) int {
	if n := minConnectionTarget(cfg); n > 0 {
		return n
	}
	return 1
}

func (p *Pool[T]) activeConnCount() int {
	n := 0
	for _, c := range p.LoadConns() {
		if c.IsActive() {
			n++
		}
	}
	return n
}

// ensureMinConnections reactivates idle/draining connections, then dials,
// until the pool meets MinConnections.
func (p *Pool[T]) ensureMinConnections() {
	if p.activeConnCount() >= minConnectionTarget(p.Config()) {
		return
	}
	if !atomic.CompareAndSwapInt32(&p.isScaling, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&p.isScaling, 0)
	if p.shutdownStarted.Load() || p.ctx.Err() != nil {
		return
	}
	p.fillToMinConnections()
}

// fillToMinConnections grows active connections to the MinConnections floor.
// Idle and draining connections are reused before dialing. Caller must hold
// isScaling (or be the only scale-up worker).
func (p *Pool[T]) fillToMinConnections() {
	reactivated := 0
	dialed := 0
	for {
		if p.ctx.Err() != nil {
			return
		}
		cfg := p.Config()
		target := minConnectionTarget(cfg)
		if target < 1 || p.activeConnCount() >= target {
			break
		}
		if p.ReactivateIdleConn() {
			reactivated++
			p.metrics.IncIdleReactivation()
			continue
		}
		if cfg.MaxConnections > 0 && int(p.connCount.Load()) >= cfg.MaxConnections {
			break
		}
		if _, err := p.AddConn(); err != nil {
			p.Logger.Warn(p.LogPrefix+": failed to fill connection pool to minConnections",
				zap.String("peer", p.ID),
				zap.Int("minConnections", target),
				zap.Error(err))
			return
		}
		dialed++
		p.metrics.IncScaleUp()
	}
	if reactivated > 0 || dialed > 0 {
		cfg := p.Config()
		p.Logger.Info(p.LogPrefix+": scaling up connection pool; filling minConnections",
			zap.String("peer", p.ID),
			zap.Int("reactivated", reactivated),
			zap.Int("dialed", dialed),
			zap.Int("minConnections", minConnectionTarget(cfg)),
			zap.Int("active_connections", p.activeConnCount()),
			zap.Int32("total_connections", p.connCount.Load()))
		p.RefreshMetrics()
	}
}

// MaybeScaleDown checks whether the pool can be reduced by one connection.
// When scaling is on, a connection is marked draining only if the remaining
// active connections can absorb current load without crossing the hysteresis
// band, and never below ScaleDownFloor. When scaling is off, extras drain
// toward 1 regardless of load; in-flight streams finish on the draining
// connection, then the idle timeout closes it.
// Lock-free read path: loads an immutable snapshot via atomic.Pointer.Load().
// The state mutation uses TransitionState so that concurrent reactivation by
// TryScaleUp is mutually exclusive with the draining transition.
func (p *Pool[T]) MaybeScaleDown() {
	cfg := p.Config()

	conns := p.LoadConns()
	active := make([]*Wrapper[T], 0, len(conns))
	for _, c := range conns {
		if c.IsActive() {
			active = append(active, c)
		}
	}

	floor := ScaleDownFloor(cfg)
	if len(active) <= floor {
		return
	}

	var totalStreams int32
	for _, c := range active {
		totalStreams += c.StreamCount()
	}

	// When scaling is on, only drain if the remaining connections can absorb
	// current load without crossing the hysteresis band (prevents oscillation
	// near the scale-up boundary). When scaling is off, skip that guard so
	// extras still drain toward 1; in-flight streams finish on the draining
	// connection.
	scaleDownThreshold := int32(float64(cfg.MaxConcurrentStreams) * (cfg.ScaleUpThreshold - cfg.ScaleDownGap))
	if cfg.DynamicScalingEnabled {
		capacityAfterDrain := scaleDownThreshold * int32(len(active)-1)
		if totalStreams >= capacityAfterDrain {
			return
		}
	}

	// Scaling on: drain the most-loaded conn so survivors keep burst capacity.
	// Scaling off: drain the least-loaded extra so traffic stays on the busy one.
	var target *Wrapper[T]
	for _, c := range active {
		if target == nil {
			target = c
			continue
		}
		if cfg.DynamicScalingEnabled {
			if c.StreamCount() > target.StreamCount() {
				target = c
			}
		} else if c.StreamCount() < target.StreamCount() {
			target = c
		}
	}
	if target == nil {
		return
	}

	// Use CAS so that a concurrent reactivation by TryScaleUp cannot race
	// with this draining transition on the same connection.
	if !target.TransitionState(StateActive, StateDraining) {
		return
	}

	// Logged at Info so pool scaling can be debugged from logs: connection pool
	// metrics are aggregated across peers (not tagged by peer), so the per-pool
	// detail lives here instead. Logs can safely carry the pool ID without the
	// cardinality concerns that affect metrics.
	p.Logger.Info(p.LogPrefix+": scaling down connection pool; marked connection for draining",
		zap.String("peer", p.ID),
		zap.Int("active_connections_before", len(active)),
		zap.Int("active_connections_after", len(active)-1),
		zap.Int32("total_streams", totalStreams),
		zap.Int32("drained_conn_stream_count", target.StreamCount()),
		zap.Int32("scale_down_threshold", scaleDownThreshold))
	p.metrics.IncScaleDown()
	p.RefreshMetrics()
}

// CleanupIdleConns advances draining connections with zero streams to the idle
// state, and cancels connections that have exceeded the idle timeout so their
// watcher goroutines can close and remove them.
// It skips closing idle connections while a scale-up is in progress so that
// TryScaleUp can reactivate them instead of dialing a new connection.
// All state transitions use TransitionState so they are mutually exclusive
// with concurrent reactivation by TryScaleUp:
//
//	Time 1 CleanupIdleConns: reads isScaling==0, adds c to toClose
//	Time 2 TryScaleUp:       CAS isScaling 0->1, reactivates c (idle->active)
//	Time 3 CleanupIdleConns: TransitionState(idle->closing) fails -> skips cancel
func (p *Pool[T]) CleanupIdleConns() {
	// If a scale-up goroutine is running, hold off -- idle connections may be
	// reactivated by TryScaleUp instead of being closed.
	if atomic.LoadInt32(&p.isScaling) == 1 {
		return
	}

	now := time.Now()
	idleTimeout := p.Config().IdleTimeout

	conns := p.LoadConns()
	var drained []*Wrapper[T]
	var toClose []*Wrapper[T]
	for _, c := range conns {
		if c.GetState() == StateDraining && c.StreamCount() == 0 {
			drained = append(drained, c)
		} else if c.GetState() == StateIdle && !c.IdleSince().IsZero() &&
			now.Sub(c.IdleSince()) >= idleTimeout {
			toClose = append(toClose, c)
		}
	}

	// Use CAS for the draining->idle transition so a concurrent reactivation
	// by TryScaleUp (idle->active) cannot race on the same connection.
	for _, c := range drained {
		if c.TransitionState(StateDraining, StateIdle) {
			c.setIdleNow()
		}
	}

	for _, c := range toClose {
		// Claim the connection for closure via CAS before calling Cancel.
		// If TryScaleUp wins the CAS first (idle->active), we skip this
		// connection -- it has been reactivated and must not be cancelled.
		if !c.TransitionState(StateIdle, StateClosing) {
			continue
		}
		// Info: closing an idle connection shrinks the pool, so surface it in
		// logs for scaling debuggability (metrics are not tagged by peer).
		p.Logger.Info(p.LogPrefix+": scaling down connection pool; closing idle connection after timeout",
			zap.String("peer", p.ID),
			zap.Duration("idle_duration", now.Sub(c.IdleSince())),
			zap.Int32("total_connections", p.connCount.Load()))
		// Cancelling the wrapper context causes the per-connection watcher to
		// exit, which closes the underlying connection and removes the
		// wrapper from the pool.
		c.Cancel()
	}

	if len(drained) > 0 || len(toClose) > 0 {
		p.RefreshMetrics()
	}
}

// TryScaleUp triggers a background goroutine to satisfy the need for more
// connection capacity. It receives leastLoadedConn -- the connection with the
// fewest active streams, as selected by PickConn. If even the least-loaded
// connection is at or above the scale-up threshold then all connections are
// over budget and the pool needs to grow. It first tries to reactivate an
// existing idle connection (avoiding a new dial); only if none are available
// does it open a new connection, subject to the MaxConnections cap.
// isScaling is atomically set to 1 on entry (via CAS) and reset to 0 when
// the goroutine finishes -- this serves a dual purpose: it ensures at most one
// scale-up goroutine runs at a time, and it signals CleanupIdleConns to hold
// off closing idle connections while a reactivation may be in progress.
func (p *Pool[T]) TryScaleUp(leastLoadedConn *Wrapper[T]) {
	cfg := p.Config()
	if !cfg.DynamicScalingEnabled {
		return
	}

	threshold := int32(float64(cfg.MaxConcurrentStreams) * cfg.ScaleUpThreshold)
	needMin := p.activeConnCount() < minConnectionTarget(cfg)
	needLoad := leastLoadedConn != nil && leastLoadedConn.StreamCount() >= threshold
	if !needMin && !needLoad {
		return
	}

	// Set isScaling to 1 (from 0) to claim the scale-up slot.
	// If another goroutine already holds it, bail out.
	if !atomic.CompareAndSwapInt32(&p.isScaling, 0, 1) {
		return
	}
	p.addingCount.Add(1)
	defer p.addingCount.Add(-1)
	if p.shutdownStarted.Load() || p.ctx.Err() != nil {
		atomic.StoreInt32(&p.isScaling, 0)
		return
	}

	p.connWg.Add(1)
	go func() {
		defer func() {
			atomic.StoreInt32(&p.isScaling, 0)
			p.connWg.Done()
		}()

		if p.ctx.Err() != nil {
			return
		}

		p.fillToMinConnections()
		cfg := p.Config()
		if !cfg.DynamicScalingEnabled {
			return
		}
		// Min-fill may have reactivated or dialed; re-read the least-loaded
		// active conn so we do not dial again from a stale snapshot.
		leastLoadedConn := p.PickConn()
		if leastLoadedConn == nil {
			return
		}
		threshold := int32(float64(cfg.MaxConcurrentStreams) * cfg.ScaleUpThreshold)
		if leastLoadedConn.StreamCount() < threshold {
			return
		}

		// Prefer reactivating an idle connection over dialing a new one.
		if p.ReactivateIdleConn() {
			// Logged at Info so pool scaling is debuggable from logs (metrics
			// are aggregated across peers and not tagged by peer).
			p.Logger.Info(p.LogPrefix+": scaling up connection pool; reactivated idle connection",
				zap.String("peer", p.ID),
				zap.Int32("active_conn_stream_count", leastLoadedConn.StreamCount()),
				zap.Int32("scale_up_threshold", threshold),
				zap.Int32("total_connections", p.connCount.Load()))
			p.metrics.IncIdleReactivation()
			p.RefreshMetrics()
			return
		}

		// No idle connection available; dial a new one if below the cap.
		// connCount is maintained atomically -- no mutex needed for this check.
		if int(p.connCount.Load()) >= cfg.MaxConnections {
			if p.atMaxConnectionsLogged.CompareAndSwap(false, true) {
				p.Logger.Info(p.LogPrefix+": cannot scale up connection pool; at max connections",
					zap.String("peer", p.ID),
					zap.Int32("total_connections", p.connCount.Load()),
					zap.Int("max_connections", cfg.MaxConnections))
			}
			return
		}

		if _, err := p.AddConn(); err != nil {
			p.Logger.Warn(p.LogPrefix+": failed to scale up connection pool",
				zap.String("peer", p.ID),
				zap.Error(err))
		} else {
			// Logged at Info so pool scaling is debuggable from logs (metrics
			// are aggregated across peers and not tagged by peer).
			p.Logger.Info(p.LogPrefix+": scaling up connection pool; opened new connection",
				zap.String("peer", p.ID),
				zap.Int32("active_conn_stream_count", leastLoadedConn.StreamCount()),
				zap.Int32("scale_up_threshold", threshold),
				zap.Int32("total_connections", p.connCount.Load()))
			p.metrics.IncScaleUp()
		}
	}()
}

// ReactivateIdleConn finds a connection to bring back to active, avoiding an
// unnecessary dial. It prefers idle connections (zero in-flight streams) over
// draining ones (streams still in flight), and uses CAS so that a concurrent
// CleanupIdleConns cannot cancel a connection that is being reactivated.
// Only one ReactivateIdleConn call runs at a time (guarded by the isScaling
// CAS in TryScaleUp).
// Returns true if a connection was reactivated.
func (p *Pool[T]) ReactivateIdleConn() bool {
	conns := p.LoadConns()

	// First pass: prefer idle connections (no in-flight streams, cheapest to
	// reactivate).
	for _, c := range conns {
		// Only reactivate if the connection context is still live; a cancelled
		// context means CleanupIdleConns has already scheduled it for closure.
		if c.GetState() == StateIdle && c.ctx.Err() == nil {
			if c.TransitionState(StateIdle, StateActive) {
				atomic.StoreInt64(&c.lastIdleAtNano, 0)
				return true
			}
		}
	}

	// Second pass: reactivate a draining connection if no idle one is available.
	// A draining connection still has in-flight streams but can accept new ones
	// once reactivated, preventing accumulation of stuck draining connections
	// under sustained load.
	for _, c := range conns {
		if c.GetState() == StateDraining && c.ctx != nil && c.ctx.Err() == nil {
			if c.TransitionState(StateDraining, StateActive) {
				return true
			}
		}
	}

	return false
}

// RefreshMetrics scans the pool and updates the active, draining, and idle
// connection gauges.
func (p *Pool[T]) RefreshMetrics() {
	conns := p.LoadConns()
	var active, draining, idle int64
	for _, c := range conns {
		switch c.GetState() {
		case StateActive:
			active++
		case StateDraining:
			draining++
		case StateIdle:
			idle++
		}
	}
	p.metrics.SetCounts(active, draining, idle)
	if int(p.connCount.Load()) < p.Config().MaxConnections {
		p.atMaxConnectionsLogged.Store(false)
	}
}
