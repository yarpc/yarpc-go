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

import "net"

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
	return newTrackingListener(l, h)
}

var _ net.Listener = (*TrackingListener)(nil)

// TrackingListener is a [net.Listener] that tracks connections for shutdown purposes.
// Implements the [net.Listener] interface.
type TrackingListener struct {
	listener net.Listener
	helper   ShutdownHelper
}

// newTrackingListener is the constructor for [TrackingListener].
func newTrackingListener(l net.Listener, h ShutdownHelper) net.Listener {
	return &TrackingListener{listener: l, helper: h}
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
func (tl *TrackingListener) Accept() (net.Conn, error) {
	conn, err := tl.listener.Accept()
	if err != nil {
		return nil, err
	}
	tc := newTrackedConn(conn, tl.getTrackerFunc())
	return tc, nil
}

// Close implements the [net.Listener] interface for [TrackingListener].
func (tl *TrackingListener) Close() error {
	return tl.listener.Close()
}

// Addr implements the [net.Listener] interface for [TrackingListener].
func (tl *TrackingListener) Addr() net.Addr {
	return tl.listener.Addr()
}
