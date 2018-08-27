// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpchttp

import (
	"net"
	"time"

	"go.uber.org/atomic"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeer"
)

type httpPeer struct {
	*yarpcpeer.AbstractPeer

	dialer                *dialerInternals
	addr                  string
	changed               chan struct{}
	released              chan struct{}
	timer                 *time.Timer
	innocentUntilUnixNano *atomic.Int64
}

func newPeer(addr string, dialer *dialerInternals) *httpPeer {
	// Create a defused timer for later use.
	timer := time.NewTimer(0)
	if !timer.Stop() {
		// not reachable, but if the timer wins the race, it would mean
		// deadlock later, so best to conditionally drain the channel just in
		// that case.
		<-timer.C
	}

	return &httpPeer{
		AbstractPeer: yarpcpeer.NewAbstractPeer(yarpc.Address(addr)),
		dialer:       dialer,
		addr:         addr,
		changed:      make(chan struct{}, 1),
		released:     make(chan struct{}, 0),
		timer:        timer,
		innocentUntilUnixNano: atomic.NewInt64(0),
	}
}

// The HTTP dialer polls for whether a peer is available by attempting to
// connect. The dialer does not preserve the connection because HTTP servers
// may behave oddly if they don't receive a request immediately.
// Instead, we treat the peer as available until proven otherwise with a fresh
// connection attempt.
func (p *httpPeer) isAvailable() bool {
	// If there's no open connection, we probe by connecting.
	dialer := &net.Dialer{Timeout: p.dialer.connTimeout}
	conn, err := dialer.Dial("tcp", p.addr)
	if conn != nil {
		conn.Close()
	}
	if conn != nil && err == nil {
		return true
	}
	return false
}

func (p *httpPeer) OnSuspect() {
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
	innocentDurationUnixNano := p.dialer.jitter(p.dialer.innocenceWindow.Nanoseconds())
	p.innocentUntilUnixNano.Store(now + innocentDurationUnixNano)

	// Kick the state change channel (if it hasn't been kicked already).
	// But leave status as available.
	select {
	case p.changed <- struct{}{}:
	default:
	}
}

func (p *httpPeer) OnDisconnected() {
	p.AbstractPeer.SetStatus(yarpc.Unavailable)

	// Kick the state change channel (if it hasn't been kicked already).
	select {
	case p.changed <- struct{}{}:
	default:
	}
}

func (p *httpPeer) Release() {
	close(p.released)
}

func (p *httpPeer) MaintainConn() {
	defer func() {
		p.dialer.connectorsGroup.Done()
	}()

	var attempts uint

	backoff := p.dialer.connBackoffStrategy.Backoff()

	// Attempt to retain an open connection to each peer so long as it is
	// retained.
	p.AbstractPeer.SetStatus(yarpc.Unavailable)
	for {
		// Invariant: Status is Unavailable initially, or after exponential
		// back-off, or after OnDisconnected, but still Available after
		// OnSuspect.
		if p.isAvailable() {
			p.AbstractPeer.SetStatus(yarpc.Available)
			// Reset on success
			attempts = 0
			if !p.waitForChange() {
				break
			}
			// Invariant: the status is Unavailable if change is triggered by
			// OnDisconnected, but remains Available if triggered by OnSuspect.
		} else {
			p.AbstractPeer.SetStatus(yarpc.Unavailable)
			// Back-off on fail
			if !p.sleep(backoff.Duration(attempts)) {
				break
			}
			attempts++
			p.AbstractPeer.SetStatus(yarpc.Unavailable)
		}
	}
	p.AbstractPeer.SetStatus(yarpc.Unavailable)
}

// waitForChange waits for the dialer to send a peer connection status
// change notification, but exits early if the dialer releases the peer or
// stops.  waitForChange returns whether it is resuming due to a connection
// status change event.
func (p *httpPeer) waitForChange() (changed bool) {
	// Wait for a connection status change
	select {
	case <-p.changed:
		return true
	case <-p.released:
		return false
	}
}

// sleep waits for a duration, but exits early if the dialer releases the
// peer or stops.  sleep returns whether it successfully waited the entire
// duration.
func (p *httpPeer) sleep(delay time.Duration) (completed bool) {
	p.timer.Reset(delay)

	select {
	case <-p.timer.C:
		return true
	case <-p.released:
	}

	if !p.timer.Stop() {
		// This branch is very difficult to reach, as stopping a timer almost
		// always succeeds.
		<-p.timer.C
	}
	return false
}
