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

package http

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

var errNoConnsAvailable = yarpcerrors.UnavailableErrorf("http2 pool: no connections available")

// http2PoolConfig controls how an http2Pool scales the number of HTTP/2
// connections it maintains to a single peer.
type http2PoolConfig struct {
	// dynamicScalingEnabled gates all automatic scaling. When false the pool
	// holds a single connection and never scales up or down, and the
	// monitor loop is not started.
	dynamicScalingEnabled  bool
	minConns               int
	maxConns               int
	scaleUpThreshold       float64
	scaleDownGap           float64
	idleTimeout            time.Duration
	scalingMonitorInterval time.Duration

	// maxConcurrentStreams is the assumed HTTP/2 SETTINGS_MAX_CONCURRENT_STREAMS
	// ceiling per pooled connection. A pooled *http2.Transport doesn't expose
	// the peer's real negotiated value (unlike *http2.ClientConn.State()), so
	// scaling decisions are made against this fixed assumption instead.
	maxConcurrentStreams int32
}

// validate rejects configurations the pool cannot run with, mirroring the gRPC
// pool's validateResolvedConnPool, plus the monitor interval that
// time.NewTicker would otherwise panic on.
func (c http2PoolConfig) validate() error {
	if c.minConns < 0 {
		return fmt.Errorf("http2 pool: minConns must be non-negative, got %d", c.minConns)
	}
	if c.maxConns < c.minConns {
		return fmt.Errorf("http2 pool: maxConns (%d) must be >= minConns (%d)", c.maxConns, c.minConns)
	}
	if c.maxConns < 1 {
		return fmt.Errorf("http2 pool: maxConns must be at least 1, got %d", c.maxConns)
	}
	if c.maxConcurrentStreams < 1 {
		return fmt.Errorf("http2 pool: maxConcurrentStreams must be at least 1, got %d", c.maxConcurrentStreams)
	}
	if c.scaleUpThreshold <= 0 || c.scaleUpThreshold > 1 {
		return fmt.Errorf("http2 pool: scaleUpThreshold must be in (0, 1], got %v", c.scaleUpThreshold)
	}
	if c.scaleUpThreshold-c.scaleDownGap <= 0 {
		return fmt.Errorf("http2 pool: scaleUpThreshold (%.2f) minus scaleDownGap (%.2f) must be > 0",
			c.scaleUpThreshold, c.scaleDownGap)
	}
	if c.idleTimeout < 0 {
		return fmt.Errorf("http2 pool: idleTimeout must be non-negative, got %v", c.idleTimeout)
	}
	if c.scalingMonitorInterval <= 0 {
		return fmt.Errorf("http2 pool: scalingMonitorInterval must be positive, got %v", c.scalingMonitorInterval)
	}
	return nil
}

func defaultHTTP2PoolConfig() http2PoolConfig {
	return http2PoolConfig{
		dynamicScalingEnabled:  defaultHTTP2PoolDynamicScalingEnabled,
		minConns:               defaultHTTP2PoolMinConns,
		maxConns:               defaultHTTP2PoolMaxConns,
		scaleUpThreshold:       defaultHTTP2PoolScaleUpThreshold,
		scaleDownGap:           defaultHTTP2PoolScaleDownGap,
		idleTimeout:            defaultHTTP2PoolConnIdleTimeout,
		scalingMonitorInterval: defaultHTTP2PoolScalingMonitorInterval,
		maxConcurrentStreams:   defaultHTTP2PoolMaxConcurrentStreams,
	}
}

// http2Pool maintains a set of *http2.Transport instances dedicated to a
// single peer address, scaling the number of connections up when observed
// load approaches the assumed per-connection concurrency limit, and back
// down when load drops. Each pooled http2Conn owns a private
// *http2.Transport (built via newTransport) rather than sharing one across
// the whole pool: since a *http2.Transport already dials and pools its own
// connection(s) to whatever authority a request targets, dedicating one
// Transport per pool slot -- and always routing requests for this peer's
// address through it -- makes each slot behave like a single physical
// connection, while letting the Transport's own client-conn pool transparently
// redial that connection after GOAWAY, with no reconnect logic of our own.
type http2Pool struct {
	addr         string
	newTransport func() *http2.Transport
	cfg          http2PoolConfig
	logger       *zap.Logger

	// connsPtr is an immutable slice, replaced via copy-on-write so reads
	// on the request hot path (pickConn) never block on a lock.
	connsPtr atomic.Pointer[[]*http2Conn]

	scalingUp atomic.Bool
	// closed is set when Close begins. A closed pool hands out no connections
	// and opens no new ones, so no socket can be created after Close that
	// nothing is left to release.
	closed    atomic.Bool
	stop      chan struct{}
	monitorWG sync.WaitGroup
	closeOnce sync.Once
}

