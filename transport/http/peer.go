// Copyright (c) 2020 Uber Technologies, Inc.
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
	"net"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/zap"
)

type httpPeer struct {
	*abstractpeer.Peer

	transport             *Transport
	addr                  string
	changed               chan struct{}
	released              chan struct{}
	timer                 *time.Timer
	innocentUntilUnixNano *atomic.Int64
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

	p.transport.logger.Info(
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
			p.transport.logger.Info(
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
	p.transport.logger.Info(
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
