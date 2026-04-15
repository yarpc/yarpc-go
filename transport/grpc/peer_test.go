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
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/integrationtest"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

var spec = integrationtest.TransportSpec{
	Identify: hostport.Identify,
	NewServerTransport: func(t *testing.T, addr string) peer.Transport {
		return NewTransport(BackoffStrategy(backoff.None))
	},
	NewClientTransport: func(t *testing.T) peer.Transport {
		return NewTransport(BackoffStrategy(backoff.None))
	},
	NewUnaryOutbound: func(x peer.Transport, peerChooser peer.Chooser) transport.UnaryOutbound {
		return x.(*Transport).NewOutbound(peerChooser)
	},
	NewInbound: func(t peer.Transport, address string) transport.Inbound {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			panic(err.Error())
		}
		return t.(*Transport).NewInbound(listener)
	},
	Addr: func(_ peer.Transport, inbound transport.Inbound) string {
		return yarpctest.ZeroAddrToHostPort(inbound.(*Inbound).listener.Addr())
	},
}

func TestPeerWithRoundRobin(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	permanent, permanentAddr := spec.NewServer(t, "127.0.0.1:0")
	defer permanent.Stop()

	temporary, temporaryAddr := spec.NewServer(t, "127.0.0.1:0")
	defer temporary.Stop()

	// Construct a client with a bank of peers. We will keep one running all
	// the time. We'll shut one down temporarily.
	// The round robin peer list should only choose peers that have
	// successfully connected.
	client, c := spec.NewClient(t, []string{
		permanentAddr,
		temporaryAddr,
	})
	defer client.Stop()

	integrationtest.Blast(ctx, t, c)

	// Shut down one task in the peer list.
	require.NoError(t, temporary.Stop())

	// One of these requests may fail since one of the peers has gone down but
	// the gRPC transport will not know until a request is attempted.
	integrationtest.Call(ctx, c)
	integrationtest.Call(ctx, c)
	// All subsequent should succeed since the peer should be removed on
	// connection fail.
	integrationtest.Blast(ctx, t, c)

	// Restore the server on the temporary port.
	restored, _ := spec.NewServer(t, temporaryAddr)
	defer restored.Stop()
	integrationtest.Blast(ctx, t, c)
}

func TestPeerIntegration(t *testing.T) {
	t.Skip("Skipping due to test flakiness")
	spec.Test(t)
}

func TestReconnectionCalledForIDLE(t *testing.T) {
	logger := zaptest.NewLogger(t)

	grpcTransport := NewTransport(Logger(logger))
	require.NoError(t, grpcTransport.Start())

	chooser := roundrobin.New(grpcTransport, roundrobin.Logger(logger))
	outbound := grpcTransport.NewOutbound(chooser)
	require.NoError(t, outbound.Start())

	permanent, permanentAddr := spec.NewServer(t, "127.0.0.1:0")
	defer permanent.Stop()

	temporary, temporaryAddr := spec.NewServer(t, "127.0.0.1:0")
	defer temporary.Stop()

	require.NoError(t, chooser.Update(peer.ListUpdates{
		Additions: []peer.Identifier{
			hostport.Identify(permanentAddr),
			hostport.Identify(temporaryAddr),
		},
	}))

	dispatcher := integrationtest.CreateAndStartClientDispatcher(t, outbound)
	defer dispatcher.Stop()

	rawClient := raw.New(dispatcher.ClientConfig(integrationtest.ServiceName))

	makeBlastCall(t, rawClient, 1*time.Second)

	// Shut down one service.
	require.NoError(t, temporary.Stop())

	waitForPeerStatus(t, chooser, temporaryAddr, peer.Unavailable, 2*time.Second)
	makeBlastCall(t, rawClient, 1*time.Second)

	// Restore the server on the temporary port.
	restored, _ := spec.NewServer(t, temporaryAddr)
	defer restored.Stop()

	waitForPeerStatus(t, chooser, temporaryAddr, peer.Available, 2*time.Second)
	makeBlastCall(t, rawClient, 1*time.Second)
}

func waitForPeerStatus(t *testing.T, peerList *roundrobin.List, peerAddr string, status peer.ConnectionStatus, wait time.Duration) {
	peerAvailable := make(chan struct{})
	go func() {
		for {
			for _, p := range peerList.Peers() {
				if p.Identifier() == peerAddr {
					if p.Status().ConnectionStatus == status {
						close(peerAvailable)
						return
					}
				}
			}

			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-time.After(wait):
		t.Fatal("failed waiting to connect to peer")
	case <-peerAvailable:
		return
	}
}

func makeBlastCall(t *testing.T, rawClient raw.Client, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	integrationtest.Blast(ctx, t, rawClient)
}

// --- pool unit test helpers ---

// peerForPool builds a grpcPeer with all fields needed for pool unit tests.
// It dials lazily so no real server is required.
func peerForPool(t *testing.T) *grpcPeer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	tr := NewTransport()
	return &grpcPeer{
		Peer:     abstractpeer.NewPeer(abstractpeer.PeerIdentifier("127.0.0.1:1"), tr),
		t:        tr,
		ctx:      ctx,
		cancel:   cancel,
		stoppedC: make(chan struct{}),
		grpcDialOpts: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		poolCfg: connPoolConfig{
			dynamicScalingEnabled: true,
			maxConcurrentStreams:   100,
			scaleUpThreshold:      0.8, // threshold = 80
			minConnections:        1,
			maxConnections:        5,
		},
	}
}

