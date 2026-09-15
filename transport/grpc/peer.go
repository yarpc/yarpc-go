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
	"sync/atomic"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/abstractpeer"
	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type grpcPeer struct {
	*abstractpeer.Peer

	t *Transport

	// ctx/cancel are the peer's own lifetime context. It is the parent of the
	// pool's context, so cancelling it cascades to the pool and, transitively,
	// to every connection wrapper -- the same single context tree this peer has
	// always had. It is also read directly by monitorConnWrapper to tell "the
	// whole peer is shutting down" apart from "just this connection was
	// evicted" (see the abstractlist.stop() deadlock note there).
	ctx    context.Context
	cancel context.CancelFunc

	// pool owns this peer's gRPC connections: dialing, dynamic scaling, idle
	// cleanup, and the pool metrics/logging for those events. The per-connection
	// health watch is plugged in through pool.OnConnAdded.
	pool *connpool.Pool[*grpc.ClientConn]

	// startupPool is the snapshot at peer creation (TransportOptions, YAML,
	// outbound overlay). livePoolCfg() overlays the live provider on top.
	startupPool          connpool.Config
	outboundLiveProvider LiveConnectionPoolProvider // per-outbound hook; nil uses the global hook
	lastValidLivePool    atomic.Value               // connpool.Config, last snapshot that passed validation
	invalidLiveWarned    atomic.Bool                // warn once per invalid live streak; skip later ticks until valid again
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

	startupPool := options.resolvedPoolConfig(t.baseConnPoolConfig())
	if err := validateResolvedConnPool(startupPool); err != nil {
		cancel()
		return nil, err
	}

	p := &grpcPeer{
		Peer:                 abstractpeer.NewPeer(abstractpeer.PeerIdentifier(address), t),
		t:                    t,
		ctx:                  ctx,
		cancel:               cancel,
		startupPool:          startupPool,
		outboundLiveProvider: options.poolConfigProvider,
	}
	p.lastValidLivePool.Store(p.startupPool)
	t.options.logger.Debug("grpc: connection pool config resolved",
		zap.String("peer", address),
		zap.Bool("dynamicScalingEnabled", p.startupPool.DynamicScalingEnabled),
		zap.Int("minConnections", p.startupPool.MinConnections),
		zap.Int("maxConnections", p.startupPool.MaxConnections),
		zap.Int32("maxConcurrentStreams", p.startupPool.MaxConcurrentStreams),
		zap.Float64("scaleUpThreshold", p.startupPool.ScaleUpThreshold),
		zap.Float64("scaleDownGap", p.startupPool.ScaleDownGap),
		zap.Duration("idleTimeout", p.startupPool.IdleTimeout),
		zap.Duration("scalingMonitorInterval", p.startupPool.ScalingMonitorInterval),
	)

	dial := func(context.Context) (*grpc.ClientConn, error) {
		//lint:ignore SA1019 grpc.Dial is deprecated
		return grpc.Dial(address, dialOptions...)
	}
	p.pool = connpool.NewPool(p.ctx, p.livePoolCfg, dial, t.options.logger, p.HostPort(), connpool.NewReporter(t.metrics))
	p.pool.LogPrefix = "grpc"
	p.pool.OnConnAdded = p.monitorConnWrapper

	// All connections are created via the pool's AddConn -- no special primary
	// connection. ScaleDownFloor is 1 when scaling is off and min capped by max
	// when on, so an invalid MinConnections > MaxConnections cannot over-dial.
	initialConnCount := connpool.ScaleDownFloor(p.startupPool)

	// Start the monitor when scaling is on at create, or when a live hook may
	// enable it later (and to wind extras down if live turns scaling off).
	startMonitor := p.startupPool.DynamicScalingEnabled || p.liveProvider() != nil

	if err := p.pool.Start(initialConnCount, startMonitor); err != nil {
		t.options.logger.Warn("grpc: failed to create peer; initial connection fill failed",
			zap.String("peer", address),
			zap.Int("initialConnCount", initialConnCount),
			zap.Error(err),
		)
		p.cancel()
		return nil, err
	}

	t.options.logger.Info("grpc: peer created",
		zap.String("peer", address),
		zap.Int("initialConnCount", initialConnCount),
	)

	return p, nil
}

// pickConn returns the active connection in the pool with the lowest current
// stream count. Returns nil if the pool contains no active connections.
func (p *grpcPeer) pickConn() *connpool.Wrapper[*grpc.ClientConn] {
	return p.pool.PickConn()
}

