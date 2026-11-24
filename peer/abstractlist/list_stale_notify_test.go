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

package abstractlist

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/peeraddr"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/peer/abstractpeer"
)

// stubPeer is a peer.Peer whose connection status is set directly by the test.
type stubPeer struct {
	id peer.Identifier

	mu     sync.Mutex
	status peer.ConnectionStatus
}

var _ peer.Peer = (*stubPeer)(nil)

func (p *stubPeer) Identifier() string { return p.id.Identifier() }

func (p *stubPeer) Status() peer.Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	return peer.Status{ConnectionStatus: p.status}
}

func (p *stubPeer) setStatus(s peer.ConnectionStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = s
}

func (p *stubPeer) StartRequest() {}
func (p *stubPeer) EndRequest()   {}

// stubTransport hands out one shared stubPeer per identifier, like a real
// transport, and never calls back into subscribers by itself. Tests deliver
// notifications explicitly to simulate late or out-of-order delivery.
type stubTransport struct {
	mu    sync.Mutex
	peers map[string]*stubPeer
}

var _ peer.Transport = (*stubTransport)(nil)

func newStubTransport() *stubTransport {
	return &stubTransport{peers: make(map[string]*stubPeer)}
}

func (t *stubTransport) RetainPeer(id peer.Identifier, _ peer.Subscriber) (peer.Peer, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	p, ok := t.peers[id.Identifier()]
	if !ok {
		// Like the real transports, name the peer by its address, without
		// the "#n" index.
		p = &stubPeer{
			id:     abstractpeer.Identify(peeraddr.Address(id.Identifier())),
			status: peer.Unavailable,
		}
		t.peers[id.Identifier()] = p
	}
	return p, nil
}

func (t *stubTransport) ReleasePeer(peer.Identifier, peer.Subscriber) error { return nil }

func (t *stubTransport) peer(id peer.Identifier) *stubPeer {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.peers[id.Identifier()]
}

// facadeFor returns the live facade the list holds for pid (nil if absent).
func facadeFor(pl *List, pid peer.Identifier) *peerFacade {
	pl.lock.RLock()
	defer pl.lock.RUnlock()
	return pl.peers[pid.Identifier()]
}

// addAvailablePeer adds pid to the list and brings it up via the normal
// notification path.
func addAvailablePeer(t *testing.T, pl *List, tr *stubTransport, pid peer.Identifier) *peerFacade {
	t.Helper()
	require.NoError(t, pl.Update(peer.ListUpdates{Additions: []peer.Identifier{pid}}))
	tr.peer(pid).setStatus(peer.Available)
	pl.NotifyStatusChanged(pid)
	pf := facadeFor(pl, pid)
	require.NotNil(t, pf)
	require.Equal(t, 1, pl.NumAvailable(), "added peer must become available")
	return pf
}

// A transport may notify a subscriber after the list has removed and released
// the peer. An "Available" notification arriving then must not put the removed
// peer back into the implementation.
func TestLateAvailableNotificationAfterRemoveIsIgnored(t *testing.T) {
	tr := newStubTransport()
	impl := &mraList{}
	pl := New("stale-notify", tr, impl, NoShuffle(), FailFast())
	require.NoError(t, pl.Start())
	defer func() { assert.NoError(t, pl.Stop()) }()

	pid := abstractpeer.Identify("10.0.0.1:4040")
	pf := addAvailablePeer(t, pl, tr, pid)

	// Service discovery removes the peer.
	require.NoError(t, pl.Update(peer.ListUpdates{Removals: []peer.Identifier{pid}}))
	require.Equal(t, 0, pl.NumAvailable())
	require.Nil(t, impl.mra)
	require.Nil(t, facadeFor(pl, pid))

	// The transport's connection is still up and a notification for the
	// removed facade is delivered late.
	require.Equal(t, peer.Available, tr.peer(pid).Status().ConnectionStatus)
	pf.NotifyStatusChanged(pid)

	assert.Equal(t, 0, pl.NumAvailable(), "removed peer must not be counted as available")
	assert.Nil(t, impl.mra, "removed peer must not be re-added to the implementation")
	assert.Empty(t, pl.Peers())

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	_, _, err := pl.Choose(ctx, &transport.Request{})
	require.Error(t, err, "removed peer must not be chosen")
	assert.Contains(t, err.Error(), "has no peers")
}

