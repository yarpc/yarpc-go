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

package http

import (
	"net"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
)

type httpPeer struct {
	*hostport.Peer

	transport *Transport
	addr      string
	changed   chan struct{}
	released  chan struct{}
	timer     *time.Timer
}

func newPeer(addr string, t *Transport) *httpPeer {
	// Create a defused timer for later use.
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	return &httpPeer{
		Peer:      hostport.NewPeer(hostport.PeerIdentifier(addr), t),
		transport: t,
		addr:      addr,
		changed:   make(chan struct{}, 1),
		released:  make(chan struct{}, 0),
		timer:     timer,
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

func (p *httpPeer) OnDisconnected() {
	p.Peer.SetStatus(peer.Unavailable)

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
	var attempts uint

	backoff := p.transport.connBackoffStrategy.Backoff()

	// Wait for start (so we can be certain that we have a channel).
	<-p.transport.once.Started()

	// Attempt to retain an open connection to each peer so long as it is
	// retained.
	for {
		p.Peer.SetStatus(peer.Connecting)
		if p.isAvailable() {
			p.Peer.SetStatus(peer.Available)
			// Reset on success
			attempts = 0
			if !p.waitForChange() {
				break
			}
		} else {
			p.Peer.SetStatus(peer.Unavailable)
			// Back-off on fail
			if !p.sleep(backoff.Duration(attempts)) {
				break
			}
			attempts++
		}
	}
	p.Peer.SetStatus(peer.Unavailable)

	p.transport.connectorsGroup.Done()
}

// waitForChange waits for the transport to send a peer connection status
// change notification, but exits early if the transport releases the peer or
// stops.  waitForChange returns whether it is resuming due to a connection
// status change event.
func (p *httpPeer) waitForChange() (changed bool) {
	// Wait for a connection status change
	select {
	case <-p.changed:
		return true
	case <-p.released:
		return false
	case <-p.transport.once.Stopping():
		return false
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
		<-p.timer.C
	}
	return false
}
