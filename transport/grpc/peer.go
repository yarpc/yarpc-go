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
	"sync"

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

	mu      sync.RWMutex
	conns   []*grpcClientConnWrapper
	poolCfg connPoolConfig
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
		poolCfg: connPoolConfig{
			dynamicScalingEnabled: t.options.clientConnPoolDynamicScalingEnabled,
			maxConcurrentStreams:   t.options.clientConnPoolMaxConcurrentStreams,
			scaleUpThreshold:      t.options.clientConnPoolScaleUpThreshold,
			minConnections:        t.options.clientConnPoolMinConnections,
			maxConnections:        t.options.clientConnPoolMaxConnections,
			idleTimeout:           t.options.clientConnPoolIdleTimeout,
		},
	}

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
	// We acquire mu briefly after ctx is cancelled to ensure any in-flight
	// addConn() that passed its context check has already called connWg.Add(1)
	// before we call connWg.Wait().
	go func() {
		<-p.ctx.Done()
		// Acquire mu after cancellation to ensure any addConn() call that
		// passed its ctx.Err() check has already called connWg.Add(1).
		// The empty critical section is intentional: we need the memory
		// barrier, not exclusive access to any data.
		p.mu.Lock()
		p.mu.Unlock() //nolint:staticcheck
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

	// Hold mu while registering with connWg and appending to conns.  The
	// lifecycle goroutine in newPeer acquires mu after ctx is cancelled so
	// that it observes any Add(1) call that raced with cancellation.
	p.mu.Lock()
	if p.ctx.Err() != nil {
		p.mu.Unlock()
		_ = clientConn.Close()
		return p.ctx.Err()
	}
	p.connWg.Add(1)
	p.conns = append(p.conns, w)
	p.mu.Unlock()

	go p.monitorConnWrapper(w)
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
		close(w.stoppedC)
		p.removeConn(w)
		// Recompute connection status now that this peer is gone.
		p.recomputeConnectionStatus()
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
	p.mu.RLock()
	best := peer.Unavailable
	for _, c := range p.conns {
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
	p.mu.RUnlock()
	p.setConnectionStatus(best)
}

// removeConn removes a wrapper from the peer's connection pool.
func (p *grpcPeer) removeConn(w *grpcClientConnWrapper) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, c := range p.conns {
		if c == w {
			p.conns = append(p.conns[:i], p.conns[i+1:]...)
			return
		}
	}
}

// pickConn returns the active connection in the pool with the lowest current
// stream count.  Returns nil if the pool contains no active connections.
func (p *grpcPeer) pickConn() *grpcClientConnWrapper {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var best *grpcClientConnWrapper
	for _, c := range p.conns {
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
