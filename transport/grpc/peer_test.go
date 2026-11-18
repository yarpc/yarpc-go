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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/integrationtest"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
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

const testConnPoolServiceName = "test-svc"

// peerForPool builds a grpcPeer with a connpool.Pool wired up for pool/
// monitor unit tests, mirroring newPeer's wiring. Start(0, false) is used so
// no connection is dialed and no scaling monitor runs automatically -- tests
// add connections explicitly via p.pool.AddConn(), which dials through
// dialTestClientConn (no real server required). meter may be nil to leave
// metrics disabled.
func peerForPool(t *testing.T, cfg connpool.Config, meter *metrics.Scope) *grpcPeer {
	t.Helper()
	return peerForPoolOnTransport(t, NewTransport(), cfg, meter)
}

// peerForPoolOnTransport is peerForPool for a caller-supplied Transport, e.g.
// one with a live pool provider. The transport must be fixed up front: the
// per-connection goroutine reads p.t, so it cannot be reassigned afterwards.
func peerForPoolOnTransport(t *testing.T, tr *Transport, cfg connpool.Config, meter *metrics.Scope) *grpcPeer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	p := &grpcPeer{
		Peer:        abstractpeer.NewPeer(abstractpeer.PeerIdentifier("127.0.0.1:1"), tr),
		t:           tr,
		ctx:         ctx,
		cancel:      cancel,
		startupPool: cfg,
	}
	p.lastValidLivePool.Store(cfg)
	dial := func(context.Context) (*grpc.ClientConn, error) {
		return dialTestClientConn(t), nil
	}
	m := connpool.NewMetrics(connpool.MetricsParams{
		Meter:       meter,
		Logger:      tr.options.logger,
		ServiceName: testConnPoolServiceName,
		Transport:   "grpc",
	})
	p.pool = connpool.NewPool(p.ctx, p.livePoolCfg, dial, tr.options.logger, p.HostPort(), connpool.NewReporter(m))
	p.pool.LogPrefix = "grpc"
	// Always wired, matching real newPeer: Pool.AddConn's connWg.Add(1) is only
	// ever balanced by the goroutine OnConnAdded runs, so any test that calls
	// p.pool.AddConn() needs this set or the pool can never finish tearing down.
	p.pool.OnConnAdded = p.monitorConnWrapper
	require.NoError(t, p.pool.Start(0, false))
	t.Cleanup(func() {
		p.cancel()
		p.pool.Wait()
	})
	return p
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

// assertConnPoolMetricTags checks every conn_pool_* series on root carries the
// component/service/transport tags the grpc transport has always used.
func assertConnPoolMetricTags(t *testing.T, root *metrics.Root, serviceName string) {
	t.Helper()
	want := map[string]string{
		"component": "yarpc",
		"service":   serviceName,
		"transport": "grpc",
	}
	samples := scrapePoolMetrics(t, root)
	require.NotEmpty(t, samples)
	for _, s := range samples {
		assert.Equal(t, want, s.labels, "series %s", s.name)
	}
}

// --- newPeer wiring ---
//
// The scaling algorithm itself is exhaustively covered by
// transport/internal/connpool's own test suite. The tests below cover only
// what is specific to grpc: that newPeer feeds the pool the right
// configuration, initial connection count and logging identity.

func startTestPeer(t *testing.T, tr *Transport) *grpcPeer {
	t.Helper()
	p, err := tr.newPeer("127.0.0.1:1", emptyDialOpts)
	require.NoError(t, err)
	t.Cleanup(func() {
		p.stop()
		p.wait()
	})
	return p
}

func TestNewPeerWiresEveryConfigFieldToPool(t *testing.T) {
	tr := NewTransport(
		WithDynamicConnectionScaling(true),
		MaxConcurrentStreams(123),
		ScaleUpThreshold(0.6),
		ScaleDownGap(0.2),
		MinConnections(2),
		MaxConnections(7),
		ConnIdleTimeout(3*time.Minute),
		ScalingMonitorInterval(45*time.Second),
	)
	p := startTestPeer(t, tr)

	want := connpool.Config{
		DynamicScalingEnabled:  true,
		MaxConcurrentStreams:   123,
		ScaleUpThreshold:       0.6,
		ScaleDownGap:           0.2,
		MinConnections:         2,
		MaxConnections:         7,
		IdleTimeout:            3 * time.Minute,
		ScalingMonitorInterval: 45 * time.Second,
	}
	assert.Equal(t, want, p.startupPool, "startup snapshot")
	assert.Equal(t, want, p.pool.Config(), "the pool must read its config through livePoolCfg")
}

