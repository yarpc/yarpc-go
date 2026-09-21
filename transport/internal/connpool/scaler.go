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

// RunScalingMonitor runs as a background goroutine for the lifetime of the
// pool. It periodically evaluates whether connections should be added or
// removed. It exits when the pool's context is cancelled (via Stop).
//
// The caller of Start has already accounted for this goroutine in connWg;
// RunScalingMonitor calls connWg.Done on exit.
func (p *Pool[T]) RunScalingMonitor() {
	defer p.connWg.Done()

	interval := p.Config.ScalingMonitorInterval
	switch {
	case interval <= 0:
		interval = defaultScalingMonitorInterval
	case interval < defaultScalingMonitorInterval:
		if p.Logger != nil {
			p.Logger.Warn("connpool: scalingMonitorInterval is below the minimum; clamping to avoid pool thrashing",
				zap.Duration("configured", interval),
				zap.Duration("effective", defaultScalingMonitorInterval),
				zap.String("pool", p.ID))
		}
		interval = defaultScalingMonitorInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.EvaluateScaling()
		case <-p.ctx.Done():
			return
		}
	}
}

// EvaluateScaling runs one round of idle cleanup followed by a scale-down
// check. It is exported so callers with a bespoke cadence can drive it
// directly instead of relying on RunScalingMonitor's ticker.
func (p *Pool[T]) EvaluateScaling() {
	p.CleanupIdleConns()
	p.MaybeScaleDown()
}

// MaybeScaleDown checks whether the pool can be reduced by one connection. A
// connection is marked for draining when the remaining active connections
// can absorb the current aggregate load without immediately triggering
// another scale-up. The scale-down threshold applies a hysteresis gap below
// ScaleUpThreshold to prevent oscillation when load hovers near the scale-up
// boundary.
func (p *Pool[T]) MaybeScaleDown() {
	conns := p.LoadConns()
	active := make([]*Wrapper[T], 0, len(conns))
	for _, c := range conns {
		if c.IsActive() {
			active = append(active, c)
		}
	}

	// Never drain below MinConnections.
	if len(active) <= p.Config.MinConnections {
		return
	}

	// scaleDownThreshold introduces a hysteresis band below ScaleUpThreshold.
	// Scale down only when load would fit within the lower threshold on the
	// reduced pool, preventing oscillation near the scale-up boundary.
	scaleDownThreshold := int32(float64(p.Config.MaxConcurrentStreams) * (p.Config.ScaleUpThreshold - p.Config.ScaleDownGap))

	var totalLoad int32
	for _, c := range active {
		totalLoad += c.StreamCount()
	}

	// Only drain if the remaining (n-1) connections can absorb current load
	// without crossing the scale-down threshold.
	capacityAfterDrain := scaleDownThreshold * int32(len(active)-1)
	if totalLoad >= capacityAfterDrain {
		return
	}

	// Drain the most-loaded active connection: this maximises residual
	// capacity in the surviving connections, improving burst absorption.
	var mostLoaded *Wrapper[T]
	for _, c := range active {
		if mostLoaded == nil || c.StreamCount() > mostLoaded.StreamCount() {
			mostLoaded = c
		}
	}
	if mostLoaded == nil {
		return
	}

	// Use CAS so a concurrent reactivation by TryScaleUp cannot race with
	// this draining transition on the same connection.
	if !mostLoaded.TransitionState(StateActive, StateDraining) {
		return
	}

	if p.Logger != nil {
		p.Logger.Info("connpool: scaling down pool; marked connection for draining",
			zap.String("pool", p.ID),
			zap.Int("active_connections_before", len(active)),
			zap.Int("active_connections_after", len(active)-1),
			zap.Int32("total_load", totalLoad),
			zap.Int32("drained_conn_load", mostLoaded.StreamCount()),
			zap.Int32("scale_down_threshold", scaleDownThreshold))
	}
	p.metrics.IncScaleDown()
	p.RefreshMetrics()
}

// CleanupIdleConns advances draining connections with zero load to the idle
// state, and removes connections that have exceeded IdleTimeout. It skips
// closing idle connections while a scale-up is in progress so TryScaleUp can
// reactivate them instead of dialing a new connection.
//
// All state transitions use TransitionState so they are mutually exclusive
// with concurrent reactivation by TryScaleUp:
//
//	Time 1 CleanupIdleConns: reads isScaling==0, adds c to toClose
//	Time 2 TryScaleUp:       CAS isScaling 0->1, reactivates c (idle->active)
//	Time 3 CleanupIdleConns: TransitionState(idle->closing) fails -> skipped
func (p *Pool[T]) CleanupIdleConns() {
	// If a scale-up goroutine is running, hold off -- idle connections may
	// be reactivated by TryScaleUp instead of being closed.
	if atomic.LoadInt32(&p.isScaling) == 1 {
		return
	}

	now := time.Now()

	conns := p.LoadConns()
	var drained []*Wrapper[T]
	var toClose []*Wrapper[T]
	for _, c := range conns {
		if c.GetState() == StateDraining && c.StreamCount() == 0 {
			drained = append(drained, c)
		} else if c.GetState() == StateIdle && !c.IdleSince().IsZero() &&
			now.Sub(c.IdleSince()) >= p.Config.IdleTimeout {
			toClose = append(toClose, c)
		}
	}

	for _, c := range drained {
		if c.TransitionState(StateDraining, StateIdle) {
			c.setIdleNow()
		}
	}

	for _, c := range toClose {
		// Claim the connection for closure via CAS. If TryScaleUp wins the
		// CAS first (idle->active), skip it -- it has been reactivated and
		// must not be closed. Cancelling here only starts teardown; the
		// actual close/remove/metrics-refresh/StoppedC sequencing happens
		// asynchronously in the per-connection watcher spawned from
		// OnConnAdded, once it observes the cancelled context -- this pool
		// never removes a connection directly.
		if !c.TransitionState(StateIdle, StateClosing) {
			continue
		}
		if p.Logger != nil {
			p.Logger.Info("connpool: scaling down pool; closing idle connection after timeout",
				zap.String("pool", p.ID),
				zap.Duration("idle_duration", now.Sub(c.IdleSince())),
				zap.Int32("total_connections", p.connCount.Load()))
		}
		c.Cancel()
	}

	if len(drained) > 0 || len(toClose) > 0 {
		p.RefreshMetrics()
	}
}

