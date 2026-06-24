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
	"runtime"
	"sync"
	"sync/atomic"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type grpcPeer struct {
	*abstractpeer.Peer

	t        *Transport
	ctx      context.Context
	cancel   context.CancelFunc
	stoppedC chan struct{}

	// grpcDialOpts are stored so that additional pooled connections can be
	// dialled with the same parameters as the initial connection.
	grpcDialOpts []grpc.DialOption

	// isScaling is set to 1 (via CAS) while a background scale-up goroutine is
	// running.  This prevents multiple concurrent scale-up operations.
	isScaling int32 // accessed atomically

	// connWg tracks active monitorConnWrapper goroutines.
	// stoppedC is closed once all goroutines finish after the peer is stopped.
	connWg sync.WaitGroup

	// metrics reports this peer's connection-state counts and scaling events
	// into the transport-wide shared metrics (which are not tagged by peer).
	metrics *peerPoolReporter

	// connsPtr holds a pointer to the current immutable connection slice.
	// Readers (pickConn, recomputeConnectionStatus, refreshPoolMetrics, etc.)
	// do a single atomic.Pointer.Load() — no lock acquired on the read path.
	// Writers (addConn, removeConn) use a CAS loop: load the current pointer,
	// build the new slice, and CompareAndSwap. connCount is updated after a
	// successful CAS.
	connsPtr atomic.Pointer[[](*grpcClientConnWrapper)]

	// addingCount tracks addConn calls that have passed their entry point but
	// have not yet called connWg.Add(1) or bailed. The lifecycle goroutine
	// spins on this counter (after setting shutdownStarted) to guarantee that
	// connWg.Wait is not called before any racing connWg.Add(1) completes.
	addingCount atomic.Int32

	// shutdownStarted is set to true when the peer begins shutting down.
	// addConn checks this after incrementing addingCount to avoid calling
	// connWg.Add(1) after the lifecycle goroutine has passed its spin.
	shutdownStarted atomic.Bool

	// connCount mirrors len(loadConns()) atomically so tryScaleUp can check the
	// pool size without a full slice load.
	connCount atomic.Int32

	poolCfg connPoolConfig
}

// loadConns returns the current immutable connection snapshot.
// Safe to call from any goroutine without holding any lock.
func (p *grpcPeer) loadConns() []*grpcClientConnWrapper {
	ptr := p.connsPtr.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}

// storeConns atomically publishes a new connection slice and updates connCount.
func (p *grpcPeer) storeConns(conns []*grpcClientConnWrapper) {
	p.connsPtr.Store(&conns)
	p.connCount.Store(int32(len(conns)))
}