func TestNewPeerPoolConfigIsLive(t *testing.T) {
	var mu sync.Mutex
	overlay := ClientConnectionPoolConfig{MaxConnections: 9}
	tr := NewTransport(
		MaxConnections(5),
		WithGlobalLiveConnectionPoolProvider(func() ClientConnectionPoolConfig {
			mu.Lock()
			defer mu.Unlock()
			return overlay
		}),
	)
	p := startTestPeer(t, tr)

	assert.Equal(t, 5, p.startupPool.MaxConnections)
	assert.Equal(t, 9, p.pool.Config().MaxConnections, "pool must see the live overlay")

	mu.Lock()
	overlay = ClientConnectionPoolConfig{MaxConnections: 11}
	mu.Unlock()
	assert.Equal(t, 11, p.pool.Config().MaxConnections, "pool must see later live changes")
}

func TestNewPeerInitialConnCount(t *testing.T) {
	tests := []struct {
		name string
		opts []TransportOption
		want int
	}{
		{"scaling off dials one", []TransportOption{WithDynamicConnectionScaling(false), MinConnections(3), MaxConnections(5)}, 1},
		{"scaling on dials minConnections", []TransportOption{WithDynamicConnectionScaling(true), MinConnections(3), MaxConnections(5)}, 3},
		{"scaling on, min equals max", []TransportOption{WithDynamicConnectionScaling(true), MinConnections(4), MaxConnections(4)}, 4},
		{"scaling on, zero min still dials one", []TransportOption{WithDynamicConnectionScaling(true), MinConnections(0), MaxConnections(5)}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := startTestPeer(t, NewTransport(tt.opts...))
			assert.Len(t, p.pool.LoadConns(), tt.want)
		})
	}
}

// TestPoolLogsKeepGRPCPrefixAndPeerField guards the observable log contract
// the grpc scaler has always had: message text prefixed "grpc: " and the
// address under the "peer" field. The pool is configured to emit these via
// LogPrefix and ID in newPeer.
func TestPoolLogsKeepGRPCPrefixAndPeerField(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	tr := NewTransport(
		Logger(zap.New(core)),
		WithDynamicConnectionScaling(true),
		MinConnections(1),
		MaxConnections(5),
	)
	p := startTestPeer(t, tr)

	_, err := p.pool.AddConn()
	require.NoError(t, err)
	_, err = p.pool.AddConn()
	require.NoError(t, err)
	logs.TakeAll()

	// 3 active connections, 0 streams: the pool can drain one.
	p.pool.EvaluateScaling()

	entries := logs.FilterMessage("grpc: scaling down connection pool; marked connection for draining").All()
	require.Len(t, entries, 1, "all logs: %v", logs.All())
	assert.Equal(t, zapcore.InfoLevel, entries[0].Level)
	assert.Equal(t, "127.0.0.1:1", entries[0].ContextMap()["peer"])
}

// --- pickConn / tryScaleUp delegation ---

func TestPickConnDelegatesToPool(t *testing.T) {
	p := peerForPool(t, connpool.Config{}, nil)
	assert.Nil(t, p.pickConn(), "empty pool returns nil")

	_, err := p.pool.AddConn()
	require.NoError(t, err)
	assert.Same(t, p.pool.PickConn(), p.pickConn())
}

func TestTryScaleUpDelegatesToPool(t *testing.T) {
	cfg := connpool.Config{
		DynamicScalingEnabled: true,
		MaxConcurrentStreams:  1,
		ScaleUpThreshold:      0.5,
		MaxConnections:        2,
	}
	p := peerForPool(t, cfg, nil)
	w, err := p.pool.AddConn()
	require.NoError(t, err)
	w.IncStreamCount()

	p.tryScaleUp(w)

	require.Eventually(t, func() bool {
		return len(p.pool.LoadConns()) == 2
	}, time.Second, time.Millisecond, "tryScaleUp must delegate to pool.TryScaleUp")
}

// --- monitorConnWrapper ---