// TryScaleUp triggers a background goroutine to satisfy the need for more
// connection capacity. It receives leastLoadedConn -- the connection with
// the fewest in-flight streams/requests, as selected by PickConn. If even
// the least-loaded connection is at or above the scale-up threshold then all
// connections are over budget and the pool needs to grow. It first tries to
// reactivate an existing idle connection (avoiding a new dial); only if none
// are available does it open a new connection, subject to MaxConnections.
func (p *Pool[T]) TryScaleUp(leastLoadedConn *Wrapper[T]) {
	if !p.Config.DynamicScalingEnabled {
		return
	}

	threshold := int32(float64(p.Config.MaxConcurrentStreams) * p.Config.ScaleUpThreshold)
	if leastLoadedConn.StreamCount() < threshold {
		return
	}

	// Set isScaling to 1 (from 0) to claim the scale-up slot. If another
	// goroutine already holds it, bail out.
	if !atomic.CompareAndSwapInt32(&p.isScaling, 0, 1) {
		return
	}

	// Unlike AddConn's addingCount dance, this check is not raced against
	// the teardown goroutine's Gosched spin: it checks shutdownStarted/
	// ctx.Err() directly, accepting the (harmless) possibility that a
	// scale-up goroutine's own AddConn call loses the race and simply gets
	// ctx.Canceled back.
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

		// Prefer reactivating an idle connection over dialing a new one.
		if p.ReactivateIdleConn() {
			if p.Logger != nil {
				p.Logger.Info("connpool: scaling up pool; reactivated idle connection",
					zap.String("pool", p.ID),
					zap.Int32("least_loaded_conn_load", leastLoadedConn.StreamCount()),
					zap.Int32("scale_up_threshold", threshold),
					zap.Int32("total_connections", p.connCount.Load()))
			}
			p.metrics.IncIdleReactivation()
			p.RefreshMetrics()
			return
		}

		// No idle connection available; dial a new one if below the cap.
		if int(p.connCount.Load()) >= p.Config.MaxConnections {
			if p.atMaxConnectionsLogged.CompareAndSwap(false, true) && p.Logger != nil {
				p.Logger.Info("connpool: cannot scale up pool; at max connections",
					zap.String("pool", p.ID),
					zap.Int32("total_connections", p.connCount.Load()),
					zap.Int("max_connections", p.Config.MaxConnections))
			}
			return
		}

		if _, err := p.AddConn(); err != nil {
			if p.Logger != nil {
				p.Logger.Warn("connpool: failed to scale up pool",
					zap.String("pool", p.ID),
					zap.Error(err))
			}
		} else {
			if p.Logger != nil {
				p.Logger.Info("connpool: scaling up pool; opened new connection",
					zap.String("pool", p.ID),
					zap.Int32("least_loaded_conn_load", leastLoadedConn.StreamCount()),
					zap.Int32("scale_up_threshold", threshold),
					zap.Int32("total_connections", p.connCount.Load()))
			}
			// AddConn already refreshed the gauges for this connection;
			// only the counter needs bumping here.
			p.metrics.IncScaleUp()
		}
	}()
}

// ReactivateIdleConn finds a connection to bring back to active, avoiding an
// unnecessary dial. It prefers idle connections (zero in-flight load) over
// draining ones (load still in flight), and uses CAS so a concurrent
// CleanupIdleConns cannot close a connection that is being reactivated.
// Returns true if a connection was reactivated.
func (p *Pool[T]) ReactivateIdleConn() bool {
	conns := p.LoadConns()

	// First pass: prefer idle connections (no in-flight load, cheapest to
	// reactivate).
	for _, c := range conns {
		// Only reactivate if the connection's context is still live; a
		// cancelled context means CleanupIdleConns has already claimed it.
		if c.GetState() == StateIdle && c.ctx.Err() == nil {
			if c.TransitionState(StateIdle, StateActive) {
				atomic.StoreInt64(&c.lastIdleAtNano, 0)
				return true
			}
		}
	}

	// Second pass: reactivate a draining connection if no idle one is
	// available. It still has in-flight load but can accept new
	// streams/requests once reactivated, preventing accumulation of stuck
	// draining connections under sustained load.
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
	if int(p.connCount.Load()) < p.Config.MaxConnections {
		p.atMaxConnectionsLogged.Store(false)
	}
}