// newHTTP2Pool builds a pool for addr, or returns an error if cfg is invalid.
func newHTTP2Pool(addr string, newTransport func() *http2.Transport, cfg http2PoolConfig, logger *zap.Logger) (*http2Pool, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	p := &http2Pool{
		addr:         addr,
		newTransport: newTransport,
		cfg:          cfg,
		logger:       logger,
		stop:         make(chan struct{}),
	}
	p.connsPtr.Store(&[]*http2Conn{})

	// Building a slot does no I/O (the Transport dials lazily on its first
	// request), so the pool can safely start with its minimum size.
	initial := 1
	if cfg.dynamicScalingEnabled {
		initial = cfg.minConns
	}
	for i := 0; i < initial; i++ {
		p.addConn(p.newConn())
	}

	if cfg.dynamicScalingEnabled {
		p.monitorWG.Add(1)
		go p.monitorLoop()
	}
	return p, nil
}

// maxConnCount is the ceiling on pool size: maxConns when dynamic scaling is
// enabled, otherwise the single fixed connection.
func (p *http2Pool) maxConnCount() int {
	if !p.cfg.dynamicScalingEnabled {
		return 1
	}
	return p.cfg.maxConns
}

// newConn creates a new pool slot. Unlike dialing a raw *http2.ClientConn,
// this performs no I/O: the underlying *http2.Transport connects lazily on
// its first RoundTrip, so this cannot fail.
func (p *http2Pool) newConn() *http2Conn {
	return newHTTP2Conn(p.newTransport())
}

// pickConn returns the least-loaded usable connection, growing the pool if
// none is currently usable, and kicking off a scale-up if the chosen
// connection is already busy. The returned connection has already had
// incInflight called on it; the caller is responsible for a matching
// decInflight once its request completes.
//
// A connection already at or over the configured maxConcurrentStreams is
// excluded from selection entirely, not just deprioritized: a
// *http2.Transport, unlike a *http2.ClientConn, has no CanTakeNewRequest
// gate to safely queue an excess request against the peer's real limit
// instead of exceeding it, and its own default over-capacity handling has
// known raciness against concurrent callers. Dispatching another request
// onto an already-saturated slot risks the peer resetting the stream with a
// protocol error instead of yarpc ever finding out it was full.
//
// incInflight is called here, before returning, rather than leaving it to
// the caller: reserving the slot as part of the same decision that judged
// it eligible closes most of the gap a concurrent burst of callers could
// otherwise race through between "checked eligible" and "recorded as used".
func (p *http2Pool) pickConn() (*http2Conn, error) {
	if p.closed.Load() {
		return nil, errNoConnsAvailable
	}
	conns := *p.connsPtr.Load()
	maxPerConn := int(p.cfg.maxConcurrentStreams)

	var best *http2Conn
	for _, c := range conns {
		if !c.usable() {
			continue
		}
		if maxPerConn > 0 && c.streamsActive() >= maxPerConn {
			continue
		}
		if best == nil || c.streamsActive() < best.streamsActive() {
			best = c
		}
	}

	if best == nil {
		// Cold start, every connection is parked, or every connection is
		// already at capacity: grow the pool so the caller has something to
		// use now.
		conn, err := p.growPool()
		if err != nil {
			return nil, err
		}
		conn.incInflight()
		return conn, nil
	}

	p.maybeScaleUp(best)
	best.incInflight()
	return best, nil
}