func TestMonitorConnWrapperCleanup(t *testing.T) {
	p := peerForPool(t, connpool.Config{}, nil)

	w, err := p.pool.AddConn()
	require.NoError(t, err)
	require.Len(t, p.pool.LoadConns(), 1)

	p.cancel()

	select {
	case <-w.StoppedC:
	case <-time.After(2 * time.Second):
		t.Fatal("StoppedC not closed after context cancellation")
	}

	assert.Empty(t, p.pool.LoadConns(), "wrapper should be removed from pool on cleanup")

	done := make(chan struct{})
	go func() { p.pool.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pool did not finish tearing down after monitorConnWrapper exited")
	}
}

// TestMonitorConnWrapperReconnectsIdleConn verifies the Idle -> Connect()
// nudge: a freshly created ClientConn is Idle and, absent the nudge, would
// stay Idle until the first RPC.
func TestMonitorConnWrapperReconnectsIdleConn(t *testing.T) {
	p := peerForPool(t, connpool.Config{}, nil)
	w, err := p.pool.AddConn()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return w.Conn.GetState() != connectivity.Idle
	}, 2*time.Second, 5*time.Millisecond, "monitorConnWrapper must call Connect() on an idle connection")
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
	t.Run("empty pool sets unavailable", func(t *testing.T) {
		p := peerForPool(t, connpool.Config{}, nil)
		p.recomputeConnectionStatus()
		assert.Equal(t, peer.Unavailable, p.Peer.Status().ConnectionStatus)
	})

	t.Run("all non-active conns sets unavailable", func(t *testing.T) {
		p := peerForPool(t, connpool.Config{}, nil)
		w1, err := p.pool.AddConn()
		require.NoError(t, err)
		require.True(t, w1.TransitionState(connpool.StateActive, connpool.StateDraining))

		w2, err := p.pool.AddConn()
		require.NoError(t, err)
		require.True(t, w2.TransitionState(connpool.StateActive, connpool.StateDraining))
		require.True(t, w2.TransitionState(connpool.StateDraining, connpool.StateIdle))

		p.recomputeConnectionStatus()
		assert.Equal(t, peer.Unavailable, p.Peer.Status().ConnectionStatus)
	})

	t.Run("active conn in non-ready state sets unavailable or connecting", func(t *testing.T) {
		p := peerForPool(t, connpool.Config{}, nil)
		_, err := p.pool.AddConn()
		require.NoError(t, err)

		p.recomputeConnectionStatus()

		// A freshly created ClientConn is Idle; grpcStatusToYARPCStatus(Idle)
		// = Unavailable.  Either way it must not be Available since no server
		// is listening.
		status := p.Peer.Status().ConnectionStatus
		assert.NotEqual(t, peer.Available, status,
			"no server is listening so the connection cannot be Ready")
	})

	t.Run("monitorConnWrapper defer sets unavailable after removal", func(t *testing.T) {
		p := peerForPool(t, connpool.Config{}, nil)

		w, err := p.pool.AddConn()
		require.NoError(t, err)

		p.cancel()

		select {
		case <-w.StoppedC:
		case <-time.After(2 * time.Second):
			t.Fatal("monitorConnWrapper did not exit")
		}

		// After the connection is removed, recomputeConnectionStatus sets
		// the peer to Unavailable.
		assert.Equal(t, peer.Unavailable, p.Peer.Status().ConnectionStatus)
	})
}

// TestMonitorConnWrapperMetrics verifies that the monitorConnWrapper defer
// calls pool.RefreshMetrics so gauges drop to zero after the connection exits.
func TestMonitorConnWrapperMetrics(t *testing.T) {
	root := metrics.New()
	p := peerForPool(t, connpool.Config{}, root.Scope())

	w, err := p.pool.AddConn()
	require.NoError(t, err)

	// AddConn already refreshed the gauges for this connection, and already
	// started the monitorConnWrapper goroutine (OnConnAdded) for it.
	g, _ := poolMetricValues(t, root)
	assert.Equal(t, int64(1), g["conn_pool_active_connections"])

	p.cancel()

	select {
	case <-w.StoppedC:
	case <-time.After(2 * time.Second):
		t.Fatal("monitorConnWrapper did not exit")
	}

	// After removal the gauge must drop to zero.
	g, _ = poolMetricValues(t, root)
	assert.Equal(t, int64(0), g["conn_pool_active_connections"])
	assertConnPoolMetricTags(t, root, testConnPoolServiceName)
}
