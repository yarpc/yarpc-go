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
	"time"

	"google.golang.org/grpc"
)

// connState is the lifecycle state of a pooled gRPC connection.
type connState int32

const (
	// connStateActive means the connection accepts new streams.
	connStateActive connState = iota
	// connStateDraining means the connection is no longer accepting new
	// streams but is waiting for in-flight streams to complete.
	connStateDraining
	// connStateIdle means all streams have finished; the connection is
	// waiting for the idle timeout before being closed.
	connStateIdle
)

// connPoolConfig holds configuration for the per-peer connection pool.
// Values are derived from transportOptions at peer creation time.
type connPoolConfig struct {
	// dynamicScalingEnabled gates all automatic connection-pool scaling.
	// When false the pool is never grown or shrunk automatically.
	dynamicScalingEnabled bool
	// maxConcurrentStreams is the HTTP/2 SETTINGS_MAX_CONCURRENT_STREAMS
	// value enforced by the server (default 250).
	maxConcurrentStreams int32
	// scaleUpThreshold is the fraction of maxConcurrentStreams at which a
	// new connection is opened (e.g. 0.8 → scale up at 200 active streams).
	scaleUpThreshold float64
	// minConnections is the minimum number of connections kept in the pool.
	minConnections int
	// maxConnections is the maximum number of connections allowed in the pool.
	maxConnections int
	// idleTimeout is how long a drained connection stays idle before it is
	// closed and removed from the pool.
	idleTimeout time.Duration
}

// grpcClientConnWrapper wraps a single *grpc.ClientConn with connection-pool
// metadata.  All fields that are accessed concurrently are updated atomically.
type grpcClientConnWrapper struct {
	clientConn *grpc.ClientConn

	// ctx is derived from the peer's context.  Cancelling it stops this
	// connection's monitor goroutine, which in turn closes clientConn.
	ctx    context.Context
	cancel context.CancelFunc

	// streamCount is the number of in-flight streams on this connection.
	streamCount int32 // accessed atomically
	// state is the current connState of this wrapper.
	state connState // accessed atomically

	createdAt      time.Time
	lastIdleAtNano int64 // atomic unix nanos; set when transitioning to connStateIdle

	// stoppedC is closed by the connection's monitor goroutine after the
	// underlying clientConn has been closed.
	stoppedC chan struct{}
}

func newConnWrapper(parentCtx context.Context, clientConn *grpc.ClientConn) *grpcClientConnWrapper {
	ctx, cancel := context.WithCancel(parentCtx)
	return &grpcClientConnWrapper{
		clientConn: clientConn,
		ctx:        ctx,
		cancel:     cancel,
		state:      connStateActive,
		createdAt:  time.Now(),
		stoppedC:   make(chan struct{}),
	}
}

// incStreamCount atomically increments the active stream count.
func (w *grpcClientConnWrapper) incStreamCount() {
	atomic.AddInt32(&w.streamCount, 1)
}

// decStreamCount atomically decrements the active stream count.
func (w *grpcClientConnWrapper) decStreamCount() {
	atomic.AddInt32(&w.streamCount, -1)
}

// getStreamCount returns the current number of active streams.
func (w *grpcClientConnWrapper) getStreamCount() int32 {
	return atomic.LoadInt32(&w.streamCount)
}

// getState returns the current connection state.
func (w *grpcClientConnWrapper) getState() connState {
	return connState(atomic.LoadInt32((*int32)(&w.state)))
}

// setState atomically updates the connection state.
func (w *grpcClientConnWrapper) setState(s connState) {
	atomic.StoreInt32((*int32)(&w.state), int32(s))
}

// isActive reports whether the connection is currently accepting new streams.
func (w *grpcClientConnWrapper) isActive() bool {
	return w.getState() == connStateActive
}

// setIdleNow records the current time as the idle start time.
func (w *grpcClientConnWrapper) setIdleNow() {
	atomic.StoreInt64(&w.lastIdleAtNano, time.Now().UnixNano())
}

// idleSince returns the time when this connection entered the idle state,
// or the zero time if it has not become idle yet.
func (w *grpcClientConnWrapper) idleSince() time.Time {
	ns := atomic.LoadInt64(&w.lastIdleAtNano)
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}
