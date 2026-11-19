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
// This source file contains the implementation of a [net.Listener] wrapper.
package http2

import (
	"net"
	"sync"
	"sync/atomic"
)

// WrapListener wraps a [net.Listener] to support two-phase graceful shutdown
// with two HTTP/2 GOAWAY frames.
func WrapListener(l net.Listener, h ShutdownHelper) net.Listener {
	if l == nil || h == nil {
		return l
	}
	// No need to track anything if the helper is a no-op
	if _, ok := h.(*noopHelper); ok {
		return l
	}
	tl := newTrackingListener(l, h)
	if lt, ok := h.(listenerTracker); ok {
		lt.trackListener(tl)
	}
	return tl
}

// listenerTracker is a [ShutdownHelper] that stops accepting new connections
// on the listeners wrapped with it, once it begins shutting down.
type listenerTracker interface {
	trackListener(tl *TrackingListener)
}

var _ net.Listener = (*TrackingListener)(nil)

// TrackingListener is a [net.Listener] that tracks connections for shutdown purposes.
// Implements the [net.Listener] interface.
type TrackingListener struct {
	listener net.Listener
	helper   ShutdownHelper

	stopped    atomic.Bool   // listener closed by stopAccepting
	closeOnce  sync.Once     // closes listener
	closeErr   error         // what closing listener returned
	closed     chan struct{} // closed by Close
	closedOnce sync.Once     // closes closed
}

// newTrackingListener is the constructor for [TrackingListener].
func newTrackingListener(l net.Listener, h ShutdownHelper) *TrackingListener {
	return &TrackingListener{listener: l, helper: h, closed: make(chan struct{})}
}

// getTrackerFunc returns a ConnectionTrackerFunc that tracks connections using
// the listener's ShutdownHelper.
func (tl *TrackingListener) getTrackerFunc() ConnectionTrackerFunc {
	if tl.helper == nil {
		return noopTrackerFunc
	}
	return func(conn net.Conn) func() {
		if conn == nil {
			return nil
		}
		return tl.helper.TrackConnection(conn)
	}
}

// Accept implements the [net.Listener] interface for [TrackingListener].
// Once stopAccepting closed the listener, it only returns when Close is
// invoked.
func (tl *TrackingListener) Accept() (net.Conn, error) {
	conn, err := tl.listener.Accept()
	if err != nil {
		if tl.stopped.Load() {
			<-tl.closed
		}
		return nil, err
	}
	tc := newTrackedConn(conn, tl.getTrackerFunc())
	return tc, nil
}

// Close implements the [net.Listener] interface for [TrackingListener].
// It returns what closing the listener did, even if stopAccepting did it.
func (tl *TrackingListener) Close() error {
	tl.closedOnce.Do(func() { close(tl.closed) })
	return tl.closeListener()
}

// stopAccepting closes the listener, so that new connections are refused,
// but leaves Accept blocking until Close is invoked: [net/http.Server.Serve]
// fails with the error Accept returns, unless its Shutdown began.
func (tl *TrackingListener) stopAccepting() {
	tl.stopped.Store(true)
	_ = tl.closeListener()
}

// closeListener closes the listener once, and returns what that did.
func (tl *TrackingListener) closeListener() error {
	tl.closeOnce.Do(func() { tl.closeErr = tl.listener.Close() })
	return tl.closeErr
}

// Addr implements the [net.Listener] interface for [TrackingListener].
func (tl *TrackingListener) Addr() net.Addr {
	return tl.listener.Addr()
}
