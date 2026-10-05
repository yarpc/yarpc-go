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
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// _defaultScalingMonitorInterval is both the default and the minimum allowed
// interval for the scaling monitor. Values below this would cause excessive
// pool churn and are rejected at config validation time.
const _defaultScalingMonitorInterval = 30 * time.Second

// runScalingMonitor runs as a background goroutine for the lifetime of the
// peer.  It periodically evaluates whether connections should be removed
// from the pool.  It exits when the peer's context is cancelled.
func (p *grpcPeer) runScalingMonitor() {
	defer p.connWg.Done()

	// Apply live minConnections / idle cleanup without waiting for the first tick.
	if p.t != nil {
		p.evaluateScaling()
	}

	timer := time.NewTimer(p.scalingMonitorInterval())
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			p.evaluateScaling()
			timer.Reset(p.scalingMonitorInterval())
		case <-p.ctx.Done():
			return
		}
	}
}

// scalingMonitorInterval returns the wait until the next monitor pass.
// Live ScalingMonitorInterval overlays are picked up here; values under 30s
// are still clamped so a live flap cannot thrash the pool.
func (p *grpcPeer) scalingMonitorInterval() time.Duration {
	interval := p.livePoolCfg().scalingMonitorInterval
	switch {
	case interval <= 0:
		return _defaultScalingMonitorInterval
	case interval < _defaultScalingMonitorInterval:
		if p.t != nil && p.t.options != nil && p.intervalClampWarned.CompareAndSwap(false, true) {
			p.t.options.logger.Warn("grpc: scalingMonitorInterval is below the minimum; clamping to avoid pool thrashing",
				zap.Duration("configured", interval),
				zap.Duration("effective", _defaultScalingMonitorInterval),
				zap.String("peer", p.HostPort()))
		}
		return _defaultScalingMonitorInterval
	default:
		p.intervalClampWarned.Store(false)
		return interval
	}
}

func (p *grpcPeer) evaluateScaling() {
	cfg := p.livePoolCfg()
	if cfg.dynamicScalingEnabled {
		p.ensureMinConnections()
	}
	p.cleanupIdleConns()
	p.maybeScaleDown()
}