// tryScaleUp triggers a background goroutine to satisfy the need for more
// connection capacity if leastLoadedConn -- the connection with the fewest
// active streams, as selected by pickConn -- is over the scale-up threshold.
func (p *grpcPeer) tryScaleUp(leastLoadedConn *connpool.Wrapper[*grpc.ClientConn]) {
	p.pool.TryScaleUp(leastLoadedConn)
}

// monitorConnWrapper is the pool's OnConnAdded callback, which the pool runs in
// its own goroutine. It runs the per-connection health loop: it watches gRPC
// connectivity state changes, keeps the peer status up to date, and triggers
// reconnection when a connection goes idle. It cleans up when the wrapper's
// context is cancelled (i.e. when the peer is stopped or the connection is
// evicted by the pool), and per the OnConnAdded contract must call Remove and
// then ConnDone exactly once.
func (p *grpcPeer) monitorConnWrapper(w *connpool.Wrapper[*grpc.ClientConn]) {
	addr := p.Peer.Identifier()
	p.t.options.logger.Debug("grpc: connection watcher started", zap.String("peer", addr))
	defer func() {
		p.t.options.logger.Debug("grpc: connection watcher cleanup starting", zap.String("peer", addr))
		_ = w.Conn.Close()
		p.pool.Remove(w)
		// Skip status notification during peer shutdown: NotifyStatusChanged
		// acquires list.lock, but the caller (abstractlist.stop) already holds
		// it while waiting on p.wait() — deadlock. Metrics are safe to update.
		if p.ctx.Err() == nil {
			p.recomputeConnectionStatus()
		}
		p.pool.RefreshMetrics()
		// Close StoppedC after metrics are updated so that any goroutine
		// waiting on StoppedC (e.g. tests) observes a consistent metric state.
		close(w.StoppedC)
		p.pool.ConnDone()
		p.t.options.logger.Debug("grpc: connection watcher cleanup complete", zap.String("peer", addr))
	}()

	var grpcStatus connectivity.State
	for {
		grpcStatus = w.Conn.GetState()

		// When a connection falls back to IDLE, no automatic reconnection
		// happens. There are two options:
		// - Let the next outgoing call trigger reconnection, but this may
		//   lead to a failed request if the host is unreachable or a context
		//   deadline occurs before the connection is re-established.
		// - Reconnect manually so the connection is ready before the next call.
		// We choose the second option.
		if grpcStatus == connectivity.Idle {
			p.t.options.logger.Debug("grpc: connection idle; triggering reconnect", zap.String("peer", addr))
			w.Conn.Connect()
		}

		// If this connection is Ready the peer is Available regardless of
		// other connections — no need to scan the pool.  For any other state
		// (Connecting, TransientFailure, etc.) we must check all connections
		// to derive the correct aggregate status.
		// Skip when shutting down: NotifyStatusChanged acquires list.lock,
		// but the caller (abstractlist.stop) may already hold it while
		// waiting on p.wait().
		if p.ctx.Err() != nil {
			p.t.options.logger.Debug("grpc: connection watcher stopping; peer shutting down",
				zap.String("peer", addr), zap.String("grpcState", grpcStatus.String()))
			break
		}
		p.recomputeConnectionStatus()

		if !w.Conn.WaitForStateChange(w.Context(), grpcStatus) {
			p.t.options.logger.Debug("grpc: connection watcher stopping; connection context done",
				zap.String("peer", addr), zap.String("grpcState", grpcStatus.String()))
			break
		}
		p.t.options.logger.Debug("grpc: connection state changed",
			zap.String("peer", addr),
			zap.String("previousState", grpcStatus.String()),
			zap.String("newState", w.Conn.GetState().String()))
	}
}

// recomputeConnectionStatus derives the YARPC peer status from the aggregate
// gRPC connectivity state across all active pool connections and publishes it.
// The peer is Available if any connection is Ready, Connecting if any is
// connecting (and none are Ready), and Unavailable if the pool is empty or
// all connections are in a terminal/unknown state.
func (p *grpcPeer) recomputeConnectionStatus() {
	conns := p.pool.LoadConns()
	best := peer.Unavailable
	for _, c := range conns {
		if !c.IsActive() {
			continue
		}
		s := grpcStatusToYARPCStatus(c.Conn.GetState())
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
	p.pool.Wait()
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
