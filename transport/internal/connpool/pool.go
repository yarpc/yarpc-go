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

package connpool

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Config holds tuning parameters for a Pool.
type Config struct {
	// DynamicScalingEnabled gates all automatic pool scaling. When false the
	// pool is never grown or shrunk automatically by the scaler in
	// scaler.go, though PickConn/AddConn still work.
	DynamicScalingEnabled bool
	// MaxConcurrentStreams is the maximum number of concurrent
	// streams/requests a single connection is expected to carry (e.g.
	// HTTP/2's SETTINGS_MAX_CONCURRENT_STREAMS, commonly 250).
	MaxConcurrentStreams int32
	// ScaleUpThreshold is the fraction of MaxConcurrentStreams at which a
	// new connection is opened (e.g. 0.8 -> scale up at 200 active
	// streams out of 250).
	ScaleUpThreshold float64
	// ScaleDownGap is subtracted from ScaleUpThreshold to derive the
	// scale-down threshold, creating a hysteresis band that prevents
	// oscillation when the stream count hovers near the scale-up boundary.
	ScaleDownGap float64
	// MinConnections is the minimum number of connections kept in the pool.
	MinConnections int
	// MaxConnections is the maximum number of connections allowed in the
	// pool.
	MaxConnections int
	// IdleTimeout is how long a drained connection stays idle before it is
	// closed and removed from the pool.
	IdleTimeout time.Duration
	// ScalingMonitorInterval is how often the background monitor evaluates
	// the pool for scale-down and idle cleanup.
	ScalingMonitorInterval time.Duration
}

// Pool manages a dynamically-scaled set of connections of type T for a
// single logical peer. Membership is a lock-free copy-on-write slice
// (readers take a single atomic pointer load; writers CAS in a new slice),
// pool lifecycle/shutdown coordination is atomic-only (addingCount,
// shutdownStarted, and connWg -- deliberately no mutex, since this mechanism
// has been performance-tested), and TryScaleUp is single-flighted via the
// isScaling CAS flag.
//
// Pool does not itself watch connection health or spawn a per-connection
// lifecycle goroutine, since that is inherently specific to the kind of
// connection being pooled (e.g. gRPC's connectivity.State/WaitForStateChange
// vs. HTTP/2's pollable ClientConnState). A caller supplies OnConnAdded to
// plug in its own watcher: AddConn calls it synchronously immediately after
// publishing the new connection, with the connWg accounting for that
// goroutine already reserved. That goroutine is responsible for calling
// Remove and then ConnDone exactly once, when it observes the connection's
// context has been cancelled.
type Pool[T Conn] struct {
	// Dial creates one new underlying connection. Called by AddConn, both
	// for the initial fill (Start) and by the scaler when scaling up.
	Dial func(ctx context.Context) (T, error)

	// OnConnAdded, if set, is invoked synchronously by AddConn immediately
	// after a new connection has been published into the pool. AddConn has
	// already called connWg.Add(1) on this connection's behalf; whatever
	// goroutine OnConnAdded spawns must call Remove and then ConnDone
	// exactly once, when it observes w.Context().Done().
	OnConnAdded func(*Wrapper[T])

	Config Config
	Logger *zap.Logger
	// ID identifies the pool in log lines, e.g. the peer's address.
	ID string

	metrics *Reporter

	ctx    context.Context
	cancel context.CancelFunc

	connsPtr  atomic.Pointer[[]*Wrapper[T]]
	connCount atomic.Int32

	// isScaling is a CAS-guarded single-flight flag: at most one scale-up
	// goroutine runs at a time, and its being set also tells CleanupIdleConns
	// to hold off closing idle connections that TryScaleUp might reactivate.
	isScaling              int32
	atMaxConnectionsLogged atomic.Bool

	// connWg tracks background pool goroutines: per-connection watchers
	// spawned via OnConnAdded, RunScalingMonitor, and TryScaleUp workers.
	// stoppedC closes once they have all finished and this pool's metric
	// contribution has been zeroed.
	connWg sync.WaitGroup

	// addingCount tracks AddConn calls that have passed their entry point
	// but have not yet called connWg.Add(1) or bailed. The teardown
	// goroutine spins on this counter (after setting shutdownStarted) to
	// guarantee that connWg.Wait is not called before any racing
	// connWg.Add(1) completes.
	addingCount atomic.Int32

	// shutdownStarted is set to true when the pool begins shutting down.
	// AddConn checks this after incrementing addingCount to avoid calling
	// connWg.Add(1) after the teardown goroutine has passed its spin.
	shutdownStarted atomic.Bool

	stoppedC chan struct{}
}