// dialTestClientConn returns a real *grpc.ClientConn to a passthrough address.
// No actual network connection is made; the conn is registered for cleanup.
func dialTestClientConn(t *testing.T) *grpc.ClientConn {
	t.Helper()
	cc, err := grpc.NewClient(
		"passthrough:///localhost:0",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cc.Close() })
	return cc
}

// --- pickConn ---

func TestPickConn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		conns   []*grpcClientConnWrapper
		wantNil bool
		wantIdx int // index into conns of the expected winner (ignored when wantNil)
	}{
		{
			desc:    "empty pool returns nil",
			conns:   nil,
			wantNil: true,
		},
		{
			desc: "all non-active returns nil",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateDraining, 5),
				makeConn(connStateIdle, 0),
			},
			wantNil: true,
		},
		{
			desc: "single active conn is returned",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 10),
			},
			wantIdx: 0,
		},
		{
			// Lowest is at index 1.
			desc: "picks conn with lowest stream count",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateActive, 30),
				makeConn(connStateActive, 5),
				makeConn(connStateActive, 20),
			},
			wantIdx: 1,
		},
		{
			// Draining conn is excluded; lowest active is at index 2.
			desc: "skips non-active conns when picking lowest",
			conns: []*grpcClientConnWrapper{
				makeConn(connStateDraining, 1),
				makeConn(connStateActive, 20),
				makeConn(connStateActive, 10),
			},
			wantIdx: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			p := peerForScaleDown(t, tt.conns, connPoolConfig{})
			got := p.pickConn()
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Same(t, tt.conns[tt.wantIdx], got)
		})
	}
}

// --- removeConn ---

func TestRemoveConn(t *testing.T) {
	t.Parallel()

	t.Run("removes from beginning", func(t *testing.T) {
		t.Parallel()
		c0, c1, c2 := makeConn(connStateActive, 0), makeConn(connStateActive, 0), makeConn(connStateActive, 0)
		p := peerForScaleDown(t, []*grpcClientConnWrapper{c0, c1, c2}, connPoolConfig{})
		p.removeConn(c0)
		require.Len(t, p.conns, 2)
		assert.Same(t, c1, p.conns[0])
		assert.Same(t, c2, p.conns[1])
	})

	t.Run("removes from middle", func(t *testing.T) {
		t.Parallel()
		c0, c1, c2 := makeConn(connStateActive, 0), makeConn(connStateActive, 0), makeConn(connStateActive, 0)
		p := peerForScaleDown(t, []*grpcClientConnWrapper{c0, c1, c2}, connPoolConfig{})
		p.removeConn(c1)
		require.Len(t, p.conns, 2)
		assert.Same(t, c0, p.conns[0])
		assert.Same(t, c2, p.conns[1])
	})

	t.Run("removes from end", func(t *testing.T) {
		t.Parallel()
		c0, c1, c2 := makeConn(connStateActive, 0), makeConn(connStateActive, 0), makeConn(connStateActive, 0)
		p := peerForScaleDown(t, []*grpcClientConnWrapper{c0, c1, c2}, connPoolConfig{})
		p.removeConn(c2)
		require.Len(t, p.conns, 2)
		assert.Same(t, c0, p.conns[0])
		assert.Same(t, c1, p.conns[1])
	})

	t.Run("no-op when conn not in pool", func(t *testing.T) {
		t.Parallel()
		c0, c1 := makeConn(connStateActive, 0), makeConn(connStateActive, 0)
		notInPool := makeConn(connStateActive, 0)
		p := peerForScaleDown(t, []*grpcClientConnWrapper{c0, c1}, connPoolConfig{})
		assert.NotPanics(t, func() { p.removeConn(notInPool) })
		assert.Len(t, p.conns, 2)
	})
}

// --- monitorConnWrapper ---

