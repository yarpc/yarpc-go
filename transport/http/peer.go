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
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

// h2TransportSeq is a process-wide, monotonically increasing counter used to
// tag every per-peer *http2.Transport with a stable, human-readable ID for
// debug logging (a pointer value alone is hard to eyeball in logs). This
// exists purely to make it easy to verify, from logs, that distinct peers
// (whether from duplicate-address configs or from isolated Dialers) each get
// their own *http2.Transport/connection.
var h2TransportSeq = atomic.NewInt64(0)

type httpPeer struct {
	*abstractpeer.Peer

	transport             *Transport
	addr                  string
	changed               chan struct{}
	released              chan struct{}
	timer                 *time.Timer
	innocentUntilUnixNano *atomic.Int64

	// h2Transport and h2Client are dedicated to this peer: every httpPeer
	// gets its own *http2.Transport (built by transport.newH2Transport) so
	// that duplicate peers pointed at the same address end up with
	// independent HTTP/2 connections instead of sharing one.
	h2Transport *http2.Transport
	h2Client    *http.Client
	// h2ID identifies this peer's dedicated h2Transport in debug logs; see
	// h2TransportSeq.
	h2ID int64
	// h2DialCount counts how many times this peer's h2Transport has dialed a
	// new underlying connection. Logged on every dial so debugging can
	// distinguish "reused an existing HTTP/2 connection" from "opened a new
	// one".
	h2DialCount *atomic.Int64
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

	h2Transport := t.newH2Transport()
	h2ID := h2TransportSeq.Inc()
	h2DialCount := atomic.NewInt64(0)

	// Wrap the dial function so every new HTTP/2 connection this peer opens
	// gets logged with the peer's address and its h2Transport's ID. Two
	// httpPeers pointed at the same address (e.g. via duplicate-peer
	// identifiers or via isolated Dialers) will show up here as two distinct
	// h2ID values each dialing independently, proving they don't share a
	// connection.
	innerDialTLSContext := h2Transport.DialTLSContext
	h2Transport.DialTLSContext = func(ctx context.Context, network, dialAddr string, cfg *tls.Config) (net.Conn, error) {
		seq := h2DialCount.Inc()
		t.logger.Info("http2: dialing new connection for peer",
			zap.String("peer", addr),
			zap.Int64("h2TransportID", h2ID),
			zap.Int64("dialSeq", seq),
			zap.String("network", network),
			zap.String("dialAddr", dialAddr),
		)
		conn, err := innerDialTLSContext(ctx, network, dialAddr, cfg)
		if err != nil {
			t.logger.Info("http2: dial failed for peer",
				zap.String("peer", addr),
				zap.Int64("h2TransportID", h2ID),
				zap.Int64("dialSeq", seq),
				zap.Error(err),
			)
			return nil, err
		}
		t.logger.Info("http2: dial succeeded for peer",
			zap.String("peer", addr),
			zap.Int64("h2TransportID", h2ID),
			zap.Int64("dialSeq", seq),
			zap.String("localAddr", conn.LocalAddr().String()),
			zap.String("remoteAddr", conn.RemoteAddr().String()),
		)
		return conn, nil
	}

	t.logger.Info("http2: created dedicated transport for peer",
		zap.String("peer", addr),
		zap.Int64("h2TransportID", h2ID),
	)

	// Record every dial this peer's dedicated *http2.Transport makes against
	// the transport-wide (not per-peer) HTTP/2 connection metrics. See
	// http2ConnectionMetrics for why these metrics are never tagged by peer.
	//
	// This must be a distinct variable from innerDialTLSContext above: that
	// one is captured by reference inside the logging closure just installed,
	// so reusing it here (with =, not :=) would repoint the logging closure's
	// own "inner" reference at itself once we overwrite h2Transport.DialTLSContext,
	// causing infinite self-recursion (and a stack-overflow crash) on every dial.
	loggingDialTLSContext := h2Transport.DialTLSContext
	h2Transport.DialTLSContext = func(ctx context.Context, network, dialAddr string, cfg *tls.Config) (net.Conn, error) {
		conn, err := loggingDialTLSContext(ctx, network, dialAddr, cfg)
		if err != nil {
			t.h2ConnMetrics.incConnectionDialFailures()
			return nil, err
		}
		t.h2ConnMetrics.incConnectionsDialed()
		return conn, nil
	}

	return &httpPeer{
		Peer:                  abstractpeer.NewPeer(abstractpeer.PeerIdentifier(addr), t),
		transport:             t,
		addr:                  addr,
		changed:               make(chan struct{}, 1),
		released:              make(chan struct{}),
		timer:                 timer,
		innocentUntilUnixNano: atomic.NewInt64(0),
		h2Transport:           h2Transport,
		h2Client:              &http.Client{Transport: h2Transport},
		h2ID:                  h2ID,
		h2DialCount:           h2DialCount,
	}
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
