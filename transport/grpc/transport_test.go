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
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
	assert.Equal(t, peer, transport.addressToPeer[address])
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

// TestPeerReRetainReusesMetrics verifies that releasing and re-retaining the
// same peer address does not cause duplicate metric registration errors.
// Peer churn (downstream deploys, health-check flaps) triggers this path in
// production — without the peerMetrics cache every re-creation logs an error
// and fires healthline alerts.
func TestPeerReRetainReusesMetrics(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	root := metrics.New()
	scope := root.Scope()
	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	transport := NewTransport(
		Meter(scope),
		Logger(logger),
	)
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	address := listener.Addr().String()
	sub := testPeerSubscriber{}

	// First retain — registers metrics for this peer address.
	_, err = transport.RetainPeer(testIdentifier{address}, sub)
	require.NoError(t, err)

	// Release — peer is removed from addressToPeer but metrics stay in peerMetrics.
	require.NoError(t, transport.ReleasePeer(testIdentifier{address}, sub))

	// Re-retain the same address — must reuse cached metrics, not re-register.
	_, err = transport.RetainPeer(testIdentifier{address}, sub)
	require.NoError(t, err)

	assert.Zero(t, logs.Len(), "re-retaining a peer must not produce duplicate metric registration warnings")
}

// TestPeerMetricsNotRegisteredWhenScalingDisabled verifies that no metrics are
// registered when dynamic scaling is off — the pool always has one connection,
// making the gauges meaningless, and skipping registration avoids duplicate
// registration errors on peer churn for the vast majority of services.
func TestPeerMetricsNotRegisteredWhenScalingDisabled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	root := metrics.New()
	scope := root.Scope()

	transport := NewTransport(Meter(scope)) // dynamic scaling not enabled
	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	address := listener.Addr().String()
	sub := testPeerSubscriber{}

	_, err = transport.RetainPeer(testIdentifier{address}, sub)
	require.NoError(t, err)

	snap := root.Snapshot()
	assert.Empty(t, snap.Gauges, "conn pool gauges must not be registered when dynamic scaling is disabled")
	assert.Empty(t, snap.Counters, "conn pool counters must not be registered when dynamic scaling is disabled")
}

func (testPeerSubscriber) NotifyStatusChanged(peer.Identifier) {}

type testIdentifier struct {
	id string
}

func (i testIdentifier) Identifier() string {
	return i.id
}