func TestMonitorConnWrapperCleanup(t *testing.T) {
	t.Parallel()

	p := peerForPool(t)
	cc := dialTestClientConn(t)
	w := newConnWrapper(p.ctx, cc)

	p.connWg.Add(1)
	p.mu.Lock()
	p.conns = append(p.conns, w)
	p.mu.Unlock()

	go p.monitorConnWrapper(w)

	p.mu.RLock()
	require.Len(t, p.conns, 1)
	p.mu.RUnlock()

	p.cancel()

	select {
	case <-w.stoppedC:
	case <-time.After(2 * time.Second):
		t.Fatal("stoppedC not closed after context cancellation")
	}

	p.mu.RLock()
	assert.Empty(t, p.conns, "wrapper should be removed from pool on cleanup")
	p.mu.RUnlock()

	wgDone := make(chan struct{})
	go func() { p.connWg.Wait(); close(wgDone) }()
	select {
	case <-wgDone:
	case <-time.After(2 * time.Second):
		t.Fatal("connWg did not reach zero after monitorConnWrapper exited")
	}
}

// --- stoppedC lifecycle ---

func TestStoppedCClosedAfterAllConnsFinish(t *testing.T) {
	t.Parallel()

	p := peerForPool(t)

	go func() {
		<-p.ctx.Done()
		p.mu.Lock()
		p.mu.Unlock() //nolint:staticcheck
		p.connWg.Wait()
		close(p.stoppedC)
	}()

	for i := 0; i < 2; i++ {
		cc := dialTestClientConn(t)
		w := newConnWrapper(p.ctx, cc)
		p.connWg.Add(1)
		p.mu.Lock()
		p.conns = append(p.conns, w)
		p.mu.Unlock()
		go p.monitorConnWrapper(w)
	}

	p.cancel()

	select {
	case <-p.stoppedC:
	case <-time.After(2 * time.Second):
		t.Fatal("stoppedC not closed after all connections finished")
	}
}

// --- addConn context-cancelled branch ---

func TestAddConnContextCancelled(t *testing.T) {
	t.Parallel()

	p := peerForPool(t)
	p.cancel()

	err := p.addConn()

	require.Error(t, err)
	assert.Empty(t, p.conns, "cancelled addConn must not add to pool")
}

// --- grpcStatusToYARPCStatus ---

func TestGrpcStatusToYARPCStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		grpcStatus connectivity.State
		wantYARPC  peer.ConnectionStatus
	}{
		{connectivity.Ready, peer.Available},
		{connectivity.Connecting, peer.Connecting},
		{connectivity.Idle, peer.Unavailable},
		{connectivity.TransientFailure, peer.Unavailable},
		{connectivity.Shutdown, peer.Unavailable},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.wantYARPC, grpcStatusToYARPCStatus(tt.grpcStatus), "grpcStatus=%v", tt.grpcStatus)
	}
}

// --- recomputeConnectionStatus ---

func TestRecomputeConnectionStatus(t *testing.T) {
	t.Parallel()

	t.Run("empty pool sets unavailable", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)
		p.recomputeConnectionStatus()
		assert.Equal(t, peer.Unavailable, p.Peer.Status().ConnectionStatus)
	})

	t.Run("all non-active conns sets unavailable", func(t *testing.T) {
		t.Parallel()
		p := peerForScaleDown(t, []*grpcClientConnWrapper{
			makeConn(connStateDraining, 5),
			makeConn(connStateIdle, 0),
		}, connPoolConfig{})
		p.recomputeConnectionStatus()
		assert.Equal(t, peer.Unavailable, p.Peer.Status().ConnectionStatus)
	})

	t.Run("active conn in non-ready state sets unavailable or connecting", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)
		cc := dialTestClientConn(t)
		w := newConnWrapper(p.ctx, cc)
		p.mu.Lock()
		p.conns = append(p.conns, w)
		p.mu.Unlock()

		p.recomputeConnectionStatus()

		// A freshly created ClientConn is Idle; grpcStatusToYARPCStatus(Idle)
		// = Unavailable.  Either way it must not be Available since no server
		// is listening.
		status := p.Peer.Status().ConnectionStatus
		assert.NotEqual(t, peer.Available, status,
			"no server is listening so the connection cannot be Ready")
	})

	t.Run("monitorConnWrapper defer sets unavailable after removal", func(t *testing.T) {
		t.Parallel()
		p := peerForPool(t)
		cc := dialTestClientConn(t)
		w := newConnWrapper(p.ctx, cc)
		p.connWg.Add(1)
		p.mu.Lock()
		p.conns = append(p.conns, w)
		p.mu.Unlock()

		go p.monitorConnWrapper(w)

		p.cancel()

		select {
		case <-w.stoppedC:
		case <-time.After(2 * time.Second):
			t.Fatal("monitorConnWrapper did not exit")
		}

		// After the connection is removed, recomputeConnectionStatus sets
		// the peer to Unavailable.
		assert.Equal(t, peer.Unavailable, p.Peer.Status().ConnectionStatus)
	})
}
