// Copyright (c) 2017 Uber Technologies, Inc.
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
	"net/http"
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
}

func newPeer(pid hostport.PeerIdentifier, t *Transport) *httpPeer {
	return &httpPeer{
		Peer:      hostport.NewPeer(pid, t),
		transport: t,
		addr:      pid.Identifier(),
		changed:   make(chan struct{}, 1),
		released:  make(chan struct{}, 0),
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

func (i *Inbound) onConnStateChanged(c net.Conn, s http.ConnState) {
	id := c.LocalAddr().String()
	p := i.transport.getPeerWithLock(id)
	if p == nil {
		return
	}

	// Kick the state change channel (if it hasn't been kicked already).
	select {
	case p.changed <- struct{}{}:
	default:
	}
}

func (p *httpPeer) maintainConn() {
	var attempts uint

	backoff := p.transport.newConnBackoff()

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
			if !p.sleep(backoff(attempts)) {
				break
			}
			attempts++
		}
	}

	p.transport.connectorsGroup.Done()
}

// waitForChange waits for the transport to send a peer connection status
// change notification, but exits early if the transport releases the peer or
// stops.  waitForChange returns whether it is resuming due to a connection
// status change event.
func (p *httpPeer) waitForChange() bool {
	// Wait for a connection status change
	select {
	case <-p.changed:
		return true
	case <-p.released:
		return false
	}
}

// sleep waits for a duration, but exits early if the transport releases the
// peer or stops.  sleep returns whether it successfully waited the entire
// duration.
func (p *httpPeer) sleep(delay time.Duration) bool {
	select {
	case <-time.After(delay):
		return true
	case <-p.released:
		return false
	}
}