// NewPool creates a Pool. The pool does not dial any connections or start
// its background scaler until Start is called.
func NewPool[T Conn](parentCtx context.Context, cfg Config, dial func(ctx context.Context) (T, error), logger *zap.Logger, id string, metrics *Reporter) *Pool[T] {
	ctx, cancel := context.WithCancel(parentCtx)
	p := &Pool[T]{
		Dial:     dial,
		Config:   cfg,
		Logger:   logger,
		ID:       id,
		metrics:  metrics,
		ctx:      ctx,
		cancel:   cancel,
		stoppedC: make(chan struct{}),
	}
	// Publish a nil slice so LoadConns never returns a non-nil-pointer/nil-
	// slice mismatch before the first AddConn call.
	var empty []*Wrapper[T]
	p.connsPtr.Store(&empty)
	return p
}

// Start dials initialConnCount connections and, if the pool's Config enables
// dynamic scaling, starts the background scaling monitor. It also starts the
// goroutine that tears the pool down once Stop is called. If any initial
// dial fails, Start cancels the pool's context and returns the error without
// starting anything else.
func (p *Pool[T]) Start(initialConnCount int) error {
	for range initialConnCount {
		if _, err := p.AddConn(); err != nil {
			p.cancel()
			return err
		}
	}
	if p.Config.DynamicScalingEnabled {
		p.connWg.Add(1)
		go p.RunScalingMonitor()
	}

	// Close stoppedC once all pool goroutines have finished. shutdownStarted
	// gates new connWg.Add(1) calls; addingCount ensures we wait for any
	// AddConn already past its ctx check before calling Wait.
	go func() {
		<-p.ctx.Done()
		p.shutdownStarted.Store(true)
		for p.addingCount.Load() > 0 {
			runtime.Gosched()
		}
		p.connWg.Wait()
		// Zero this pool's contribution after every pool goroutine has
		// stopped so a late RefreshMetrics snapshot cannot leave residual
		// values on the shared gauges.
		if p.metrics != nil {
			p.metrics.SetCounts(0, 0, 0)
		}
		close(p.stoppedC)
	}()

	return nil
}

// Stop begins asynchronous teardown of the pool. It does not block; use Wait
// to block until teardown completes.
func (p *Pool[T]) Stop() { p.cancel() }

// Wait blocks until the pool has fully torn down (all connections closed,
// background goroutines exited) after Stop.
func (p *Pool[T]) Wait() { <-p.stoppedC }

// LoadConns returns the current immutable connection snapshot. Safe to call
// from any goroutine without holding any lock.
func (p *Pool[T]) LoadConns() []*Wrapper[T] {
	ptr := p.connsPtr.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}

// AddConn dials a new connection to the pool's address using Dial, appends
// it to the pool, and invokes OnConnAdded so the caller can start watching
// it.
func (p *Pool[T]) AddConn() (*Wrapper[T], error) {
	conn, err := p.Dial(p.ctx)
	if err != nil {
		return nil, err
	}
	w := newWrapper(p.ctx, conn)

	// Enter the critical window: increment addingCount so the teardown
	// goroutine's spin waits for us to either bail or call connWg.Add(1).
	p.addingCount.Add(1)
	if p.ctx.Err() != nil || p.shutdownStarted.Load() {
		p.addingCount.Add(-1)
		_ = conn.Close()
		return nil, p.ctx.Err()
	}
	p.connWg.Add(1)
	p.addingCount.Add(-1)

	// CAS loop: copy the slice, append the new wrapper, publish atomically.
	for {
		old := p.connsPtr.Load()
		var oldSlice []*Wrapper[T]
		if old != nil {
			oldSlice = *old
		}
		next := make([]*Wrapper[T], len(oldSlice)+1)
		copy(next, oldSlice)
		next[len(oldSlice)] = w
		if p.connsPtr.CompareAndSwap(old, &next) {
			p.connCount.Store(int32(len(next)))
			break
		}
	}

	if p.OnConnAdded != nil {
		p.OnConnAdded(w)
	}
	p.RefreshMetrics()
	return w, nil
}

// Remove removes w from the pool by pointer identity. It only mutates pool
// membership: it does not close the connection, refresh metrics, or signal
// StoppedC -- callers (typically a per-connection watcher spawned from
// OnConnAdded) sequence those themselves, in whatever order suits that
// transport's own cleanup needs (e.g. recomputing aggregate peer status).
func (p *Pool[T]) Remove(w *Wrapper[T]) {
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
		next := make([]*Wrapper[T], len(oldSlice)-1)
		copy(next, oldSlice[:idx])
		copy(next[idx:], oldSlice[idx+1:])
		if p.connsPtr.CompareAndSwap(old, &next) {
			p.connCount.Store(int32(len(next)))
			return
		}
	}
}

// ConnDone signals that a per-connection watcher goroutine spawned from
// OnConnAdded has finished. Must be called exactly once per such goroutine.
func (p *Pool[T]) ConnDone() { p.connWg.Done() }

// PickConn returns the active connection with the fewest in-flight
// streams/requests, or nil if the pool has no active connections.
func (p *Pool[T]) PickConn() *Wrapper[T] {
	conns := p.LoadConns()
	var best *Wrapper[T]
	for _, c := range conns {
		if !c.IsActive() {
			continue
		}
		if best == nil || c.StreamCount() < best.StreamCount() {
			best = c
		}
	}
	return best
}
