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

package http

import (
	"time"

	"go.uber.org/atomic"
	"golang.org/x/net/http2"
)

// http2ConnState describes the lifecycle state of a pooled http2Conn, as
// tracked by http2Pool.
type http2ConnState int32

const (
	// http2ConnActive connections are eligible to be picked for new
	// requests.
	http2ConnActive http2ConnState = iota
	// http2ConnDraining connections are being scaled down: they are no
	// longer picked for new requests, and are closed once idle.
	http2ConnDraining
)

// http2Conn wraps a single *http2.Transport pooled by an http2Pool. A
// *http2.Transport, unlike a raw *http2.ClientConn, never goes permanently
// dead on GOAWAY: its internal client-conn pool transparently redials on the
// next RoundTrip. That self-healing comes at a cost this type works around —
// a *http2.Transport doesn't expose its live stream count or the peer's
// negotiated MaxConcurrentStreams the way http2.ClientConn.State() does, so
// both signals are tracked here instead: inflight is incremented/decremented
// by the caller around each request (the same workaround yarpc's grpc
// transport uses for its own connection pool, since google.golang.org/grpc's
// ClientConn has the identical blind spot), and the concurrency ceiling is a
// fixed value from http2PoolConfig rather than one read off the wire.
type http2Conn struct {
	transport *http2.Transport
	state     atomic.Int32
	inflight  atomic.Int32
	createdAt time.Time

	// idleAtNano is the unix-nano time this connection's inflight count last
	// dropped to zero, or 0 if it is currently serving at least one request.
	// It is a best-effort signal, like http2.ClientConnState.LastIdle: a
	// racing incInflight/decInflight pair can leave it briefly stale, which
	// is acceptable for the idle-timeout heuristic it drives.
	idleAtNano atomic.Int64
}

func newHTTP2Conn(t *http2.Transport) *http2Conn {
	return &http2Conn{
		transport: t,
		createdAt: time.Now(),
	}
}

// usable reports whether this connection may be picked for a new request.
func (c *http2Conn) usable() bool {
	return http2ConnState(c.state.Load()) == http2ConnActive
}

// incInflight records the start of a request dispatched to this connection.
func (c *http2Conn) incInflight() {
	c.inflight.Inc()
	c.idleAtNano.Store(0)
}

// decInflight records the completion of a request dispatched to this
// connection.
func (c *http2Conn) decInflight() {
	if c.inflight.Dec() == 0 {
		c.idleAtNano.Store(time.Now().UnixNano())
	}
}

// streamsActive returns the number of requests this pool has dispatched to
// this connection and not yet completed. Unlike a real HTTP/2 stream count,
// this reflects only what incInflight/decInflight have recorded.
func (c *http2Conn) streamsActive() int {
	return int(c.inflight.Load())
}

// idleSince returns the time this connection's inflight count last reached
// zero, or the zero time if it is currently serving a request.
func (c *http2Conn) idleSince() time.Time {
	ns := c.idleAtNano.Load()
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}

// markDraining marks the connection so it is no longer picked for new
// requests. It does not close the connection or wait for existing streams
// to finish; the pool's monitor loop is responsible for closing it once
// idle.
func (c *http2Conn) markDraining() {
	c.state.Store(int32(http2ConnDraining))
}

func (c *http2Conn) draining() bool {
	return http2ConnState(c.state.Load()) == http2ConnDraining
}

// close releases any connection this Transport currently holds. A
// *http2.Transport has no persistent per-instance resource beyond the
// connections it pools internally, so closing those is sufficient to free
// it; there is no Close method on *http2.Transport itself.
func (c *http2Conn) close() error {
	c.transport.CloseIdleConnections()
	return nil
}
