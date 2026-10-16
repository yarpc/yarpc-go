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
// This source file contains the implementation of a [net.Conn] wrapper.
package http2

import (
	"net"
	"time"
)

// ConnectionTrackerFunc is the function signature for keeping track of a
// connection.
type ConnectionTrackerFunc func(net.Conn)

// noopTrackerFunc is a no-op implementation of [ConnectionTrackerFunc].
func noopTrackerFunc(conn net.Conn) {}

var _ net.Conn = (*TrackedConn)(nil)

// TrackedConn represents a connection tracked for shutdown purposes.
// Implements the [net.Conn] interface.
type TrackedConn struct {
	conn      net.Conn
	trackFn   ConnectionTrackerFunc
}

// newTrackedConn is the constructor for [TrackedConn].
func newTrackedConn(conn net.Conn, fn ConnectionTrackerFunc) net.Conn {
	return &TrackedConn{conn: conn, trackFn: fn}
}

// Read implements the [net.Conn] interface for [TrackedConn].
func (tc *TrackedConn) Read(b []byte) (n int, err error) {
	return tc.conn.Read(b)
}

// Write implements the [net.Conn] interface for [TrackedConn].
func (tc *TrackedConn) Write(b []byte) (n int, err error) {
	return tc.conn.Write(b)
}

// Close implements the [net.Conn] interface for [TrackedConn].
func (tc *TrackedConn) Close() error {
	return tc.conn.Close()
}

// LocalAddr implements the [net.Conn] interface for [TrackedConn].
func (tc *TrackedConn) LocalAddr() net.Addr {
	return tc.conn.LocalAddr()
}

// RemoteAddr implements the [net.Conn] interface for [TrackedConn].
func (tc *TrackedConn) RemoteAddr() net.Addr {
	return tc.conn.RemoteAddr()
}

// SetDeadline implements the [net.Conn] interface for [TrackedConn].
func (tc *TrackedConn) SetDeadline(t time.Time) error {
	return tc.conn.SetDeadline(t)
}

// SetReadDeadline implements the [net.Conn] interface for [TrackedConn].
func (tc *TrackedConn) SetReadDeadline(t time.Time) error {
	return tc.conn.SetReadDeadline(t)
}

// SetWriteDeadline implements the [net.Conn] interface for [TrackedConn].
func (tc *TrackedConn) SetWriteDeadline(t time.Time) error {
	return tc.conn.SetWriteDeadline(t)
}
