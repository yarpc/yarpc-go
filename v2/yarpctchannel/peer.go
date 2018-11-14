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

package yarpctchannel

import (
	"context"
	"time"

	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcpeer"
)

type tchannelPeer struct {
	*yarpcpeer.AbstractPeer

	dialer   *Dialer
	addr     string
	changed  chan struct{}
	released chan struct{}
	timer    *time.Timer
}

func newPeer(id yarpc.Identifier, addr string, dialer *Dialer) *tchannelPeer {
	// Create a defused timer for later use.
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	return &tchannelPeer{
		AbstractPeer: yarpcpeer.NewAbstractPeer(id),

		addr:     addr,
		dialer:   dialer,
		changed:  make(chan struct{}, 1),
		released: make(chan struct{}, 0),
		timer:    timer,
	}
}

func (p *tchannelPeer) MaintainConn() {
	defer func() {
		p.dialer.connectorsGroup.Done()
	}()

	cancel := func() {}

	backoff := p.dialer.ConnBackoff.Backoff()
	var attempts uint

	pl := p.dialer.ch.RootPeers()

	// Attempt to retain an open connection to each peer so long as it is
	// retained.
	for {
		tp := pl.GetOrAdd(p.addr)

		inbound, outbound := tp.NumConnections()
		if inbound+outbound > 0 {
			p.setStatus(yarpc.Available)
			// Reset on success
			attempts = 0
			if !p.waitForChange() {
				break
			}

		} else {
			p.setStatus(yarpc.Unavailable)

			// Attempt to connect
			ctx := context.Background()
			ctx, cancel = context.WithTimeout(ctx, p.dialer.ConnTimeout)
			_, err := tp.Connect(ctx)

			if err == nil {
				p.setStatus(yarpc.Available)
			} else {
				p.setStatus(yarpc.Unavailable)
				// Back-off on fail
				if !p.sleep(backoff.Duration(attempts)) {
					break
				}
				attempts++
			}

		}
	}

	cancel()
}

func (p *tchannelPeer) setStatus(status yarpc.ConnectionStatus) {
	p.AbstractPeer.SetStatus(status)
}

func (p *tchannelPeer) Release() {
	close(p.released)
}

func (p *tchannelPeer) OnStatusChanged() {
	select {
	case p.changed <- struct{}{}:
	default:
	}
}

// waitForChange waits for the dialer to send a peer connection status
// change notification, but exits early if the dialer releases the peer or
// stops.  waitForChange returns whether it is resuming due to a connection
// status change event.
func (p *tchannelPeer) waitForChange() (changed bool) {
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
func (p *tchannelPeer) sleep(delay time.Duration) (completed bool) {
	p.timer.Reset(delay)

	select {
	case <-p.timer.C:
		return true
	case <-p.released:
	}

	if !p.timer.Stop() {
		<-p.timer.C
	}
	return false
}
