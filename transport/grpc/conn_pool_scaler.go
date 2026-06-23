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
	interval := p.poolCfg.scalingMonitorInterval
	switch {
	case interval <= 0:
		interval = _defaultScalingMonitorInterval
	case interval < _defaultScalingMonitorInterval:
		p.t.options.logger.Warn("grpc: scalingMonitorInterval is below the minimum; clamping to avoid pool thrashing",
			zap.Duration("configured", interval),
			zap.Duration("effective", _defaultScalingMonitorInterval),
			zap.String("peer", p.HostPort()))
		interval = _defaultScalingMonitorInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.evaluateScaling()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *grpcPeer) evaluateScaling() {
	p.cleanupIdleConns()
	p.maybeScaleDown()
}

// maybeScaleDown checks whether the pool can be reduced by one connection.
// A connection is marked for draining when the remaining active connections
// can absorb the current aggregate stream load without triggering another
// scale-up.  The scale-down threshold applies a hysteresis gap below
// scaleUpThreshold to prevent oscillation when stream count hovers near
// the scale-up boundary.
// Lock-free read path: loads an immutable snapshot via atomic.Pointer.Load().
// The state mutation uses transitionState so that concurrent reactivation by
// tryScaleUp is mutually exclusive with the draining transition.
func (p *grpcPeer) maybeScaleDown() {
	conns := p.loadConns()
	active := make([]*grpcClientConnWrapper, 0, len(conns))
	for _, c := range conns {
		if c.isActive() {
			active = append(active, c)
		}
	}

	// Never drain below minConnections.
	if len(active) <= p.poolCfg.minConnections {
		return
	}

	// scaleDownThreshold introduces a hysteresis band below scaleUpThreshold.
	// Scale down only when load would fit within the lower threshold on the
	// reduced pool, preventing oscillation near the scale-up boundary.
	scaleDownThreshold := int32(float64(p.poolCfg.maxConcurrentStreams) * (p.poolCfg.scaleUpThreshold - p.poolCfg.scaleDownGap))

	var totalStreams int32
	for _, c := range active {
		totalStreams += c.getStreamCount()
	}

	// Only drain if the remaining (n-1) connections can absorb current load
	// without crossing the scale-down threshold.
	capacityAfterDrain := scaleDownThreshold * int32(len(active)-1)
	if totalStreams >= capacityAfterDrain {
		return
	}

	// Drain the most-loaded active connection: this maximises residual
	// stream capacity in the surviving connections, improving burst absorption.
	var mostLoaded *grpcClientConnWrapper
	for _, c := range active {
		if mostLoaded == nil || c.getStreamCount() > mostLoaded.getStreamCount() {
			mostLoaded = c
		}
	}

	if mostLoaded == nil {
		return
	}

	// Use CAS so that a concurrent reactivation by tryScaleUp cannot race
	// with this draining transition on the same connection.
	if !mostLoaded.transitionState(connStateActive, connStateDraining) {
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
		zap.Int32("drained_conn_stream_count", mostLoaded.getStreamCount()),
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

	conns := p.loadConns()
	var drained []*grpcClientConnWrapper
	var toClose []*grpcClientConnWrapper
	for _, c := range conns {
		if c.getState() == connStateDraining && c.getStreamCount() == 0 {
			drained = append(drained, c)
		} else if c.getState() == connStateIdle && !c.idleSince().IsZero() &&
			now.Sub(c.idleSince()) >= p.poolCfg.idleTimeout {
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
	if !p.poolCfg.dynamicScalingEnabled {
		return
	}

	threshold := int32(float64(p.poolCfg.maxConcurrentStreams) * p.poolCfg.scaleUpThreshold)
	if leastLoadedConn.getStreamCount() < threshold {
		return
	}

	// Set isScaling to 1 (from 0) to claim the scale-up slot.
	// If another goroutine already holds it, bail out.
	if !atomic.CompareAndSwapInt32(&p.isScaling, 0, 1) {
		return
	}

	go func() {
		defer atomic.StoreInt32(&p.isScaling, 0)

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
		if int(p.connCount.Load()) >= p.poolCfg.maxConnections {
			p.t.options.logger.Info("grpc: cannot scale up connection pool; at max connections",
				zap.String("peer", p.HostPort()),
				zap.Int32("total_connections", p.connCount.Load()),
				zap.Int("max_connections", p.poolCfg.maxConnections))
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
}
