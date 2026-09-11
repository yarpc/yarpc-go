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
	"context"
	"net"
	"sync"
	stdatomic "sync/atomic"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

type httpPeer struct {
	*abstractpeer.Peer

	transport             *Transport
	addr                  string
	changed               chan struct{}
	released              chan struct{}
	timer                 *time.Timer
	innocentUntilUnixNano *atomic.Int64

	// h2Pool manages this peer's HTTP/2 connection(s): every httpPeer gets
	// its own pool (rather than sharing one across peers) so that duplicate
	// peers pointed at the same address end up with independent HTTP/2
	// connections. The pool is pinned to exactly one connection today (see
	// Transport.h2PoolConfig).
	//
	// It is created lazily, on first use of h2Sender, rather than in
	// newPeer: Pool.Start spawns a background teardown-watcher goroutine
	// even with zero initial connections, and most peers (anything not
	// using UseHTTP2) never need one at all. Read via loadH2Pool, which is
	// safe to call concurrently without holding h2DialMu.
	h2Pool stdatomic.Pointer[connpool.Pool[*http2.ClientConn]]

	// h2DialMu serializes both lazily creating h2Pool and the lazy first
	// dial in h2Sender: nothing else calls h2Pool.AddConn today, so without
	// this, concurrent requests to a cold peer could each create their own
	// pool or dial their own connection, exceeding the pool's pinned
	// single-connection invariant.
	h2DialMu sync.Mutex
}

func newPeer(addr string, t *Transport) *httpPeer {
	// Create a defused timer for later use.
	timer := time.NewTimer(0)
	if !timer.Stop() {
		// not reachable, but if the timer wins the race, it would mean
		// deadlock later, so best to conditionally drain the channel just in
		// that case.
		<-timer.C
	}

	return &httpPeer{
		Peer:                  abstractpeer.NewPeer(abstractpeer.PeerIdentifier(addr), t),
		transport:             t,
		addr:                  addr,
		changed:               make(chan struct{}, 1),
		released:              make(chan struct{}),
		timer:                 timer,
		innocentUntilUnixNano: atomic.NewInt64(0),
	}
}

// loadH2Pool returns this peer's HTTP/2 connection pool, or nil if h2Sender
// has never been called for this peer.
func (p *httpPeer) loadH2Pool() *connpool.Pool[*http2.ClientConn] {
	return p.h2Pool.Load()
}

// h2Sender returns a sender that routes through this peer's HTTP/2
// connection pool, creating the pool and dialing its first connection
// lazily on demand.
func (p *httpPeer) h2Sender() (sender, error) {
	if pool := p.loadH2Pool(); pool != nil {
		if w := pool.PickConn(); w != nil {
			return &h2ConnSender{wrapper: w}, nil
		}
	}

	p.h2DialMu.Lock()
	defer p.h2DialMu.Unlock()

	pool := p.loadH2Pool()
	if pool == nil {
		pool = connpool.NewPool(
			context.Background(),
			p.transport.h2PoolConfig,
			func(ctx context.Context) (*http2.ClientConn, error) { return p.transport.dialH2Conn(ctx, p.addr) },
			p.transport.logger,
			p.addr,
			connpool.NewReporter(p.transport.h2PoolMetrics),
		)
		pool.OnConnAdded = p.watchH2Conn
		// initialConnCount is 0, not 1: unlike gRPC's Dial, a raw HTTP/2
		// dial can fail synchronously on a genuinely down destination, and
		// Pool.Start cancels (permanently disables) the pool if any initial
		// dial fails -- eagerly dialing here would take the peer's HTTP/2
		// path out permanently instead of retrying on the next request.
		// This call cannot itself fail with count 0.
		_ = pool.Start(0)
		p.h2Pool.Store(pool)
		p.transport.h2ActivePeers.Inc()
	}

	// Re-check: another goroutine may have dialed the first connection while
	// we were waiting for the lock.
	if w := pool.PickConn(); w != nil {
		return &h2ConnSender{wrapper: w}, nil
	}
	w, err := pool.AddConn()
	if err != nil {
		return nil, err
	}
	return &h2ConnSender{wrapper: w}, nil
}

// watchH2Conn is h2Pool's OnConnAdded callback. http2.ClientConn has no
// blocking wait-for-state-change primitive (unlike grpc.ClientConn's
// WaitForStateChange, see transport/grpc/peer.go), so health is polled
// instead: once the connection can no longer take new requests, it is
// scheduled for removal, and this goroutine (as required by
// connpool.Pool.OnConnAdded's contract) closes it and removes it from the
// pool once that removal is observed.
func (p *httpPeer) watchH2Conn(w *connpool.Wrapper[*http2.ClientConn]) {
	go func() {
		ticker := time.NewTicker(defaultH2ConnHealthPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if !w.Conn.CanTakeNewRequest() {
					w.TransitionState(connpool.StateActive, connpool.StateDraining)
					w.Cancel()
				}
			case <-w.Context().Done():
				w.Conn.Close()
				pool := p.loadH2Pool()
				pool.Remove(w)
				// AddConn refreshes the pool's connection-state gauges after
				// publishing a new connection, but Remove itself does not (see
				// Remove's doc comment) -- without this, a connection removed
				// here would leave the shared gauges reporting it as
				// active/draining until the pool's next AddConn, understating
				// a leak if one never comes (e.g. this goroutine failing to
				// reach this point at all).
				pool.RefreshMetrics()
				pool.ConnDone()
				return
			}
		}
	}()
}

