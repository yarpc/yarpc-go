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
	"errors"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

var errNoConnsAvailable = errors.New("http2 pool: no connections available")

// http2PoolConfig controls how an http2Pool scales the number of HTTP/2
// connections it maintains to a single peer.
type http2PoolConfig struct {
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

func defaultHTTP2PoolConfig() http2PoolConfig {
	return http2PoolConfig{
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
	stop      chan struct{}
}

func newHTTP2Pool(addr string, newTransport func() *http2.Transport, cfg http2PoolConfig, logger *zap.Logger) *http2Pool {
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
	go p.monitorLoop()
	return p
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
		// Cold start, every connection is draining, or every connection is
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

// growPool adds one more connection to the pool, unless the pool is already
// at its configured maximum, in which case it falls back to the least-bad
// existing connection (mirroring the way a single shared http2.Transport
// would queue an excess request rather than fail it).
func (p *http2Pool) growPool() (*http2Conn, error) {
	conns := *p.connsPtr.Load()
	if len(conns) >= p.cfg.maxConns {
		return leastLoaded(conns)
	}

	c := p.newConn()
	p.addConn(c)
	return c, nil
}

// leastLoaded returns the conn with the fewest active streams, regardless
// of usable(), as a last-resort fallback when nothing better is available.
func leastLoaded(conns []*http2Conn) (*http2Conn, error) {
	var best *http2Conn
	for _, c := range conns {
		if best == nil || c.streamsActive() < best.streamsActive() {
			best = c
		}
	}
	if best == nil {
		return nil, errNoConnsAvailable
	}
	return best, nil
}

// maybeScaleUp adds an additional connection, at most once concurrently,
// when least is already busy enough that new requests risk queuing behind
// it. Unlike dialing a raw *http2.ClientConn, growing the pool here is pure
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
	if max <= 0 {
		return
	}
	if float64(least.streamsActive()) < float64(max)*p.cfg.scaleUpThreshold {
		return
	}
	if len(*p.connsPtr.Load()) >= p.cfg.maxConns {
		return
	}
	if !p.scalingUp.CompareAndSwap(false, true) {
		return // a scale-up is already in flight
	}
	defer p.scalingUp.Store(false)

	conns := *p.connsPtr.Load()
	if len(conns) >= p.cfg.maxConns {
		return
	}
	if fresh, err := leastLoaded(conns); err == nil &&
		float64(fresh.streamsActive()) < float64(max)*p.cfg.scaleUpThreshold {
		return
	}

	p.addConn(p.newConn())
}

// monitorLoop periodically scales the pool down when load no longer
// justifies the current connection count, and reaps drained/idle
// connections.
func (p *http2Pool) monitorLoop() {
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

// maybeScaleDown marks the most-loaded connection as draining if the
// remaining connections can absorb the current total load, with a
// scaleDownGap safety margin, and the pool is above its configured
// minimum.
func (p *http2Pool) maybeScaleDown() {
	conns := activeConns(*p.connsPtr.Load())
	if len(conns) <= p.cfg.minConns {
		return
	}

	maxPerConn := int(p.cfg.maxConcurrentStreams)
	if maxPerConn <= 0 {
		return
	}

	var totalActive int
	var mostLoaded *http2Conn
	for _, c := range conns {
		totalActive += c.streamsActive()
		if mostLoaded == nil || c.streamsActive() > mostLoaded.streamsActive() {
			mostLoaded = c
		}
	}
	if mostLoaded == nil {
		return
	}

	totalCapacity := maxPerConn * len(conns)
	remainingCapacity := totalCapacity - maxPerConn
	if remainingCapacity <= 0 {
		return
	}
	if float64(totalActive) < float64(remainingCapacity)*(1-p.cfg.scaleDownGap) {
		mostLoaded.markDraining()
	}
}

// cleanupConns closes and removes draining connections that have gone
// idle, and marks connections that have been idle past idleTimeout as
// draining so a future tick can close them.
func (p *http2Pool) cleanupConns() {
	conns := *p.connsPtr.Load()
	for _, c := range conns {
		if c.draining() && c.streamsActive() == 0 {
			if err := c.close(); err != nil {
				p.logger.Warn("http2 pool: failed to close drained connection",
					zap.String("peer", p.addr),
					zap.Error(err))
			}
			p.removeConn(c)
			continue
		}
		if idleSince := c.idleSince(); !idleSince.IsZero() && time.Since(idleSince) > p.cfg.idleTimeout {
			c.markDraining()
		}
	}
}

func activeConns(conns []*http2Conn) []*http2Conn {
	out := make([]*http2Conn, 0, len(conns))
	for _, c := range conns {
		if !c.draining() {
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

// removeConn removes c from the pool via copy-on-write. It is a no-op if c
// is not present (e.g. concurrently removed already).
func (p *http2Pool) removeConn(c *http2Conn) {
	for {
		old := p.connsPtr.Load()
		idx := -1
		for i, cur := range *old {
			if cur == c {
				idx = i
				break
			}
		}
		if idx == -1 {
			return
		}
		next := make([]*http2Conn, 0, len(*old)-1)
		next = append(next, (*old)[:idx]...)
		next = append(next, (*old)[idx+1:]...)
		if p.connsPtr.CompareAndSwap(old, &next) {
			return
		}
	}
}

// Close stops the pool's monitor loop and closes every connection it
// holds.
func (p *http2Pool) Close() {
	close(p.stop)
	for _, c := range *p.connsPtr.Load() {
		if err := c.close(); err != nil {
			p.logger.Warn("http2 pool: failed to close connection",
				zap.String("peer", p.addr),
				zap.Error(err))
		}
	}
}