// growPool makes one more connection available: it re-activates a parked
// connection if there is one, otherwise adds a new one, unless the pool is
// already at its configured maximum, in which case it falls back to the
// least-bad existing connection (mirroring the way a single shared
// http2.Transport would queue an excess request rather than fail it).
func (p *http2Pool) growPool() (*http2Conn, error) {
	if p.closed.Load() {
		return nil, errNoConnsAvailable
	}
	if c := p.unparkConn(); c != nil {
		return c, nil
	}

	c := p.newConn()
	if !p.addConnBelowMax(c) {
		// Another caller filled the pool first, or it was already full.
		return leastLoaded(*p.connsPtr.Load())
	}
	if p.closed.Load() {
		// Close ran while this connection was being added, possibly after it
		// already walked the pool, so nothing else will release it.
		c.shutdown()
		return nil, errNoConnsAvailable
	}
	p.logger.Debug("http2 pool: added connection",
		zap.String("peer", p.addr), zap.Int("conns", len(*p.connsPtr.Load())))
	return c, nil
}

// unparkConn re-activates the most recently parked connection and returns it,
// or returns nil if no connection is parked.
func (p *http2Pool) unparkConn() *http2Conn {
	conns := *p.connsPtr.Load()
	for i := len(conns) - 1; i >= 0; i-- {
		if conns[i].parked() && conns[i].unpark() {
			return conns[i]
		}
	}
	return nil
}

// leastLoaded returns the usable conn with the fewest active streams. Parked
// conns are only considered when no conn is usable, as a last resort so a
// request is not failed while the pool still holds a connection. Saturation
// is deliberately not considered: callers use this once they have decided to
// queue an excess request on the least-bad connection.
func leastLoaded(conns []*http2Conn) (*http2Conn, error) {
	var best, bestParked *http2Conn
	for _, c := range conns {
		if c.usable() {
			if best == nil || c.streamsActive() < best.streamsActive() {
				best = c
			}
			continue
		}
		if bestParked == nil || c.streamsActive() < bestParked.streamsActive() {
			bestParked = c
		}
	}
	switch {
	case best != nil:
		return best, nil
	case bestParked != nil:
		return bestParked, nil
	default:
		return nil, errNoConnsAvailable
	}
}

// maybeScaleUp makes an additional connection available, at most once
// concurrently, when least is already busy enough that new requests risk
// queuing behind it. A parked connection is re-activated in preference to
// building a new one. Unlike dialing a raw *http2.ClientConn, growing the pool here is pure
// in-memory work with no I/O, so this runs synchronously on the caller's
// goroutine rather than kicking off a background dial.
//
// Because pickConn's callers can pass a snapshot of least that is already
// stale by the time this goroutine wins the CAS below (e.g. a burst of
// requests that all observed the same overloaded connection before any
// scale-up landed), the threshold is re-checked against a fresh scan of the
// pool once the CAS is won, rather than trusting least. Without that
// re-check, every caller in the burst would independently pass its own
// stale check as scalingUp flips back to false between each near-instant
// scale-up, growing the pool all the way to maxConns instead of by one.
func (p *http2Pool) maybeScaleUp(least *http2Conn) {
	max := p.cfg.maxConcurrentStreams
	if p.closed.Load() || !p.cfg.dynamicScalingEnabled || max <= 0 {
		return
	}
	if float64(least.streamsActive()) < float64(max)*p.cfg.scaleUpThreshold {
		return
	}
	if !p.scalingUp.CompareAndSwap(false, true) {
		return // a scale-up is already in flight
	}
	defer p.scalingUp.Store(false)

	conns := *p.connsPtr.Load()
	if fresh, err := leastLoaded(activeConns(conns)); err == nil &&
		float64(fresh.streamsActive()) < float64(max)*p.cfg.scaleUpThreshold {
		return
	}

	if c := p.unparkConn(); c != nil {
		p.logger.Debug("http2 pool: re-activated parked connection", zap.String("peer", p.addr))
		return
	}
	c := p.newConn()
	if !p.addConnBelowMax(c) {
		return
	}
	if p.closed.Load() {
		c.shutdown() // see growPool
		return
	}
	p.logger.Debug("http2 pool: added connection",
		zap.String("peer", p.addr), zap.Int("conns", len(*p.connsPtr.Load())))
}