func (t *Transport) newPeer(address string, options *dialOptions) (*grpcPeer, error) {
	dialOptions := append([]grpc.DialOption{
		grpc.WithUserAgent(UserAgent),
		grpc.WithDefaultCallOptions(
			grpc.ForceCodecV2(customCodec{}),
			grpc.MaxCallRecvMsgSize(t.options.clientMaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(t.options.clientMaxSendMsgSize),
		),
	}, options.grpcOptions(t)...)

	if t.options.clientMaxHeaderListSize != nil {
		dialOptions = append(dialOptions, grpc.WithMaxHeaderListSize(*t.options.clientMaxHeaderListSize))
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &grpcPeer{
		Peer:         abstractpeer.NewPeer(abstractpeer.PeerIdentifier(address), t),
		t:            t,
		ctx:          ctx,
		cancel:       cancel,
		stoppedC:     make(chan struct{}),
		grpcDialOpts: dialOptions,
		metrics:      newPeerPoolReporter(t.metrics),
		poolCfg: connPoolConfig{
			dynamicScalingEnabled:  t.options.clientConnPoolDynamicScalingEnabled,
			maxConcurrentStreams:   t.options.clientConnPoolMaxConcurrentStreams,
			scaleUpThreshold:       t.options.clientConnPoolScaleUpThreshold,
			scaleDownGap:           t.options.clientConnPoolScaleDownGap,
			minConnections:         t.options.clientConnPoolMinConnections,
			maxConnections:         t.options.clientConnPoolMaxConnections,
			idleTimeout:            t.options.clientConnPoolIdleTimeout,
			scalingMonitorInterval: t.options.clientConnPoolScalingMonitorInterval,
		},
	}
	t.options.logger.Debug("grpc: connection pool config resolved",
		zap.String("peer", address),
		zap.Bool("dynamicScalingEnabled", p.poolCfg.dynamicScalingEnabled),
		zap.Int("minConnections", p.poolCfg.minConnections),
		zap.Int("maxConnections", p.poolCfg.maxConnections),
		zap.Int32("maxConcurrentStreams", p.poolCfg.maxConcurrentStreams),
		zap.Float64("scaleUpThreshold", p.poolCfg.scaleUpThreshold),
		zap.Float64("scaleDownGap", p.poolCfg.scaleDownGap),
		zap.Duration("idleTimeout", p.poolCfg.idleTimeout),
		zap.Duration("scalingMonitorInterval", p.poolCfg.scalingMonitorInterval),
	)
	// Publish an empty slice so loadConns() never returns nil before the first
	// addConn() call.
	p.storeConns(nil)

	// All connections are created via addConn — no special primary connection.
	initialConnCount := 1
	if p.poolCfg.dynamicScalingEnabled {
		initialConnCount = p.poolCfg.minConnections
	}
	for i := 0; i < initialConnCount; i++ {
		if err := p.addConn(); err != nil {
			p.cancel()
			return nil, err
		}
	}

	if p.poolCfg.dynamicScalingEnabled {
		go p.runScalingMonitor()
	}

	// Close stoppedC once all monitorConnWrapper goroutines have finished.
	// shutdownStarted gates new connWg.Add(1) calls; addingCount ensures we
	// wait for any addConn already past its ctx check before calling Wait.
	go func() {
		<-p.ctx.Done()
		p.shutdownStarted.Store(true)
		for p.addingCount.Load() > 0 {
			runtime.Gosched()
		}
		p.connWg.Wait()
		close(p.stoppedC)
	}()

	return p, nil
}

// addConn dials a new connection to the peer's address using the same options
// as the initial connection, appends the wrapper to the pool, and starts its
// monitor goroutine.
func (p *grpcPeer) addConn() error {
	//lint:ignore SA1019 grpc.Dial is deprecated
	clientConn, err := grpc.Dial(p.Peer.Identifier(), p.grpcDialOpts...)
	if err != nil {
		return err
	}
	w := newConnWrapper(p.ctx, clientConn)

	// Enter the critical window: increment addingCount so the lifecycle
	// goroutine's spin waits for us to either bail or call connWg.Add(1).
	p.addingCount.Add(1)
	if p.ctx.Err() != nil || p.shutdownStarted.Load() {
		p.addingCount.Add(-1)
		_ = clientConn.Close()
		return p.ctx.Err()
	}
	p.connWg.Add(1)
	p.addingCount.Add(-1)

	// CAS loop: copy the slice, append the new wrapper, publish atomically.
	for {
		old := p.connsPtr.Load()
		var oldSlice []*grpcClientConnWrapper
		if old != nil {
			oldSlice = *old
		}
		next := make([]*grpcClientConnWrapper, len(oldSlice)+1)
		copy(next, oldSlice)
		next[len(oldSlice)] = w
		if p.connsPtr.CompareAndSwap(old, &next) {
			p.connCount.Store(int32(len(next)))
			break
		}
	}

	go p.monitorConnWrapper(w)
	p.refreshPoolMetrics()
	return nil
}

// monitorConnWrapper runs the per-connection health loop: it watches gRPC
// connectivity state changes, keeps the peer status up to date, and triggers
// reconnection when a connection goes idle.  It cleans up when the wrapper's
// context is cancelled (i.e. when the peer is stopped or the connection is
// evicted by the pool).
func (p *grpcPeer) monitorConnWrapper(w *grpcClientConnWrapper) {
	defer func() {
		_ = w.clientConn.Close()
		p.removeConn(w)
		// Skip status notification during peer shutdown: NotifyStatusChanged
		// acquires list.lock, but the caller (abstractlist.stop) already holds
		// it while waiting on p.wait() — deadlock. Metrics are safe to update.
		if p.ctx.Err() == nil {
			p.recomputeConnectionStatus()
		}
		p.refreshPoolMetrics()
		// Close stoppedC after metrics are updated so that any goroutine
		// waiting on stoppedC (e.g. tests) observes a consistent metric state.
		close(w.stoppedC)
		p.connWg.Done()
	}()

	var grpcStatus connectivity.State
	for {
		grpcStatus = w.clientConn.GetState()

		// When a connection falls back to IDLE, no automatic reconnection
		// happens. There are two options:
		// - Let the next outgoing call trigger reconnection, but this may
		//   lead to a failed request if the host is unreachable or a context
		//   deadline occurs before the connection is re-established.
		// - Reconnect manually so the connection is ready before the next call.
		// We choose the second option.
		if grpcStatus == connectivity.Idle {
			w.clientConn.Connect()
		}

		// If this connection is Ready the peer is Available regardless of
		// other connections — no need to scan the pool.  For any other state
		// (Connecting, TransientFailure, etc.) we must check all connections
		// to derive the correct aggregate status.
		// Skip when shutting down: NotifyStatusChanged acquires list.lock,
		// but the caller (abstractlist.stop) may already hold it while
		// waiting on p.wait().
		if p.ctx.Err() != nil {
			break
		}
		p.recomputeConnectionStatus()

		if !w.clientConn.WaitForStateChange(w.ctx, grpcStatus) {
			break
		}
	}
}

// recomputeConnectionStatus derives the YARPC peer status from the aggregate
// gRPC connectivity state across all active pool connections and publishes it.
// The peer is Available if any connection is Ready, Connecting if any is
// connecting (and none are Ready), and Unavailable if the pool is empty or
// all connections are in a terminal/unknown state.
func (p *grpcPeer) recomputeConnectionStatus() {
	conns := p.loadConns()
	best := peer.Unavailable
	for _, c := range conns {
		if !c.isActive() {
			continue
		}
		s := grpcStatusToYARPCStatus(c.clientConn.GetState())
		if s == peer.Available {
			best = peer.Available
			break
		}
		if s == peer.Connecting && best == peer.Unavailable {
			best = peer.Connecting
		}
	}
	p.setConnectionStatus(best)
}

// removeConn removes a wrapper from the peer's connection pool.
// Lock-free: uses a CAS loop to atomically publish the new slice.
func (p *grpcPeer) removeConn(w *grpcClientConnWrapper) {
	for {
		old := p.connsPtr.Load()
		if old == nil {
			return
		}
		oldSlice := *old
		idx := -1
		for i, c := range oldSlice {
			if c == w {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		next := make([]*grpcClientConnWrapper, len(oldSlice)-1)
		copy(next, oldSlice[:idx])
		copy(next[idx:], oldSlice[idx+1:])
		if p.connsPtr.CompareAndSwap(old, &next) {
			p.connCount.Store(int32(len(next)))
			return
		}
	}
}

// pickConn returns the active connection in the pool with the lowest current
// stream count.  Returns nil if the pool contains no active connections.
// Lock-free: reads the immutable snapshot via a single atomic.Pointer.Load().
func (p *grpcPeer) pickConn() *grpcClientConnWrapper {
	conns := p.loadConns()
	var best *grpcClientConnWrapper
	for _, c := range conns {
		if !c.isActive() {
			continue
		}
		if best == nil || c.getStreamCount() < best.getStreamCount() {
			best = c
		}
	}
	return best
}

func (p *grpcPeer) setConnectionStatus(status peer.ConnectionStatus) {
	p.t.options.logger.Debug(
		"peer status change",
		zap.String("status", status.String()),
		zap.String("peer", p.Peer.Identifier()),
		zap.String("transport", "grpc"),
	)
	p.Peer.SetStatus(status)
	p.Peer.NotifyStatusChanged()
}

// StartRequest and EndRequest are no-ops now.
// They previously aggregated pending request count from all subscibed peer
// lists and distributed change notifications.
// This was fraught with concurrency hazards so we moved pending request count
// tracking into the lists themselves.

func (p *grpcPeer) StartRequest() {}

func (p *grpcPeer) EndRequest() {}

func (p *grpcPeer) stop() {
	p.cancel()
}

func (p *grpcPeer) wait() {
	<-p.stoppedC
}

func grpcStatusToYARPCStatus(grpcStatus connectivity.State) peer.ConnectionStatus {
	switch grpcStatus {
	case connectivity.Ready:
		return peer.Available
	case connectivity.Connecting:
		return peer.Connecting
	default:
		return peer.Unavailable
	}
}
