// Copyright (c) 2019 Uber Technologies, Inc.
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

package tchannel

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/abstractpeer"
)

type tchannelPeer struct {
	*abstractpeer.Peer

	transport *Transport
	addr      string
	changed   chan struct{}
	pending   chan struct{}
	released  chan struct{}
	timer     *time.Timer
}

func newPeer(addr string, t *Transport) *tchannelPeer {
	// Create a defused timer for later use.
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	return &tchannelPeer{
		addr:      addr,
		Peer:      abstractpeer.NewPeer(abstractpeer.PeerIdentifier(addr), t),
		transport: t,
		changed:   make(chan struct{}, 1),
		released:  make(chan struct{}),
		timer:     timer,
	}
}

func (p *tchannelPeer) MaintainConn() {
	cancel := func() {}

	backoff := p.transport.connBackoffStrategy.Backoff()
	var attempts uint

	// Wait for start (so we can be certain that we have a channel).
	<-p.transport.once.Started()
	pl := p.transport.peerList()
	if pl == nil {
		return
	}

	// Attempt to retain an open connection to each peer so long as it is
	// retained.
	for {
		tp := pl.GetOrAdd(p.addr)

		inbound, outbound := tp.NumConnections()
		if inbound+outbound > 0 {
			p.setConnectionStatus(peer.Available)
			// Reset on success
			attempts = 0
			if !p.waitForChange() {
				break
			}

		} else {
			p.setConnectionStatus(peer.Connecting)

			// Attempt to connect
			ctx := context.Background()
			ctx, cancel = context.WithTimeout(ctx, p.transport.connTimeout)
			_, err := tp.Connect(ctx)

			if err == nil {
				p.setConnectionStatus(peer.Available)
			} else {
				p.setConnectionStatus(peer.Unavailable)
				// Back-off on fail
				if !p.sleep(backoff.Duration(attempts)) {
					break
				}
				attempts++
			}

		}
	}

	p.transport.connectorsGroup.Done()
	cancel()
}

func (p *tchannelPeer) Release() {
	close(p.released)
}

func (p *tchannelPeer) setConnectionStatus(status peer.ConnectionStatus) {
	p.Peer.SetStatus(status)
	p.Peer.NotifyStatusChanged()
}

func (p *tchannelPeer) OnStatusChanged() {
	select {
	case p.changed <- struct{}{}:
	default:
	}
}

// waitForChange waits for the transport to send a peer connection status
// change notification, but exits early if the transport releases the peer or
// stops.  waitForChange returns whether it is resuming due to a connection
// status change event.
func (p *tchannelPeer) waitForChange() (changed bool) {
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
func (p *tchannelPeer) sleep(delay time.Duration) (completed bool) {
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

func (p *tchannelPeer) monitorPendingRequestCount() {
	for {
		select {
		case <-p.released:
			return
		case <-p.pending:
			p.Peer.NotifyStatusChanged()
		}
	}
}

func (p *tchannelPeer) notifyPendingRequestCountChanged() {
	// kick the pending request count change channel.
	// monitorPendingRequestCount broadcasts changes to subscribers so
	// StartRequest() and EndRequest() don't reply to peer lists on the stack,
	// possibly causing deadlock.
	select {
	case p.pending <- struct{}{}:
	default:
	}
}

func (p *tchannelPeer) StartRequest() {
	p.Peer.StartRequest()
	p.notifyPendingRequestCountChanged()
}

func (p *tchannelPeer) EndRequest() {
	p.Peer.EndRequest()
	p.notifyPendingRequestCountChanged()
}
