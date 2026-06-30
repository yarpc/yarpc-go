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

package grpc

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"google.golang.org/grpc"
)

func TestTransportLifecycle(t *testing.T) {
	transport := NewTransport()
	assert.NoError(t, transport.Start())
	assert.True(t, transport.IsRunning())
	assert.NoError(t, transport.Stop())
	assert.False(t, transport.IsRunning())
}

func TestRetainReleasePeerSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	transport := NewTransport()
	assert.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	address := listener.Addr().String()
	peerSubscriber := testPeerSubscriber{}

	peer, err := transport.RetainPeer(testIdentifier{address}, peerSubscriber)
	assert.NoError(t, err)
	assert.Equal(t, peer, transport.peers[peerKey{address: address}])
	assert.NoError(t, transport.ReleasePeer(testIdentifier{address}, peerSubscriber))
}

func TestRetainReleasePeerErrorPeerIdentifier(t *testing.T) {
	transport := NewTransport()
	assert.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()
}

func TestReleasePeerErrorNoPeer(t *testing.T) {
	transport := NewTransport()
	assert.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	address := "not_retained"
	peerSubscriber := testPeerSubscriber{}

	assert.Equal(t, peer.ErrTransportHasNoReferenceToPeer{
		TransportName:  "grpc.Transport",
		PeerIdentifier: address,
	}, transport.ReleasePeer(testIdentifier{address}, peerSubscriber))
}

// TestStopWaitsForReleasedPeerCleanup verifies that Transport.Stop() blocks
// until the async cleanup goroutine launched by ReleasePeer finishes.
func TestStopWaitsForReleasedPeerCleanup(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	transport := NewTransport()
	require.NoError(t, transport.Start())

	address := listener.Addr().String()
	sub := testPeerSubscriber{}

	_, err = transport.RetainPeer(testIdentifier{address}, sub)
	require.NoError(t, err)

	// ReleasePeer with 0 remaining subscribers spawns an async cleanup goroutine.
	require.NoError(t, transport.ReleasePeer(testIdentifier{address}, sub))

	// Stop must not return until releasedCleanupWg reaches zero.
	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, transport.Stop())
	}()

	select {
	case <-done:
		// Stop completed — cleanup goroutine finished first.
	case <-time.After(5 * time.Second):
		t.Fatal("Transport.Stop() did not return within 5s; releasedCleanupWg may not be awaited")
	}
}

// TestStopReleasesLockBeforeWait verifies that Transport.Stop() does not hold
// t.lock while calling p.wait(), so concurrent operations that need the lock
// (e.g. ReleasePeer called from a peer-list shutdown) do not deadlock.
func TestStopReleasesLockBeforeWait(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	transport := NewTransport()
	require.NoError(t, transport.Start())

	address := listener.Addr().String()
	sub := testPeerSubscriber{}

	_, err = transport.RetainPeer(testIdentifier{address}, sub)
	require.NoError(t, err)

	// Simulate a concurrent ReleasePeer that runs while Stop is in p.wait().
	// If Stop held t.lock during p.wait() this would deadlock.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Small delay to let Stop enter p.wait().
		time.Sleep(10 * time.Millisecond)
		// This acquires t.lock internally; must not deadlock.
		_ = transport.ReleasePeer(testIdentifier{address}, sub)
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, transport.Stop())
	}()

	wg.Wait()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Transport.Stop() deadlocked while holding t.lock during p.wait()")
	}
}

type testPeerSubscriber struct{}

func (testPeerSubscriber) NotifyStatusChanged(peer.Identifier) {}

// idSubscriber is a comparable subscriber with a distinct identity, used to
// model distinct outbounds in tests.
type idSubscriber struct{ id int }

func (idSubscriber) NotifyStatusChanged(peer.Identifier) {}

type testIdentifier struct {
	id string
}

func (i testIdentifier) Identifier() string {
	return i.id
}

// countingListener counts accepted connections and signals each on acceptedC.
type countingListener struct {
	net.Listener
	acceptedC chan struct{}
}

func (l *countingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil {
		l.acceptedC <- struct{}{}
	}
	return c, err
}

func waitAccepts(t *testing.T, ch <-chan struct{}, n int) {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for range n {
		select {
		case <-ch:
		case <-timeout:
			t.Fatalf("timed out waiting for %d connection(s)", n)
		}
	}
}

// TestRetainPeerConnectionPerOutbound verifies that, with ConnectionPerOutbound,
// distinct outbounds (subscribers) dialing the same address each get their own
// peer and connection, while reusing the same subscriber dedups.
func TestRetainPeerConnectionPerOutbound(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	cl := &countingListener{Listener: listener, acceptedC: make(chan struct{}, 8)}

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(cl)
	defer grpcServer.Stop()

	address := listener.Addr().String()

	transport := NewTransport(ConnectionPerOutbound())
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{address}
	// Distinct subscribers model distinct outbounds. Use a type with identity:
	// pointers to the zero-size testPeerSubscriber can compare equal.
	sub1, sub2 := idSubscriber{1}, idSubscriber{2}

	p1, err := transport.RetainPeer(id, sub1)
	require.NoError(t, err)
	p2, err := transport.RetainPeer(id, sub2)
	require.NoError(t, err)

	// Two outbounds to the same address get two peers and two connections.
	assert.NotSame(t, p1, p2)
	assert.Len(t, transport.peers, 2)
	waitAccepts(t, cl.acceptedC, 2)

	// Reusing the same subscriber dedups: no new peer, no new connection.
	p1Again, err := transport.RetainPeer(id, sub1)
	require.NoError(t, err)
	assert.Same(t, p1, p1Again)
	assert.Len(t, transport.peers, 2)

	// Releasing one outbound leaves the other's connection intact.
	require.NoError(t, transport.ReleasePeer(id, sub1))
	assert.Len(t, transport.peers, 1)
}

// TestRetainPeerSharedByDefault verifies that without ConnectionPerOutbound,
// outbounds dialing the same address share a single peer and connection.
func TestRetainPeerSharedByDefault(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	address := listener.Addr().String()

	transport := NewTransport()
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	id := testIdentifier{address}
	sub1, sub2 := idSubscriber{1}, idSubscriber{2}

	p1, err := transport.RetainPeer(id, sub1)
	require.NoError(t, err)
	p2, err := transport.RetainPeer(id, sub2)
	require.NoError(t, err)

	assert.Same(t, p1, p2)
	assert.Len(t, transport.peers, 1)
}
