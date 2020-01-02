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

package yarpctest

import (
	"fmt"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/pkg/lifecycle"
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

// RetainErrors specifies an error for RetainPeer to return for the given
// addresses.
func RetainErrors(err error, addrs []string) FakeTransportOption {
	return func(t *FakeTransport) {
		for _, addr := range addrs {
			t.retainErrors[addr] = err
		}
	}
}

// ReleaseErrors specifies an error for ReleasePeer to return for the given
// addresses.
func ReleaseErrors(err error, addrs []string) FakeTransportOption {
	return func(t *FakeTransport) {
		for _, addr := range addrs {
			t.releaseErrors[addr] = err
		}
	}
}

// NewFakeTransport returns a fake transport.
func NewFakeTransport(opts ...FakeTransportOption) *FakeTransport {
	t := &FakeTransport{
		once:                          lifecycle.NewOnce(),
		initialConnectionStatus:       peer.Available,
		initialPeerConnectionStatuses: make(map[string]peer.ConnectionStatus),

		peers:                make(map[string]*FakePeer),
		retainErrors:         make(map[string]error),
		releaseErrors:        make(map[string]error),
		pendingStatusChanges: make(chan struct{}, 1),
		done:                 make(chan struct{}),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// FakeTransport is a fake transport.
type FakeTransport struct {
	nopOption                     string
	initialConnectionStatus       peer.ConnectionStatus
	initialPeerConnectionStatuses map[string]peer.ConnectionStatus
	retainErrors                  map[string]error
	releaseErrors                 map[string]error

	once                 *lifecycle.Once
	mu                   sync.RWMutex
	peers                map[string]*FakePeer
	changesQueue         []statusChange
	pendingStatusChanges chan struct{}
	done                 chan struct{}
}

// NopOption returns the configured nopOption. It's fake.
func (t *FakeTransport) NopOption() string {
	return t.nopOption
}

// SimulateRetainError leaves a note that any subsequent Retain for a
// particular address should return an error.
func (t *FakeTransport) SimulateRetainError(id peer.Identifier, err error) {
	t.retainErrors[id.Identifier()] = err
}

// SimulateReleaseError leaves a note that any subsequent Release for a particular
// address should return an error.
func (t *FakeTransport) SimulateReleaseError(id peer.Identifier, err error) {
	t.releaseErrors[id.Identifier()] = err
}

// SimulateStatusChange simulates a connection or disconnection to the peer,
// marking the peer connection status and notifying all subscribers.
func (t *FakeTransport) SimulateStatusChange(id peer.Identifier, status peer.ConnectionStatus) {
	t.Peer(id).simulateStatusChange(status)
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
	if err := t.retainErrors[id.Identifier()]; err != nil {
		return nil, err
	}
	peer := t.Peer(id)
	peer.subscribe(ps)
	t.enqueue(statusChange{
		Peer:   peer,
		Status: t.getInitialStatus(id.Identifier()),
	})

	t.scheduleFlush()
	return peer, nil
}

func (t *FakeTransport) getInitialStatus(addr string) peer.ConnectionStatus {
	if status, ok := t.initialPeerConnectionStatuses[addr]; ok {
		return status
	}
	return t.initialConnectionStatus
}

// ReleasePeer does nothing.
func (t *FakeTransport) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	peer := t.Peer(id)

	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.releaseErrors[id.Identifier()]; err != nil {
		return err
	}

	if count := peer.unsubscribe(ps); count == 0 {
		return fmt.Errorf("no such subscriber")
	} else if count > 1 {
		return fmt.Errorf("extra subscribers: %d", count-1)
	}

	t.scheduleFlush()
	return nil
}

// Start spins up a goroutine to asynchronously flush status change notifications.
//
// If you do not start a fake dialer, you must call Flush explicitly.
func (t *FakeTransport) Start() error {
	return t.once.Start(func() error {
		go t.monitor()
		return nil
	})
}

// Stop shuts down the fake dialer, allowing its status change notification
// loop to exit.
func (t *FakeTransport) Stop() error {
	return t.once.Stop(func() error {
		close(t.done)
		return nil
	})
}

// IsRunning returns whether the fake transport is running.
func (t *FakeTransport) IsRunning() bool {
	return t.once.IsRunning()
}

func (t *FakeTransport) scheduleFlush() {
	select {
	case t.pendingStatusChanges <- struct{}{}:
	default:
	}
}

func (t *FakeTransport) monitor() {
Loop:
	for {
		select {
		case <-t.done:
			break Loop
		case <-t.pendingStatusChanges:
			t.Flush()
		}
	}
}

type statusChange struct {
	Peer   *FakePeer
	Status peer.ConnectionStatus
}

// Flush effects all queued status changes from retaining or releasing peers.
//
// Calling RetainPeer and ReleasePeer schedules a peer status change and its
// notifications.
// Concrete dialer implementations dispatch these notifications from a
// goroutine and subscribers may obtain a lock on the peer status.
// For testability, the fake transport queues these changes and calling Flush
// dispatches the notifications synchronously, but still off the RetainPeer and
// ReleasePeer stacks.
func (t *FakeTransport) Flush() {
	for _, change := range t.dequeue() {
		change.Peer.simulateStatusChange(change.Status)
	}
}

func (t *FakeTransport) enqueue(change statusChange) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.changesQueue = append(t.changesQueue, change)
}

func (t *FakeTransport) dequeue() []statusChange {
	t.mu.Lock()
	defer t.mu.Unlock()

	queue := t.changesQueue
	t.changesQueue = make([]statusChange, 0)
	return queue
}