// The HTTP transport polls for whether a peer is available by attempting to
// connect. The transport does not preserve the connection because HTTP servers
// may behave oddly if they don't receive a request immediately.
// Instead, we treat the peer as available until proven otherwise with a fresh
// connection attempt.
func (p *httpPeer) isAvailable() bool {
	// If there's no open connection, we probe by connecting.
	dialer := &net.Dialer{Timeout: p.transport.connTimeout}
	conn, err := dialer.Dial("tcp", p.addr)
	if conn != nil {
		conn.Close()
	}
	if conn != nil && err == nil {
		return true
	}

	p.transport.logger.Debug(
		"unable to connect to peer, marking as unavailable",
		zap.String("peer", p.addr),
		zap.String("transport", "http"),
	)

	return false
}

// StartRequest and EndRequest are no-ops now.
// They previously aggregated pending request count from all subscibed peer
// lists and distributed change notifications.
// This was fraught with concurrency hazards so we moved pending request count
// tracking into the lists themselves.

func (p *httpPeer) StartRequest() {}

func (p *httpPeer) EndRequest() {}

func (p *httpPeer) notifyStatusChanged() {
	// Kick the state change channel (if it hasn't been kicked already).
	// The peer connection management loop broadcasts status changes, to avoid
	// deadlock on the stack.
	select {
	case p.changed <- struct{}{}:
	default:
	}
}

func (p *httpPeer) onSuspect() {
	now := time.Now().UnixNano()
	innocentUntil := p.innocentUntilUnixNano.Load()

	// Do not check for connectivity after every request timeout.
	// Spread them out so they only occur once in every innocence window.
	if now < innocentUntil {
		return
	}

	// Extend the window of innocence from the current time.
	// Use Store instead of CAS since races at worst extend the innocence
	// window to relatively similar distant times.
	innocentDurationUnixNano := p.transport.jitter(p.transport.innocenceWindow.Nanoseconds())
	p.innocentUntilUnixNano.Store(now + innocentDurationUnixNano)

	p.transport.logger.Debug(
		"peer marked suspicious due to timeout",
		zap.String("peer", p.addr),
		zap.Duration("duration", time.Duration(innocentDurationUnixNano)),
		zap.Time("until", time.Unix(0, innocentDurationUnixNano)),
		zap.String("transport", "http"),
	)

	p.notifyStatusChanged()
}

func (p *httpPeer) onDisconnected() {
	p.Peer.SetStatus(peer.Connecting)
	p.notifyStatusChanged()
}

func (p *httpPeer) Release() {
	close(p.released)
	// Tear down this peer's dedicated HTTP/2 connection pool, if h2Sender
	// ever created one; nothing else references it once the peer is
	// released. Async, like the rest of Release -- Transport.Stop's
	// stopPeerH2Pools is what actually waits for pool teardown to complete.
	if pool := p.loadH2Pool(); pool != nil {
		pool.Stop()
		p.transport.h2ActivePeers.Dec()
	}
}

func (p *httpPeer) MaintainConn() {
	var attempts uint

	backoff := p.transport.connBackoffStrategy.Backoff()

	// Wait for start (so we can be certain that we have a channel).
	<-p.transport.once.Started()

	// Attempt to retain an open connection to each peer so long as it is
	// retained.
	p.setStatus(peer.Connecting)
	for {
		// Invariant: Status is Connecting initially, or after exponential
		// back-off, or after onDisconnected, but still Available after
		// onSuspect.
		if p.isAvailable() {
			p.setStatus(peer.Available)
			// Reset on success
			attempts = 0
			if !p.waitForChange() {
				break
			}
			// Invariant: the status is Connecting if change is triggered by
			// onDisconnected, but remains Available if triggered by onSuspect.
		} else {
			p.setStatus(peer.Unavailable)
			// Back-off on fail
			dur := backoff.Duration(attempts)
			p.transport.logger.Debug(
				"peer connect retry back-off",
				zap.String("peer", p.addr),
				zap.Duration("sleep", dur),
				zap.Time("until", time.Now().Add(dur)),
				zap.Int("attempt", int(attempts)),
				zap.String("transport", "http"),
			)
			if !p.sleep(dur) {
				break
			}
			attempts++
			p.setStatus(peer.Connecting)
		}
	}
	p.setStatus(peer.Unavailable)

	p.transport.connectorsGroup.Done()
}

func (p *httpPeer) setStatus(status peer.ConnectionStatus) {
	p.transport.logger.Debug(
		"peer status change",
		zap.String("status", status.String()),
		zap.String("peer", p.Peer.Identifier()),
		zap.String("transport", "http"),
	)
	p.Peer.SetStatus(status)
	p.Peer.NotifyStatusChanged()
}

// waitForChange waits for the transport to send a peer connection status
// change notification, but exits early if the transport releases the peer or
// stops.  waitForChange returns whether it is resuming due to a connection
// status change event.
func (p *httpPeer) waitForChange() (changed bool) {
	for {
		select {
		case <-p.changed:
			return true
		case <-p.released:
			return false
		}
	}
}

// sleep waits for a duration, but exits early if the transport releases the
// peer or stops.  sleep returns whether it successfully waited the entire
// duration.
func (p *httpPeer) sleep(delay time.Duration) (completed bool) {
	p.timer.Reset(delay)

	select {
	case <-p.timer.C:
		return true
	case <-p.released:
	case <-p.transport.once.Stopping():
	}

	if !p.timer.Stop() {
		// This branch is very difficult to reach, as stopping a timer almost
		// always succeeds.
		<-p.timer.C
	}
	return false
}
