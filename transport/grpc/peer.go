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

	// ctx/cancel are the peer's own lifetime context, independent of the
	// pool's internal context. It is passed to connpool.NewPool as the
	// pool's parent context, so cancelling it cascades to cancel the pool
	// (and, transitively, every connection wrapper's context) -- the same
	// single-context-tree shutdown shape this peer used before the pool
	// existed. It is kept here (rather than relying on some equivalent
	// exposed by Pool) purely so monitorConnWrapper can cheaply tell "is
	// this whole peer shutting down" apart from "was just this one
	// connection evicted", to avoid the abstractlist.stop() deadlock
	// documented on that method.
	ctx    context.Context
	cancel context.CancelFunc

	// pool manages this peer's gRPC connection(s): dialing, dynamic
	// scaling, idle cleanup, and the aggregate metrics/logging for those
	// events. See monitorConnWrapper for the gRPC-specific per-connection
	// health watch plumbed in via pool.OnConnAdded.
	pool *connpool.Pool[*grpc.ClientConn]
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

	cfg := connpool.Config{
		DynamicScalingEnabled:  t.options.clientConnPoolDynamicScalingEnabled,
		MaxConcurrentStreams:   t.options.clientConnPoolMaxConcurrentStreams,
		ScaleUpThreshold:       t.options.clientConnPoolScaleUpThreshold,
		ScaleDownGap:           t.options.clientConnPoolScaleDownGap,
		MinConnections:         t.options.clientConnPoolMinConnections,
		MaxConnections:         t.options.clientConnPoolMaxConnections,
		IdleTimeout:            t.options.clientConnPoolIdleTimeout,
		ScalingMonitorInterval: t.options.clientConnPoolScalingMonitorInterval,
	}

	p := &grpcPeer{
		Peer:   abstractpeer.NewPeer(abstractpeer.PeerIdentifier(address), t),
		t:      t,
		ctx:    ctx,
		cancel: cancel,
	}

	dial := func(_ context.Context) (*grpc.ClientConn, error) {
		//lint:ignore SA1019 grpc.Dial is deprecated
		return grpc.Dial(address, dialOptions...)
	}
	p.pool = connpool.NewPool(p.ctx, cfg, dial, t.options.logger, address, connpool.NewReporter(t.metrics))
	p.pool.OnConnAdded = p.monitorConnWrapper

	t.options.logger.Debug("grpc: connection pool config resolved",
		zap.String("peer", address),
		zap.Bool("dynamicScalingEnabled", cfg.DynamicScalingEnabled),
		zap.Int("minConnections", cfg.MinConnections),
		zap.Int("maxConnections", cfg.MaxConnections),
		zap.Int32("maxConcurrentStreams", cfg.MaxConcurrentStreams),
		zap.Float64("scaleUpThreshold", cfg.ScaleUpThreshold),
		zap.Float64("scaleDownGap", cfg.ScaleDownGap),
		zap.Duration("idleTimeout", cfg.IdleTimeout),
		zap.Duration("scalingMonitorInterval", cfg.ScalingMonitorInterval),
	)

	// All connections are created via pool.AddConn (invoked by Start below)
	// -- no special primary connection.
	initialConnCount := 1
	if cfg.DynamicScalingEnabled {
		initialConnCount = cfg.MinConnections
	}
	if err := p.pool.Start(initialConnCount); err != nil {
		p.cancel()
		return nil, err
	}

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

// monitorConnWrapper is the pool's OnConnAdded callback: it spawns the
// per-connection health loop that watches gRPC connectivity state changes,
// keeps the peer status up to date, and triggers reconnection when a
// connection goes idle. It cleans up when the wrapper's context is
// cancelled (i.e. when the peer is stopped or the connection is evicted by
// the pool), per Pool.OnConnAdded's contract: it must call Remove and then
// ConnDone exactly once.
func (p *grpcPeer) monitorConnWrapper(w *connpool.Wrapper[*grpc.ClientConn]) {
	go func() {
		defer func() {
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
			// waiting on it (e.g. tests) observes a consistent metric state.
			close(w.StoppedC)
			p.pool.ConnDone()
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
				w.Conn.Connect()
			}

			// If this connection is Ready the peer is Available regardless of
			// other connections — no need to scan the pool. For any other state
			// (Connecting, TransientFailure, etc.) we must check all connections
			// to derive the correct aggregate status.
			// Skip when shutting down: NotifyStatusChanged acquires list.lock,
			// but the caller (abstractlist.stop) may already hold it while
			// waiting on p.wait().
			if p.ctx.Err() != nil {
				break
			}
			p.recomputeConnectionStatus()

			if !w.Conn.WaitForStateChange(w.Context(), grpcStatus) {
				break
			}
		}
	}()
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
