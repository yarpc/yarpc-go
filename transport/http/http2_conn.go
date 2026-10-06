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
	"io"
	"net/http"
	"sync"
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
	// http2ConnParked connections have been scaled down: they are no longer
	// picked for new requests, but stay in the pool and are re-activated when
	// the pool needs to scale back up. Requests already in flight on a parked
	// connection are left to finish.
	http2ConnParked
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
// fixed value from the pool's connpool.Config rather than one read off the wire.
type http2Conn struct {
	transport *http2.Transport
	state     atomic.Int32
	inflight  atomic.Int32
	createdAt time.Time

	// closing is set once the owning pool has been closed. It makes the
	// connection release its sockets as soon as it is idle: decInflight closes
	// them when the last in-flight request finishes, instead of leaving them
	// open until the Transport's IdleConnTimeout.
	closing atomic.Bool

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
		if c.closing.Load() {
			c.closeIdle()
		}
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

// park moves an active connection to the parked state so it is no longer
// picked for new requests. It reports whether this call made the transition.
// It does not close the connection or wait for existing streams to finish.
func (c *http2Conn) park() bool {
	return c.state.CompareAndSwap(int32(http2ConnActive), int32(http2ConnParked))
}

// unpark moves a parked connection back to the active state. It reports
// whether this call made the transition, so concurrent callers never
// re-activate the same connection twice.
func (c *http2Conn) unpark() bool {
	return c.state.CompareAndSwap(int32(http2ConnParked), int32(http2ConnActive))
}

func (c *http2Conn) parked() bool {
	return http2ConnState(c.state.Load()) == http2ConnParked
}

// closeIdle closes the sockets this Transport currently holds idle. A
// *http2.Transport has no Close method, so this is how its connections are
// released. It does not make the http2Conn unusable: the Transport simply
// redials on its next request.
func (c *http2Conn) closeIdle() {
	c.transport.CloseIdleConnections()
}

// shutdown is called when the owning pool is closed. It closes the sockets
// that are idle now, and arranges for the ones that are busy to be closed by
// decInflight as soon as their last request finishes.
func (c *http2Conn) shutdown() {
	c.closing.Store(true)
	c.closeIdle()
}

// releaseOnBodyClose arranges for this connection's in-flight slot to be
// released when resp.Body is closed, rather than when RoundTrip returns: an
// HTTP/2 stream stays open until its body is finished, so releasing any
// earlier would undercount the streams the connection is really carrying. It
// reports whether it took over responsibility for the release; if resp has no
// body to close, it returns false and the caller must release the slot.
func (c *http2Conn) releaseOnBodyClose(resp *http.Response) bool {
	if resp.Body == nil || resp.Body == http.NoBody {
		return false
	}
	resp.Body = &inflightBody{ReadCloser: resp.Body, release: c.decInflight}
	return true
}

// inflightBody calls release exactly once, on the first Close.
type inflightBody struct {
	io.ReadCloser
	release func()
	once    sync.Once
}

func (b *inflightBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(b.release)
	return err
}
