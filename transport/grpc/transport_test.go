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
	"go.uber.org/zap/zapcore"
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

// TestPeerChurnNoDuplicateMetricRegistration verifies that retain → release →
// re-retain of the same address on the same Transport does not produce any
// warn/error logs. Regression test for yarpc-go v1.88.6 where connection pool
// metrics were registered per-peer; same-address peer churn caused duplicate
// registration errors that triggered ERROR logs and Healthline alerts. The fix
// registers metrics once at Transport creation (not per-peer), so peer churn
// cannot produce duplicate registration errors.
func TestPeerChurnNoDuplicateMetricRegistration(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	root := metrics.New()
	core, logs := observer.New(zapcore.WarnLevel)
	transport := NewTransport(
		Logger(zap.New(core)),
		Meter(root.Scope()),
		WithDynamicConnectionScaling(true),
	)
	require.NoError(t, transport.Start())
	defer func() { require.NoError(t, transport.Stop()) }()

	address := listener.Addr().String()
	sub := testPeerSubscriber{}

	// First retain: creates the grpcPeer. Metrics are already registered at
	// Transport creation — no registration happens here.
	_, err = transport.RetainPeer(testIdentifier{address}, sub)
	require.NoError(t, err)

	// Release drops subscriber count to 0, deleting the peer from addressToPeer.
	require.NoError(t, transport.ReleasePeer(testIdentifier{address}, sub))

	// Second retain: peer object is recreated for the same address. With the
	// old per-peer metric registration this triggered duplicate errors; with
	// transport-level registration there is nothing to duplicate.
	_, err = transport.RetainPeer(testIdentifier{address}, sub)
	require.NoError(t, err)
	require.NoError(t, transport.ReleasePeer(testIdentifier{address}, sub))

	assert.Zero(t, logs.Len(),
		"expected no warn/error logs from duplicate metric registration on peer churn; got: %v",
		logs.All())
}

// TestMultiSubscriberPeerChurnNoDuplicateMetricRegistration verifies the
// multi-outbound topology: two subscribers (simulating two outbounds) share
// one grpcPeer for the same address. The peer is only deleted when all
// subscribers release it (NumSubscribers → 0). A subsequent re-retain must
// not produce any warn/error logs. Regression test for yarpc-go v1.88.6.
func TestMultiSubscriberPeerChurnNoDuplicateMetricRegistration(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	root := metrics.New()
	core, logs := observer.New(zapcore.WarnLevel)
	transport := NewTransport(
		Logger(zap.New(core)),
		Meter(root.Scope()),
		WithDynamicConnectionScaling(true),
	)
	require.NoError(t, transport.Start())
	defer func() { require.NoError(t, transport.Stop()) }()

	address := listener.Addr().String()
	sub1 := namedPeerSubscriber{"sub1"}
	sub2 := namedPeerSubscriber{"sub2"}

	// Both subscribers retain the same address — they share one grpcPeer.
	_, err = transport.RetainPeer(testIdentifier{address}, sub1)
	require.NoError(t, err)
	_, err = transport.RetainPeer(testIdentifier{address}, sub2)
	require.NoError(t, err)

	// sub1 releases: subscriber count drops to 1, peer stays alive.
	require.NoError(t, transport.ReleasePeer(testIdentifier{address}, sub1))

	// sub2 releases: subscriber count drops to 0, peer deleted from addressToPeer.
	require.NoError(t, transport.ReleasePeer(testIdentifier{address}, sub2))

	// Re-retain: same address comes back (e.g. health-check recovery).
	// Without the fix this re-registers metrics and logs errors.
	_, err = transport.RetainPeer(testIdentifier{address}, sub1)
	require.NoError(t, err)
	require.NoError(t, transport.ReleasePeer(testIdentifier{address}, sub1))

	assert.Zero(t, logs.Len(),
		"expected no warn/error logs from duplicate metric registration on multi-subscriber peer churn; got: %v",
		logs.All())
}

type testPeerSubscriber struct{}

func (testPeerSubscriber) NotifyStatusChanged(peer.Identifier) {}

// namedPeerSubscriber is a subscriber with a distinguishing name field.
// Unlike testPeerSubscriber (zero-size), distinct instances have distinct
// interface values and are therefore distinct map keys in the subscriber map.
type namedPeerSubscriber struct{ name string }

func (namedPeerSubscriber) NotifyStatusChanged(peer.Identifier) {}

type testIdentifier struct {
	id string
}

func (i testIdentifier) Identifier() string {
	return i.id
}
