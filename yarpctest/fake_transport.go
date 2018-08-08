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

package yarpctest

import (
	"fmt"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycletest"
)

// FakeTransportOption is an option for NewFakeTransport.
type FakeTransportOption func(*FakeTransport)

// NopTransportOption returns a no-op option for NewFakeTransport.
// The option exists to verify that options work.
func NopTransportOption(nopOption string) FakeTransportOption {
	return func(t *FakeTransport) {
		t.nopOption = nopOption
	}
}

// InitialConnectionStatus specifies the initial connection status for new
// peers of this transport.  This is Available by default.  With the status set
// to Unavailable, the test may manual simmulate connection and disconnection
// with the SimulateConnect and SimulateDisconnect methods.
func InitialConnectionStatus(s peer.ConnectionStatus) FakeTransportOption {
	return func(t *FakeTransport) {
		t.initialConnectionStatus = s
	}
}

// NewFakeTransport returns a fake transport.
func NewFakeTransport(opts ...FakeTransportOption) *FakeTransport {
	t := &FakeTransport{
		Lifecycle:               lifecycletest.NewNop(),
		initialConnectionStatus: peer.Available,
		peers: make(map[string]*FakePeer),
		mu:    &sync.Mutex{},
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// FakeTransport is a fake transport.
type FakeTransport struct {
	transport.Lifecycle
	nopOption               string
	initialConnectionStatus peer.ConnectionStatus
	peers                   map[string]*FakePeer
	mu                      *sync.Mutex
}

// NopOption returns the configured nopOption. It's fake.
func (t *FakeTransport) NopOption() string {
	return t.nopOption
}

// SimulateConnect simulates a connection to the peer, marking the peer as
// available and notifying subscribers.
func (t *FakeTransport) SimulateConnect(id peer.Identifier) {
	t.Peer(id).simulateConnect()
}

// SimulateDisconnect simulates a disconnection to the peer, marking the peer
// as unavailable and notifying subscribers.
func (t *FakeTransport) SimulateDisconnect(id peer.Identifier) {
	t.Peer(id).simulateDisconnect()
}

// Peer returns the persistent peer object for that peer identifier for the
// lifetime of the fake transport.
func (t *FakeTransport) Peer(id peer.Identifier) *FakePeer {
	t.mu.Lock()
	defer t.mu.Unlock()

	if p, ok := t.peers[id.Identifier()]; ok {
		return p
	}
	p := &FakePeer{
		id: id,
		status: peer.Status{
			ConnectionStatus: t.initialConnectionStatus,
		},
	}
	t.peers[id.Identifier()] = p
	return p
}

// RetainPeer returns a fake peer.
func (t *FakeTransport) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	peer := t.Peer(id)
	t.mu.Lock()
	defer t.mu.Unlock()
	peer.subscribers = append(peer.subscribers, ps)
	return peer, nil
}

// ReleasePeer does nothing.
func (t *FakeTransport) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	peer := t.Peer(id)
	t.mu.Lock()
	defer t.mu.Unlock()
	if subscribers, count := filterSubscriber(peer.subscribers, ps); count == 0 {
		return fmt.Errorf("no such subscriber")
	} else if count > 1 {
		return fmt.Errorf("extra subscribers: %d", count-1)
	} else {
		peer.subscribers = subscribers
	}
	return nil
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