// monitorLoop periodically scales the pool down when load no longer
// justifies the current connection count, and releases the sockets of
// connections that have stayed parked and idle.
func (p *http2Pool) monitorLoop() {
	defer p.monitorWG.Done()
	ticker := time.NewTicker(p.cfg.scalingMonitorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.maybeScaleDown()
			p.cleanupConns()
		case <-p.stop:
			return
		}
	}
}

// maybeScaleDown parks the last active connection if the remaining active
// connections can absorb the current total load while staying below the
// scale-down threshold, and the pool is above its configured minimum.
//
// The scale-down threshold is scaleUpThreshold - scaleDownGap, so the load
// the surviving connections inherit is strictly below the level at which
// maybeScaleUp would immediately re-activate the connection just parked.
// That gap is what keeps the pool from flapping between sizes.
func (p *http2Pool) maybeScaleDown() {
	if !p.cfg.dynamicScalingEnabled {
		return
	}
	conns := activeConns(*p.connsPtr.Load())
	if len(conns) <= p.cfg.minConns || len(conns) < 2 {
		return
	}

	maxPerConn := int(p.cfg.maxConcurrentStreams)
	if maxPerConn <= 0 {
		return
	}

	var totalActive int
	for _, c := range conns {
		totalActive += c.streamsActive()
	}

	remainingCapacity := float64(maxPerConn * (len(conns) - 1))
	scaleDownThreshold := p.cfg.scaleUpThreshold - p.cfg.scaleDownGap
	if scaleDownThreshold <= 0 {
		return
	}
	if float64(totalActive) < remainingCapacity*scaleDownThreshold && conns[len(conns)-1].park() {
		p.logger.Debug("http2 pool: parked connection",
			zap.String("peer", p.addr), zap.Int("active", len(conns)-1), zap.Int("inflight", totalActive))
	}
}

// cleanupConns closes the idle sockets of parked connections that have been
// idle past idleTimeout. The connection stays in the pool, so re-activating
// it later simply redials lazily on the next request.
func (p *http2Pool) cleanupConns() {
	for _, c := range *p.connsPtr.Load() {
		if !c.parked() || c.streamsActive() != 0 {
			continue
		}
		idleSince := c.idleSince()
		if idleSince.IsZero() || time.Since(idleSince) <= p.cfg.idleTimeout {
			continue
		}
		c.closeIdle()
	}
}

func activeConns(conns []*http2Conn) []*http2Conn {
	out := make([]*http2Conn, 0, len(conns))
	for _, c := range conns {
		if !c.parked() {
			out = append(out, c)
		}
	}
	return out
}

// addConn appends c to the pool via copy-on-write.
func (p *http2Pool) addConn(c *http2Conn) {
	for {
		old := p.connsPtr.Load()
		next := make([]*http2Conn, 0, len(*old)+1)
		next = append(next, *old...)
		next = append(next, c)
		if p.connsPtr.CompareAndSwap(old, &next) {
			return
		}
	}
}

// addConnBelowMax appends c via copy-on-write only if doing so keeps the pool
// within maxConnCount, and reports whether it was added. The size check and
// the swap happen against the same snapshot, so concurrent growers cannot
// overshoot the limit the way a separate check-then-addConn would.
func (p *http2Pool) addConnBelowMax(c *http2Conn) bool {
	for {
		old := p.connsPtr.Load()
		if len(*old) >= p.maxConnCount() {
			return false
		}
		next := make([]*http2Conn, 0, len(*old)+1)
		next = append(next, *old...)
		next = append(next, c)
		if p.connsPtr.CompareAndSwap(old, &next) {
			return true
		}
	}
}

// Close permanently shuts the pool down. It marks the pool closed (so
// pickConn, growPool and maybeScaleUp stop handing out or opening
// connections), stops the monitor loop and waits for it to exit, then shuts
// down every connection: idle sockets are closed immediately, and busy ones
// as soon as their last in-flight request completes. A closed pool cannot be
// reused; callers get errNoConnsAvailable. It is safe to call more than once.
func (p *http2Pool) Close() {
	p.closeOnce.Do(func() {
		p.closed.Store(true)
		close(p.stop)
		p.monitorWG.Wait()
		for _, c := range *p.connsPtr.Load() {
			c.shutdown()
		}
	})
}
