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

package yarpctest

import (
	"sync"

	"go.uber.org/yarpc/api/peer"
)

// FakePeer is a fake peer with an identifier.
type FakePeer struct {
	id peer.Identifier

	// mutable
	lock sync.RWMutex
	// subscribers needs to be modified under lock in FakeTransport
	subscribers []peer.Subscriber
	status      peer.Status
}

// Identifier returns the fake peer identifier.
func (p *FakePeer) Identifier() string {
	return p.id.Identifier()
}

// String returns a humane representation of the peer and its status for debugging.
func (p *FakePeer) String() string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.id.Identifier() + ":" + p.status.String()
}

// Status returns the fake peer status.
func (p *FakePeer) Status() peer.Status {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.status
}

// StartRequest increments pending request count.
func (p *FakePeer) StartRequest() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.status.PendingRequestCount++
}

// EndRequest decrements pending request count.
func (p *FakePeer) EndRequest() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.status.PendingRequestCount--
}

func (p *FakePeer) simulateConnect() {
	p.simulateStatusChange(peer.Available)
}

func (p *FakePeer) simulateDisconnect() {
	p.simulateStatusChange(peer.Unavailable)
}

func (p *FakePeer) simulateStatusChange(status peer.ConnectionStatus) {
	for _, sub := range p.prepareStatusChange(status) {
		sub.NotifyStatusChanged(p.id)
	}
}

func (p *FakePeer) prepareStatusChange(status peer.ConnectionStatus) []peer.Subscriber {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.status.ConnectionStatus = status
	subscribers := make([]peer.Subscriber, len(p.subscribers))
	copy(subscribers, p.subscribers)
	return subscribers
}

func (p *FakePeer) subscribe(ps peer.Subscriber) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.subscribers = append(p.subscribers, ps)
}

func (p *FakePeer) unsubscribe(ps peer.Subscriber) int {
	p.lock.Lock()
	defer p.lock.Unlock()

	subscribers, count := filterSubscriber(p.subscribers, ps)
	p.subscribers = subscribers
	return count
}

func filterSubscriber(subs []peer.Subscriber, ps peer.Subscriber) ([]peer.Subscriber, int) {
	res := make([]peer.Subscriber, 0, len(subs))
	count := 0
	for _, sub := range subs {
		if sub != ps {
			res = append(res, sub)
		} else {
			count++
		}
	}
	return res, count
}