// startScalingMonitor starts the background scale-down/idle/min-fill loop once.
func (p *grpcPeer) startScalingMonitor() {
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

// minConnectionTarget is the live fill floor for active connections, capped by
// maxConnections. It is 0 when scaling is off so fill/scale-up do not grow
// the pool.
func minConnectionTarget(cfg connPoolConfig) int {
	if !cfg.dynamicScalingEnabled || cfg.minConnections < 1 {
		return 0
	}
	if cfg.maxConnections > 0 && cfg.minConnections > cfg.maxConnections {
		return cfg.maxConnections
	}
	return cfg.minConnections
}

// scaleDownFloor is how many active connections maybeScaleDown will keep.
// Scaling on uses minConnectionTarget (min capped by max). Scaling off, or a
// zero min, keeps a single connection so extras still wind down.
func scaleDownFloor(cfg connPoolConfig) int {
	if n := minConnectionTarget(cfg); n > 0 {
		return n
	}
	return 1
}

func (p *grpcPeer) activeConnCount() int {
	n := 0
	for _, c := range p.loadConns() {
		if c.isActive() {
			n++
		}
	}
	return n
}

// ensureMinConnections reactivates idle/draining connections, then dials,
// until the pool meets live minConnections.
func (p *grpcPeer) ensureMinConnections() {
	if p.activeConnCount() >= minConnectionTarget(p.livePoolCfg()) {
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

// fillToMinConnections grows active connections to the live minConnections floor.
// Idle and draining connections are reused before dialing. Caller must hold
// isScaling (or be the only scale-up worker).
func (p *grpcPeer) fillToMinConnections() {
	reactivated := 0
	dialed := 0
	for {
		if p.ctx.Err() != nil {
			return
		}
		cfg := p.livePoolCfg()
		target := minConnectionTarget(cfg)
		if target < 1 || p.activeConnCount() >= target {
			break
		}
		if p.reactivateIdleConn() {
			reactivated++
			p.metrics.incIdleReactivation()
			continue
		}
		if cfg.maxConnections > 0 && int(p.connCount.Load()) >= cfg.maxConnections {
			break
		}
		if err := p.addConn(); err != nil {
			p.t.options.logger.Warn("grpc: failed to fill connection pool to minConnections",
				zap.String("peer", p.HostPort()),
				zap.Int("minConnections", target),
				zap.Error(err))
			return
		}
		dialed++
		p.metrics.incScaleUp()
	}
	if reactivated > 0 || dialed > 0 {
		cfg := p.livePoolCfg()
		p.t.options.logger.Info("grpc: scaling up connection pool; filling minConnections",
			zap.String("peer", p.HostPort()),
			zap.Int("reactivated", reactivated),
			zap.Int("dialed", dialed),
			zap.Int("minConnections", minConnectionTarget(cfg)),
			zap.Int("active_connections", p.activeConnCount()),
			zap.Int32("total_connections", p.connCount.Load()))
		p.refreshPoolMetrics()
	}
}

// maybeScaleDown checks whether the pool can be reduced by one connection.
// When scaling is on, a connection is marked draining only if the remaining
// active connections can absorb current load without crossing the hysteresis
// band, and never below scaleDownFloor (min capped by max). When scaling is
// off, extras drain toward 1 regardless of load; in-flight RPCs finish on the
// draining connection, then idle timeout closes it.
// Lock-free read path: loads an immutable snapshot via atomic.Pointer.Load().
// The state mutation uses transitionState so that concurrent reactivation by
// tryScaleUp is mutually exclusive with the draining transition.
func (p *grpcPeer) maybeScaleDown() {
	cfg := p.livePoolCfg()

	conns := p.loadConns()
	active := make([]*grpcClientConnWrapper, 0, len(conns))
	for _, c := range conns {
		if c.isActive() {
			active = append(active, c)
		}
	}

	floor := scaleDownFloor(cfg)
	if len(active) <= floor {
		return
	}

	var totalStreams int32
	for _, c := range active {
		totalStreams += c.getStreamCount()
	}

	// When scaling is on, only drain if the remaining connections can absorb
	// current load without crossing the hysteresis band (prevents oscillation
	// near the scale-up boundary). When scaling is off, skip that guard so
	// extras still drain toward 1; in-flight RPCs finish on the draining conn.
	scaleDownThreshold := int32(float64(cfg.maxConcurrentStreams) * (cfg.scaleUpThreshold - cfg.scaleDownGap))
	if cfg.dynamicScalingEnabled {
		capacityAfterDrain := scaleDownThreshold * int32(len(active)-1)
		if totalStreams >= capacityAfterDrain {
			return
		}
	}

	// Scaling on: drain the most-loaded conn so survivors keep burst capacity.
	// Scaling off: drain the least-loaded extra so traffic stays on the busy one.
	var target *grpcClientConnWrapper
	for _, c := range active {
		if target == nil {
			target = c
			continue
		}
		if cfg.dynamicScalingEnabled {
			if c.getStreamCount() > target.getStreamCount() {
				target = c
			}
		} else if c.getStreamCount() < target.getStreamCount() {
			target = c
		}
	}
	if target == nil {
		return
	}

	// Use CAS so that a concurrent reactivation by tryScaleUp cannot race
	// with this draining transition on the same connection.
	if !target.transitionState(connStateActive, connStateDraining) {
		return
	}

	// Logged at Info so pool scaling can be debugged from logs: connection pool
	// metrics are aggregated across peers (not tagged by peer), so the per-peer
	// detail lives here instead. Logs can safely carry the peer without the
	// cardinality concerns that affect metrics.
	p.t.options.logger.Info("grpc: scaling down connection pool; marked connection for draining",
		zap.String("peer", p.HostPort()),
		zap.Int("active_connections_before", len(active)),
		zap.Int("active_connections_after", len(active)-1),
		zap.Int32("total_streams", totalStreams),
		zap.Int32("drained_conn_stream_count", target.getStreamCount()),
		zap.Int32("scale_down_threshold", scaleDownThreshold))
	p.metrics.incScaleDown()
	p.refreshPoolMetrics()
}

// cleanupIdleConns advances draining connections with zero streams to the idle
// state, and cancels connections that have exceeded the idle timeout so their
// monitor goroutines can close and remove them.
// It skips closing idle connections while a scale-up is in progress so that
// tryScaleUp can reactivate them instead of dialing a new connection.
// Lock-free read path: uses atomic.Pointer.Load() for the snapshot.
// All state transitions use transitionState so they are mutually exclusive with
// concurrent reactivation by tryScaleUp, closing the race identified in review:
//
//	Time 1 cleanupIdleConns: reads isScaling==0, adds c to toClose
//	Time 2 tryScaleUp:       CAS isScaling 0→1, reactivates c (idle→active)
//	Time 3 cleanupIdleConns: transitionState(idle→closing) fails → skips cancel
func (p *grpcPeer) cleanupIdleConns() {
	// If a scale-up goroutine is running, hold off — idle connections may be
	// reactivated by tryScaleUp instead of being closed.
	if atomic.LoadInt32(&p.isScaling) == 1 {
		return
	}

	now := time.Now()
	idleTimeout := p.livePoolCfg().idleTimeout

	conns := p.loadConns()
	var drained []*grpcClientConnWrapper
	var toClose []*grpcClientConnWrapper
	for _, c := range conns {
		if c.getState() == connStateDraining && c.getStreamCount() == 0 {
			drained = append(drained, c)
		} else if c.getState() == connStateIdle && !c.idleSince().IsZero() &&
			now.Sub(c.idleSince()) >= idleTimeout {
			toClose = append(toClose, c)
		}
	}

	// Use CAS for the draining→idle transition so a concurrent reactivation
	// by tryScaleUp (idle→active) cannot race on the same connection.
	for _, c := range drained {
		if c.transitionState(connStateDraining, connStateIdle) {
			c.setIdleNow()
		}
	}

	for _, c := range toClose {
		// Claim the connection for closure via CAS before calling cancel.
		// If tryScaleUp wins the CAS first (idle→active), we skip this
		// connection — it has been reactivated and must not be cancelled.
		if !c.transitionState(connStateIdle, connStateClosing) {
			continue
		}
		// Info: closing an idle connection shrinks the pool, so surface it in
		// logs for scaling debuggability (metrics are not tagged by peer).
		p.t.options.logger.Info("grpc: scaling down connection pool; closing idle connection after timeout",
			zap.String("peer", p.HostPort()),
			zap.Duration("idle_duration", now.Sub(c.idleSince())),
			zap.Int32("total_connections", p.connCount.Load()))
		// Cancelling the wrapper context causes monitorConnWrapper to
		// exit, which closes the underlying clientConn and removes the
		// wrapper from the pool via removeConn.
		c.cancel()
	}

	if len(drained) > 0 || len(toClose) > 0 {
		p.refreshPoolMetrics()
	}
}

// tryScaleUp triggers a background goroutine to satisfy the need for more
// connection capacity.  It receives leastLoadedConn — the connection with the
// fewest active streams, as selected by pickConn.  If even the least-loaded
// connection is at or above the scale-up threshold then all connections are
// over budget and the pool needs to grow.  It first tries to reactivate an
// existing idle connection (avoiding a new dial); only if none are available
// does it open a new connection, subject to the maxConnections cap.
// p.isScaling is atomically set to 1 on entry (via CAS) and reset to 0 when
// the goroutine finishes — this serves a dual purpose: it ensures at most one
// scale-up goroutine runs at a time, and it signals cleanupIdleConns to hold
// off closing idle connections while a reactivation may be in progress.
func (p *grpcPeer) tryScaleUp(leastLoadedConn *grpcClientConnWrapper) {
	cfg := p.livePoolCfg()
	if !cfg.dynamicScalingEnabled {
		return
	}

	threshold := int32(float64(cfg.maxConcurrentStreams) * cfg.scaleUpThreshold)
	needMin := p.activeConnCount() < minConnectionTarget(cfg)
	needLoad := leastLoadedConn != nil && leastLoadedConn.getStreamCount() >= threshold
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
		cfg := p.livePoolCfg()
		if !cfg.dynamicScalingEnabled {
			return
		}
		// Min-fill may have reactivated or dialed; re-read the least-loaded
		// active conn so we do not dial again from a stale snapshot.
		leastLoadedConn := p.pickConn()
		if leastLoadedConn == nil {
			return
		}
		threshold := int32(float64(cfg.maxConcurrentStreams) * cfg.scaleUpThreshold)
		if leastLoadedConn.getStreamCount() < threshold {
			return
		}

		// Prefer reactivating an idle connection over dialing a new one.
		if p.reactivateIdleConn() {
			// Logged at Info so pool scaling is debuggable from logs (metrics
			// are aggregated across peers and not tagged by peer).
			p.t.options.logger.Info("grpc: scaling up connection pool; reactivated idle connection",
				zap.String("peer", p.HostPort()),
				zap.Int32("active_conn_stream_count", leastLoadedConn.getStreamCount()),
				zap.Int32("scale_up_threshold", threshold),
				zap.Int32("total_connections", p.connCount.Load()))
			p.metrics.incIdleReactivation()
			p.refreshPoolMetrics()
			return
		}

		// No idle connection available; dial a new one if below the cap.
		// connCount is maintained atomically — no mutex needed for this check.
		if int(p.connCount.Load()) >= cfg.maxConnections {
			if p.atMaxConnectionsLogged.CompareAndSwap(false, true) {
				p.t.options.logger.Info("grpc: cannot scale up connection pool; at max connections",
					zap.String("peer", p.HostPort()),
					zap.Int32("total_connections", p.connCount.Load()),
					zap.Int("max_connections", cfg.maxConnections))
			}
			return
		}

		if err := p.addConn(); err != nil {
			p.t.options.logger.Warn("grpc: failed to scale up connection pool",
				zap.String("peer", p.HostPort()),
				zap.Error(err))
		} else {
			// Logged at Info so pool scaling is debuggable from logs (metrics
			// are aggregated across peers and not tagged by peer).
			p.t.options.logger.Info("grpc: scaling up connection pool; opened new connection",
				zap.String("peer", p.HostPort()),
				zap.Int32("active_conn_stream_count", leastLoadedConn.getStreamCount()),
				zap.Int32("scale_up_threshold", threshold),
				zap.Int32("total_connections", p.connCount.Load()))
			p.metrics.incScaleUp()
		}
	}()
}

// reactivateIdleConn finds a connection to bring back to active, avoiding an
// unnecessary dial.  It prefers idle connections (zero in-flight streams) over
// draining ones (streams still in flight), and uses CAS so that a concurrent
// cleanupIdleConns cannot cancel a connection that is being reactivated.
// Only one reactivateIdleConn call runs at a time (guarded by the isScaling
// CAS in tryScaleUp).
// Returns true if a connection was reactivated.
func (p *grpcPeer) reactivateIdleConn() bool {
	conns := p.loadConns()

	// First pass: prefer idle connections (no in-flight streams, cheapest to reactivate).
	for _, c := range conns {
		// Only reactivate if the connection context is still live; a cancelled
		// context means cleanupIdleConns has already scheduled it for closure.
		if c.getState() == connStateIdle && c.ctx.Err() == nil {
			if c.transitionState(connStateIdle, connStateActive) {
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
		if c.getState() == connStateDraining && c.ctx != nil && c.ctx.Err() == nil {
			if c.transitionState(connStateDraining, connStateActive) {
				return true
			}
		}
	}

	return false
}

// refreshPoolMetrics scans the pool and updates the active, draining, and idle
// connection gauges.
func (p *grpcPeer) refreshPoolMetrics() {
	conns := p.loadConns()
	var active, draining, idle int64
	for _, c := range conns {
		switch c.getState() {
		case connStateActive:
			active++
		case connStateDraining:
			draining++
		case connStateIdle:
			idle++
		}
	}
	p.metrics.setCounts(active, draining, idle)
	if int(p.connCount.Load()) < p.livePoolCfg().maxConnections {
		p.atMaxConnectionsLogged.Store(false)
	}
}
