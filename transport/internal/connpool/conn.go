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

// Package connpool provides transport-agnostic scaffolding for a per-peer
// pool of connections that scales up and down based on per-connection load.
// It exists because protocols such as HTTP/2 cap concurrent streams per TCP
// connection (SETTINGS_MAX_CONCURRENT_STREAMS, typically 250): once a single
// connection's in-flight load nears that cap, throughput is improved by
// opening additional connections rather than queuing.
//
// The state machine, the scale-up/scale-down/idle-cleanup algorithm, and the
// aggregate metrics reporter are generic across connection types. What
// differs per transport is how a connection is dialed and health-watched, so
// those are supplied by the caller via Pool.Dial and Pool.OnConnAdded.
package connpool

import (
	"context"
	"sync/atomic"
	"time"
)

// Conn is a single physical connection managed by a Pool.
type Conn interface {
	// Close closes the underlying connection. It is called exactly once, by
	// a transport-supplied per-connection watcher, immediately before that
	// watcher removes the wrapper owning this connection from the pool.
	Close() error
}

// State is the lifecycle state of a pooled connection.
type State int32

const (
	// StateActive means the connection accepts new streams/requests.
	StateActive State = iota
	// StateDraining means the connection no longer accepts new
	// streams/requests but is waiting for in-flight ones to complete.
	StateDraining
	// StateIdle means all streams/requests on the connection have finished;
	// it is waiting for the idle timeout before being closed.
	StateIdle
	// StateClosing means the pool's cleanup routine has claimed this idle
	// connection for closure via compare-and-swap. Terminal: no further
	// transitions are valid.
	StateClosing
)

// Wrapper wraps a single pooled connection of type T with the bookkeeping a
// Pool needs: lifecycle state, in-flight load, and idle tracking. All fields
// accessed concurrently are updated atomically; Wrapper holds no mutex.
type Wrapper[T Conn] struct {
	// Conn is the underlying pooled connection.
	Conn T

	ctx    context.Context
	cancel context.CancelFunc

	streamCount int32 // accessed atomically
	state       int32 // accessed atomically; holds a State

	createdAt      time.Time
	lastIdleAtNano int64 // atomic unix nanos; set when transitioning to StateIdle

	// StoppedC is closed once this wrapper's connection has been closed and
	// removed from its Pool.
	StoppedC chan struct{}
}

func newWrapper[T Conn](parentCtx context.Context, conn T) *Wrapper[T] {
	ctx, cancel := context.WithCancel(parentCtx)
	return &Wrapper[T]{
		Conn:      conn,
		ctx:       ctx,
		cancel:    cancel,
		state:     int32(StateActive),
		createdAt: time.Now(),
		StoppedC:  make(chan struct{}),
	}
}

// Context returns a context derived from the Pool's context that is
// cancelled when this specific wrapper is scheduled for removal (idle
// timeout, or an external Cancel call).
func (w *Wrapper[T]) Context() context.Context { return w.ctx }

// Cancel schedules this wrapper for removal from its Pool. Transports that
// detect a dead connection out-of-band (e.g. a failed round trip) should
// call Cancel to let the Pool's cleanup path close and remove it.
func (w *Wrapper[T]) Cancel() { w.cancel() }

// IncStreamCount atomically increments the in-flight request/stream count.
func (w *Wrapper[T]) IncStreamCount() { atomic.AddInt32(&w.streamCount, 1) }

// DecStreamCount atomically decrements the in-flight request/stream count.
func (w *Wrapper[T]) DecStreamCount() { atomic.AddInt32(&w.streamCount, -1) }

// StreamCount returns the current in-flight request/stream count.
func (w *Wrapper[T]) StreamCount() int32 { return atomic.LoadInt32(&w.streamCount) }

// GetState returns the current lifecycle state.
func (w *Wrapper[T]) GetState() State { return State(atomic.LoadInt32(&w.state)) }

func (w *Wrapper[T]) setState(s State) { atomic.StoreInt32(&w.state, int32(s)) }

// TransitionState atomically transitions the state from `from` to `to`.
// Returns true if the transition succeeded, false if the state was not
// `from`.
func (w *Wrapper[T]) TransitionState(from, to State) bool {
	return atomic.CompareAndSwapInt32(&w.state, int32(from), int32(to))
}

// IsActive reports whether the connection is currently accepting new
// streams/requests.
func (w *Wrapper[T]) IsActive() bool { return w.GetState() == StateActive }

// setIdleNow records the current time as the idle start time.
func (w *Wrapper[T]) setIdleNow() { atomic.StoreInt64(&w.lastIdleAtNano, time.Now().UnixNano()) }

// IdleSince returns the time this connection entered the idle state, or the
// zero time if it has not become idle yet.
func (w *Wrapper[T]) IdleSince() time.Time {
	ns := atomic.LoadInt64(&w.lastIdleAtNano)
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}
