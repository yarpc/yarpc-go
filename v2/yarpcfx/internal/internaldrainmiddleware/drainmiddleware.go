// Copyright (c) 2018 Uber Technologies, Inc.
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

// Package internaldrainmiddleware provides middleware that helps applications
// shut down gracefully. The yarpcfx module applies and manages this middleware
// automatically, so most users shouldn't need to interact with this package.
package internaldrainmiddleware

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

var _errServerStopping = yarpcerrors.UnavailableErrorf("server is shutting down")

// Middleware tracks the number of in-flight requests and provides a method to
// wait until they're successfully drained.
type Middleware struct {
	// Can't use a waitgroup, because adding while waiting is a race.
	cond     *sync.Cond
	pending  atomic.Uint64
	draining atomic.Bool
}

// New constructs a new Middleware.
func New() *Middleware {
	return &Middleware{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

// Handle implements middleware.UnaryInbound.
func (m *Middleware) Handle(ctx context.Context, req *transport.Request, w transport.ResponseWriter, h transport.UnaryHandler) error {
	if err := m.before(); err != nil {
		return err
	}
	err := h.Handle(ctx, req, w)
	m.after()
	return err
}

// HandleOneway implements middleware.OnewayInbound.
func (m *Middleware) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	if err := m.before(); err != nil {
		return err
	}
	err := h.HandleOneway(ctx, req)
	m.after()
	return err
}

// Drain blocks until all in-flight requests are completed.
func (m *Middleware) Drain() error {
	if !m.draining.CAS(false, true) {
		return errors.New("already drained")
	}
	m.cond.L.Lock()
	for m.pending.Load() != 0 {
		m.cond.Wait()
	}
	m.cond.L.Unlock()
	return nil
}

func (m *Middleware) before() error {
	if m.draining.Load() {
		return _errServerStopping
	}
	m.pending.Inc()
	return nil
}

func (m *Middleware) after() {
	if m.pending.Dec() == 0 {
		m.cond.Broadcast()
	}
}
