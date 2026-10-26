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

// Package http2 provides functionality useful for HTTP/2 transport.
// This source file contains implementations of a server shutdown helper.
package http2

import (
	"context"
	"net"
	"sync"
)

// ShutdownHelper is an interface that provides a shutdown callback intended
// to be used in [Server.RegisterOnShutdown].
type ShutdownHelper interface {
	// TrackConnection is used to track connections.
	// It returns the function to stop tracking conn once closed, or nil if
	// conn is not tracked.
	TrackConnection(conn net.Conn) (untrack func())

	// BeginShutdown starts the shutdown process.
	// Note: It MUST be invoked before [net/http.Server.Shutdown] and so before IsShutdownComplete.
	BeginShutdown()

	// ShutdownCallback is invoked during the server's shutdown process.
	ShutdownCallback()

	// IsShutdownReady indicates whether the first phase of the shutdown
	// process is over on every connection, so that the server can shut down:
	// the first GOAWAY was sent, and its PING acknowledged or timed out.
	// Note: It MUST be invoked after [ShutdownHelper.BeginShutdown], and
	// before [net/http.Server.Shutdown].
	// Warning: This MAY be a blocking operation.
	IsShutdownReady(ctx context.Context) bool

	// IsShutdownComplete indicates whether the shutdown process has completed.
	// Note: It MUST be invoked after [ShutdownHelper.BeginShutdown] has been called.
	// Warning: This MAY be a blocking operation.
	IsShutdownComplete(ctx context.Context) bool
}

// NewNoopShutdownHelper creates a no-op HTTP/2 server shutdown helper.
func NewNoopShutdownHelper() ShutdownHelper {
	return &noopHelper{}
}

// NewShutdownHelper creates an HTTP/2 server shutdown helper.
func NewShutdownHelper() ShutdownHelper {
	return newGracefulShutdownHelper()
}

var (
	_ ShutdownHelper = (*noopHelper)(nil)
	_ ShutdownHelper = (*gracefulShutdownHelper)(nil)
)

// noopHelper is a no-op implementation of [ShutdownHelper].
type noopHelper struct{}

// TrackConnection implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) TrackConnection(conn net.Conn) func() { return nil }

// BeginShutdown implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) BeginShutdown() {}

// ShutdownCallback implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) ShutdownCallback() {}

// IsShutdownReady implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) IsShutdownReady(ctx context.Context) bool {
	return true
}

// IsShutdownComplete implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) IsShutdownComplete(ctx context.Context) bool {
	return true
}

// shutdownState encodes the state of a [gracefulShutdownHelper]'s shutdown process.
type shutdownState int

const (
	shutdownNone     shutdownState = iota // BeginShutdown not invoked yet.
	shutdownStarted                       // Connections are shutting down.
	shutdownReady                         // IsShutdownReady invoked: the first phase of new connections is not waited on.
	shutdownComplete                      // IsShutdownComplete invoked: new connections are refused; final state.
)

// gracefulShutdownHelper is an implementation of [ShutdownHelper] that shuts
// down every tracked HTTP/2 connection gracefully, with two GOAWAY frames,
// on server shutdown. Each [TrackedConn] runs its own shutdown.
type gracefulShutdownHelper struct {
	mu    sync.Mutex
	conns map[*TrackedConn]struct{}
	state shutdownState
	wg    sync.WaitGroup // one per connection shutting down, until it closes
	ready sync.WaitGroup // one per connection shutting down before IsShutdownReady, until its first phase is over
}

// newGracefulShutdownHelper is the constructor for [gracefulShutdownHelper].
func newGracefulShutdownHelper() *gracefulShutdownHelper {
	return &gracefulShutdownHelper{
		conns: make(map[*TrackedConn]struct{}),
	}
}

// TrackConnection implements the [ShutdownHelper] interface for [gracefulShutdownHelper].
// Only a [TrackedConn] can be shut down gracefully, so other connections are
// not tracked.
func (h *gracefulShutdownHelper) TrackConnection(conn net.Conn) func() {
	tc, ok := conn.(*TrackedConn)
	if !ok {
		return nil
	}

	h.mu.Lock()
	if h.state == shutdownComplete {
		h.mu.Unlock()
		// Too late to shut it down gracefully: refuse it, so that it serves
		// nothing. It can only come after http.Server.Shutdown gave up.
		_ = tc.Close()
		return nil
	}
	defer h.mu.Unlock()
	h.conns[tc] = struct{}{}
	if h.state != shutdownNone {
		// Accepted before the listener closed, but only sent its preface
		// after BeginShutdown. This happens before http.Server.Shutdown
		// returns, so before IsShutdownComplete waits.
		h.shutdownLocked(tc)
	}
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.conns, tc)
	}
}

// shutdownLocked starts the graceful shutdown of tc, which wg counts until
// tc closes.
// Note: only thread-safe if h.mu is held.
func (h *gracefulShutdownHelper) shutdownLocked(tc *TrackedConn) {
	if !tc.startShutdown() {
		return
	}
	h.wg.Add(1)
	ready := func() {}
	if h.state == shutdownStarted {
		// IsShutdownReady does not wait yet, so this happens before it does.
		h.ready.Add(1)
		ready = h.ready.Done
	}
	go func() {
		defer h.wg.Done()
		tc.drain()
		ready()
		<-tc.done()
	}()
}

// BeginShutdown implements the [ShutdownHelper] interface for [gracefulShutdownHelper].
func (h *gracefulShutdownHelper) BeginShutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.state != shutdownNone {
		return
	}
	h.state = shutdownStarted
	for tc := range h.conns {
		h.shutdownLocked(tc)
	}
}

// ShutdownCallback implements the [ShutdownHelper] interface for [gracefulShutdownHelper].
// It is a no-op if BeginShutdown was invoked first, as it must be: net/http
// runs it in a goroutine that may only start after IsShutdownComplete.
func (h *gracefulShutdownHelper) ShutdownCallback() {
	h.BeginShutdown()
}

// IsShutdownReady implements the [ShutdownHelper] interface for [gracefulShutdownHelper].
// Once BeginShutdown was invoked, it blocks until the first phase is over on
// every connection shutting down: the client acknowledged the PING sent with
// the first GOAWAY, or did not in time, or the connection closed. It returns
// false if ctx ends first.
func (h *gracefulShutdownHelper) IsShutdownReady(ctx context.Context) bool {
	h.mu.Lock()
	if h.state != shutdownStarted {
		h.mu.Unlock()
		return true
	}
	// No connection is waited on from now on, so that every ready.Add
	// happens before ready.Wait.
	h.state = shutdownReady
	h.mu.Unlock()

	done := make(chan struct{})
	go func() {
		h.ready.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

// IsShutdownComplete implements the [ShutdownHelper] interface for [gracefulShutdownHelper].
// Once BeginShutdown was invoked, it refuses new connections, and blocks
// until every tracked connection has closed, or until ctx ends: it then
// closes the remaining connections and returns false.
func (h *gracefulShutdownHelper) IsShutdownComplete(ctx context.Context) bool {
	h.mu.Lock()
	if h.state == shutdownNone {
		h.mu.Unlock()
		return true
	}
	// No connection is tracked from now on, so that every wg.Add happens
	// before wg.Wait.
	h.state = shutdownComplete
	h.mu.Unlock()

	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
	}

	h.mu.Lock()
	conns := make([]*TrackedConn, 0, len(h.conns))
	for tc := range h.conns {
		conns = append(conns, tc)
	}
	h.mu.Unlock()
	for _, tc := range conns {
		_ = tc.Close()
	}
	return false
}
