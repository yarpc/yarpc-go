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
	"sync"
	"sync/atomic"
	"time"

	xhttp2 "golang.org/x/net/http2"
)

// ConnectionTrackerFunc is the function signature for keeping track of a
// connection.
// It returns the function to stop tracking it, or nil if it is not tracked.
type ConnectionTrackerFunc func(net.Conn) (untrack func())

// noopTrackerFunc is a no-op implementation of [ConnectionTrackerFunc].
func noopTrackerFunc(conn net.Conn) func() { return nil }

var _ net.Conn = (*TrackedConn)(nil)

// TrackedConn represents a connection tracked for shutdown purposes.
// Implements the [net.Conn] interface.
//
// Once it reads the HTTP/2 client preface, it follows HTTP/2 frames in both
// directions, so that it can shut down gracefully with two GOAWAY frames
// written between them, as RFC 9113 §6.8 describes.
type TrackedConn struct {
	conn    net.Conn
	trackFn ConnectionTrackerFunc

	// Only used by the goroutine reading.
	sniffed   int  // bytes of the HTTP/2 client preface read so far
	untracked bool // not HTTP/2 with prior knowledge, or not tracked
	rscan     frameScanner

	lmu     sync.Mutex // serializes tracking with Close; protects untrack
	untrack func()
	tracked atomic.Bool

	idMu         sync.Mutex // protects lastStreamID
	lastStreamID uint32     // highest stream ID the client opened

	shuttingDown atomic.Bool
	pingAcked    chan struct{}
	ackOnce      sync.Once
	closed       chan struct{}
	closeOnce    sync.Once

	wmu     sync.Mutex // serializes writes, so frames go between frames; protects the next fields
	wscan   frameScanner
	goAway  goAwayState
	pending []byte // frames to write at the next frame boundary
	split   int    // where Write writes pending frames ahead of a server GOAWAY, or -1
}

// newTrackedConn is the constructor for [TrackedConn].
func newTrackedConn(conn net.Conn, fn ConnectionTrackerFunc) net.Conn {
	return &TrackedConn{
		conn:      conn,
		trackFn:   fn,
		pingAcked: make(chan struct{}),
		closed:    make(chan struct{}),
	}
}

// track hands tc to trackFn, unless already closed, and reports whether
// tc is tracked.
func (tc *TrackedConn) track() bool {
	tc.lmu.Lock()
	defer tc.lmu.Unlock()
	select {
	case <-tc.closed:
		return false
	default:
	}
	if tc.trackFn != nil {
		tc.untrack = tc.trackFn(tc)
	}
	tc.tracked.Store(tc.untrack != nil)
	return tc.untrack != nil
}

// shutdown starts the two-phase graceful shutdown of a tracked connection in
// the background; see [TrackedConn.drain]. Only the first call has an effect.
func (tc *TrackedConn) shutdown() {
	if tc.tracked.Load() && tc.shuttingDown.CompareAndSwap(false, true) {
		go tc.drain()
	}
}

// done returns a channel that is closed once tc is closed.
func (tc *TrackedConn) done() <-chan struct{} {
	return tc.closed
}

// Read implements the [net.Conn] interface for [TrackedConn].
// It sniffs the HTTP/2 client preface to only track HTTP/2 connections, then frame headers.
func (tc *TrackedConn) Read(b []byte) (n int, err error) {
	n, err = tc.conn.Read(b)
	if tc.untracked {
		return n, err
	}

	p := b[:n]
	if preface := xhttp2.ClientPreface; tc.sniffed < len(preface) {
		k := min(len(p), len(preface)-tc.sniffed)
		if string(p[:k]) != preface[tc.sniffed:tc.sniffed+k] {
			tc.untracked = true
			return n, err
		}
		tc.sniffed += k
		if tc.sniffed < len(preface) {
			return n, err
		}
		p = p[k:]

		// Track the connection only once it has been confirmed as HTTP/2
		if !tc.track() {
			tc.untracked = true
			return n, err
		}
	}
	tc.rscan.scan(p, tc.frameRead)
	return n, err
}

// Write implements the [net.Conn] interface for [TrackedConn].
// A GOAWAY frame the HTTP/2 server writes itself takes the place
// of the second GOAWAY, so the first one is written ahead of it.
func (tc *TrackedConn) Write(b []byte) (n int, err error) {
	if !tc.tracked.Load() {
		return tc.conn.Write(b)
	}

	tc.wmu.Lock()
	defer tc.wmu.Unlock()
	tc.split = -1
	tc.wscan.scan(b, tc.frameWrite)
	split := len(b)
	if tc.split >= 0 {
		split = tc.split
	}

	n, err = tc.conn.Write(b[:split])
	if err == nil && (split < len(b) || tc.wscan.atBoundary()) {
		err = tc.flushLocked()
	}
	if err == nil && split < len(b) {
		var m int
		m, err = tc.conn.Write(b[split:])
		n += m
	}
	return n, err
}

// Close implements the [net.Conn] interface for [TrackedConn].
// If shutting down, it writes the GOAWAY frames still owed before closing,
// then stops tracking the connection.
func (tc *TrackedConn) Close() (err error) {
	tc.closeOnce.Do(func() {
		if tc.shuttingDown.Load() {
			tc.writeOwedFrames()
		}
		err = tc.conn.Close()

		tc.lmu.Lock()
		close(tc.closed)
		untrack := tc.untrack
		tc.lmu.Unlock()
		if untrack != nil {
			untrack()
		}
	})
	return err
}

// CloseWrite lets net/http half-close the connection, as it does with
// a [net.TCPConn].
func (tc *TrackedConn) CloseWrite() error {
	if cw, ok := tc.conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
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