// If an address is removed and re-added, a late notification on the old
// facade must not add the old facade alongside the new one.
func TestLateNotificationOnReplacedFacadeIsIgnored(t *testing.T) {
	tr := newStubTransport()
	impl := &mraList{}
	pl := New("stale-notify", tr, impl, NoShuffle(), FailFast())
	require.NoError(t, pl.Start())
	defer func() { assert.NoError(t, pl.Stop()) }()

	pid := abstractpeer.Identify("10.0.0.1:4040")
	oldPF := addAvailablePeer(t, pl, tr, pid)

	require.NoError(t, pl.Update(peer.ListUpdates{Removals: []peer.Identifier{pid}}))
	// The transport returns the same (still Available) peer on re-add, so the
	// new facade becomes available immediately inside add().
	require.NoError(t, pl.Update(peer.ListUpdates{Additions: []peer.Identifier{pid}}))

	newPF := facadeFor(pl, pid)
	require.NotNil(t, newPF)
	require.NotSame(t, oldPF, newPF, "re-add must create a new facade")
	require.Equal(t, 1, pl.NumAvailable())
	require.Same(t, newPF, impl.mra)

	// A late notification on the old, replaced facade.
	oldPF.NotifyStatusChanged(pid)

	assert.Equal(t, 1, pl.NumAvailable(), "old facade must not be double-counted")
	assert.Same(t, newPF, impl.mra, "old facade must not replace the live one")
}

// A late notification that is not "Available" must be ignored as well.
// Otherwise it would call Implementation.Remove for a peer the implementation
// no longer holds, and decrement the available count.
func TestLateConnectingNotificationAfterRemoveIsIgnored(t *testing.T) {
	tr := newStubTransport()
	impl := &mraList{}
	pl := New("stale-notify", tr, impl, NoShuffle(), FailFast())
	require.NoError(t, pl.Start())
	defer func() { assert.NoError(t, pl.Stop()) }()

	pid := abstractpeer.Identify("10.0.0.1:4040")
	pf := addAvailablePeer(t, pl, tr, pid)
	require.NoError(t, pl.Update(peer.ListUpdates{Removals: []peer.Identifier{pid}}))
	livePF := addAvailablePeer(t, pl, tr, abstractpeer.Identify("10.0.0.2:4040"))

	// The removed peer's connection starts reconnecting, and the notification
	// for the removed facade is delivered late.
	tr.peer(pid).setStatus(peer.Connecting)
	pf.NotifyStatusChanged(pid)

	assert.Equal(t, 1, pl.NumAvailable(), "removed peer must not change the available count")
	assert.Same(t, livePF, impl.mra, "live peer must stay in the implementation")
}

// Guard-rail: notifications for live peers keep working after the fix.
func TestNotificationForLivePeerStillApplies(t *testing.T) {
	tr := newStubTransport()
	impl := &mraList{}
	pl := New("stale-notify", tr, impl, NoShuffle(), FailFast())
	require.NoError(t, pl.Start())
	defer func() { assert.NoError(t, pl.Stop()) }()

	pid := abstractpeer.Identify("10.0.0.1:4040")
	pf := addAvailablePeer(t, pl, tr, pid)

	tr.peer(pid).setStatus(peer.Unavailable)
	pf.NotifyStatusChanged(pid)
	assert.Equal(t, 0, pl.NumAvailable())
	assert.Nil(t, impl.mra)

	tr.peer(pid).setStatus(peer.Available)
	pf.NotifyStatusChanged(pid)
	assert.Equal(t, 1, pl.NumAvailable())
	assert.Same(t, pf, impl.mra)
}

// Configured peers carry an index ("host:port#n") that transports strip from
// their peers' identifiers. The list must match facades by the identifier it
// was given, so notifications for an indexed peer still apply while it is in
// the list, and stop applying once it is removed.
func TestNotificationForIndexedPeer(t *testing.T) {
	tr := newStubTransport()
	impl := &mraList{}
	pl := New("stale-notify", tr, impl, NoShuffle(), FailFast())
	require.NoError(t, pl.Start())
	defer func() { assert.NoError(t, pl.Stop()) }()

	pid := abstractpeer.Identify("10.0.0.1:4040#1")
	require.NoError(t, pl.Update(peer.ListUpdates{Additions: []peer.Identifier{pid}}))
	pf := facadeFor(pl, pid)
	require.NotNil(t, pf)
	require.Equal(t, "10.0.0.1:4040", pf.Identifier(), "transport peer must not carry the index")

	// Transports notify the facade directly rather than by identifier.
	tr.peer(pid).setStatus(peer.Available)
	pf.NotifyStatusChanged(pid)
	assert.Equal(t, 1, pl.NumAvailable(), "indexed peer must become available")
	assert.Same(t, pf, impl.mra)

	require.NoError(t, pl.Update(peer.ListUpdates{Removals: []peer.Identifier{pid}}))
	pf.NotifyStatusChanged(pid)
	assert.Equal(t, 0, pl.NumAvailable(), "removed indexed peer must not be counted as available")
	assert.Nil(t, impl.mra, "removed indexed peer must not be re-added to the implementation")
}
