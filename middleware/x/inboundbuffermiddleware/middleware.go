// Copyright (c) 2020 Uber Technologies, Inc.
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

package inboundbuffermiddleware

import (
	"context"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/priority"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

// Buffer is an inbound unary middleware that ensures gradual degradation in
// the face of excess load, quality of service for higher priority requests,
// and coordinates load shedding for dependent requests.
type Buffer struct {
	concurrency atomic.Int32

	mx        sync.Mutex
	pendingCh chan struct{}
	stopCh    chan struct{}
	doneCh    chan struct{}
	entities  []entity
	buffer    buffer

	prioritizer priority.Prioritizer
	logger      *zap.Logger
	now         func() time.Time
}

type entity struct {
	ctx   context.Context
	req   *transport.Request
	res   transport.ResponseWriter
	errCh chan error
	next  transport.UnaryHandler
}

// New creates a buffer, an inbound unary middleware.
func New(opts ...Option) *Buffer {
	options := defaultOptions
	for _, opt := range opts {
		opt.apply(&options)
	}

	logger := options.logger
	if logger == nil {
		logger = zap.NewNop()
	}

	prioritizer := options.prioritizer
	if prioritizer == nil {
		prioritizer = priority.NopPrioritizer
	}

	now := options.now
	if now == nil {
		now = time.Now
	}

	b := &Buffer{
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}, options.concurrency),
		pendingCh:   make(chan struct{}, 1),
		entities:    make([]entity, options.capacity),
		prioritizer: prioritizer,
		logger:      logger,
		now:         now,
	}
	b.buffer.Init(options.capacity)
	b.concurrency.Store(int32(options.concurrency))
	return b
}

// Start spawns workers to process inbound unary requests.
func (b *Buffer) Start(ctx context.Context) error {
	b.logger.Debug("inbound buffer middleware: initiated startup")
	for i := 0; i < int(b.concurrency.Load()); i++ {
		go worker(b, i)
	}
	return nil
}

// Stop shuts down workers and waits for them all to gracefully exit, or
// returns an error early if the deadline in context occurs first.
func (b *Buffer) Stop(ctx context.Context) error {
	b.logger.Debug("inbound buffer middleware: initiated shutdown")

	close(b.stopCh)
	for {
		select {
		case <-ctx.Done():
			b.logger.Error("inbound buffer middleware: shutdown aborted", zap.Error(ctx.Err()))
			return ctx.Err()
		case <-b.stopCh:
			if b.concurrency.Load() == 0 {
				b.logger.Debug("inbound buffer middleware: completed shutdown")
				return nil
			}
		}
	}
}

// Handle receives requests and schedules them or drops them if there's no room
// for the request in the queue.
func (b *Buffer) Handle(ctx context.Context, req *transport.Request, res transport.ResponseWriter, next transport.UnaryHandler) error {
	// Read or create a priority for this request.

	// Put the request on the queue, if we can.
	errCh := b.put(ctx, req, res, next)
	if errCh == nil {
		return yarpcerrors.Newf(yarpcerrors.CodeResourceExhausted, "too busy and insuficient priority")
	}

	// Non-blocking poke on the pending channel to wake a worker if one has not
	// already been wakened.
	// If the channel is already full, our poke will get ignored, but the next
	// worker to successfully take an item off the queue will poke the channel
	// again to keep the cycle going.
	select {
	case b.pendingCh <- struct{}{}:
	default:
	}

	// Wait for response error.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (b *Buffer) put(ctx context.Context, req *transport.Request, res transport.ResponseWriter, next transport.UnaryHandler) <-chan error {
	// Get or create priority from context.
	p, f := b.prioritizer.Priority(ctx, req.ToRequestMeta())
	priority := uint64(p)*1000 + uint64(f)

	// Extrace deadline from request context.
	deadline := maxUint64
	if d, ok := ctx.Deadline(); ok {
		deadline = uint64(d.UnixNano())
	}

	// We have postponed acquiring the lock because obtaining deadline and
	// priority did not require the buffer.
	b.mx.Lock()
	defer b.mx.Unlock()

	// Evict expired requests.
	now := uint64(b.now().UnixNano())
	for b.buffer.EvictExpired(now) != -1 {
		// Continue until empty or all expired requests evicted.
	}

	if b.buffer.Full() {
		b.buffer.EvictLowerPriority(priority)
	}

	i := b.buffer.Put(deadline, priority)
	if i < 0 {
		return nil
	}
	// fmt.Fprintf(os.Stderr, "put %d, pending %v\n", i, b.buffer.free[:b.buffer.length])

	errCh := make(chan error, 1)
	b.entities[i] = entity{
		ctx:   ctx,
		req:   req,
		res:   res,
		errCh: errCh,
		next:  next,
	}
	return errCh
}

func (b *Buffer) pop() (ctx context.Context, req *transport.Request, res transport.ResponseWriter, errCh chan error, next transport.UnaryHandler, ok bool) {
	b.mx.Lock()
	defer b.mx.Unlock()

	i := b.buffer.Pop()
	if i < 0 {
		return
	}

	e := &b.entities[i]
	ctx = e.ctx
	req = e.req
	res = e.res
	errCh = e.errCh
	next = e.next
	ok = true

	// Release garbage.
	b.entities[i] = entity{}

	return
}

func worker(b *Buffer, i int) {
	b.logger.Debug("inbound buffer middleware: worker started", zap.Int("worker", i))

	for {
		select {
		case <-b.stopCh:
			b.logger.Debug("inbound buffer middleware: worker shut down", zap.Int("worker", i))
			b.concurrency.Dec()
			b.doneCh <- struct{}{}
			return
		case <-b.pendingCh:
		}

		if ctx, req, res, errCh, next, ok := b.pop(); ok {
			// Put a token back in the notifier. There may be more pending
			// requests for other workers.
			b.pendingCh <- struct{}{}

			errCh <- next.Handle(ctx, req, res)
		}
	}
}
