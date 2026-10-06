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
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// holdServer is an h2c server whose "/hold" handler sends its headers and then
// blocks until released, holding the stream open, and whose other paths answer
// immediately. It counts the TCP connections the client opens and closes, so
// tests can prove the pool really dialed or closed a connection rather than
// just changed its bookkeeping.
//
// Connections are counted at the listener, not through http.Server.ConnState:
// h2c hijacks every connection, so ConnState never reports StateClosed.
type holdServer struct {
	*httptest.Server
	release chan struct{}
	conns   atomic.Int32 // accepted TCP connections
	closed  atomic.Int32 // accepted TCP connections since closed by the server side
	once    sync.Once
}

// countingListener counts connections accepted from it and, via
// countingConn, the ones that are closed.
type countingListener struct {
	net.Listener
	hs *holdServer
}

func (l *countingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	l.hs.conns.Add(1)
	return &countingConn{Conn: c, hs: l.hs}, nil
}

type countingConn struct {
	net.Conn
	hs   *holdServer
	once sync.Once
}

func (c *countingConn) Close() error {
	c.once.Do(func() { c.hs.closed.Add(1) })
	return c.Conn.Close()
}

func newHoldServer(t *testing.T) *holdServer {
	t.Helper()
	hs := &holdServer{release: make(chan struct{})}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hold" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Headers go out immediately, so the client has its response while
		// the stream is still open.
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-hs.release:
		case <-r.Context().Done():
		}
		_, _ = io.WriteString(w, "done")
	})
	hs.Server = httptest.NewUnstartedServer(h2c.NewHandler(handler, &http2.Server{IdleTimeout: defaultIdleConnTimeout}))
	hs.Server.Listener = &countingListener{Listener: hs.Server.Listener, hs: hs}
	hs.Server.Start()
	t.Cleanup(func() {
		hs.releaseAll()
		hs.Server.Close()
	})
	return hs
}

// releaseAll lets every held (and future) "/hold" request finish.
func (hs *holdServer) releaseAll() { hs.once.Do(func() { close(hs.release) }) }

func (hs *holdServer) addr() string { return strings.TrimPrefix(hs.URL, "http://") }

func newScalingTransport(t *testing.T, opts ...TransportOption) *Transport {
	t.Helper()
	tr := NewTransport(opts...)
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })
	return tr
}

// newScalingPeer creates a peer outside the transport's peer map (so no
// MaintainConn probe connections muddy the server's connection counts) and
// registers cleanup that releases it and waits for its pool to tear down.
func newScalingPeer(t *testing.T, tr *Transport, hs *holdServer) *httpPeer {
	t.Helper()
	p := newPeer(hs.addr(), tr)
	t.Cleanup(func() {
		p.Release()
		if pool := p.loadH2Pool(); pool != nil {
			pool.Wait()
		}
	})
	return p
}

// doReq sends a request through the same path a real outbound uses
// (h2PeerSender) and returns once the response headers are in. The caller owns
// the response body.
func doReq(t *testing.T, p *httpPeer, hs *holdServer, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, hs.URL+path, nil)
	require.NoError(t, err)
	resp, err := (&h2PeerSender{peer: p}).Do(req)
	require.NoError(t, err)
	return resp
}

func doHold(t *testing.T, p *httpPeer, hs *holdServer) *http.Response {
	t.Helper()
	return doReq(t, p, hs, "/hold")
}

// finish reads a response to the end and closes it.
func finish(t *testing.T, resp *http.Response) {
	t.Helper()
	_, err := io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
}

func finishAll(t *testing.T, resps []*http.Response) {
	t.Helper()
	for _, resp := range resps {
		finish(t, resp)
	}
}

// poolStreams returns the pool's connection count and its total in-flight
// stream count.
func poolStreams(p *httpPeer) (conns int, streams int32) {
	pool := p.loadH2Pool()
	if pool == nil {
		return 0, 0
	}
	for _, w := range pool.LoadConns() {
		conns++
		streams += w.StreamCount()
	}
	return conns, streams
}

func connCount(p *httpPeer) int {
	c, _ := poolStreams(p)
	return c
}

func activeConns(p *httpPeer) int {
	pool := p.loadH2Pool()
	if pool == nil {
		return 0
	}
	n := 0
	for _, w := range pool.LoadConns() {
		if w.IsActive() {
			n++
		}
	}
	return n
}

var eventuallyWait = 5 * testtime.Second
var eventuallyTick = 10 * testtime.Millisecond

// TestStreamCountHeldUntilBodyIsFinished is the accounting the scaler's
// decisions rest on: a stream is in flight until its response body is done,
// not merely until the headers arrive.
func TestStreamCountHeldUntilBodyIsFinished(t *testing.T) {
	hs := newHoldServer(t)
	tr := newScalingTransport(t)
	p := newScalingPeer(t, tr, hs)

	t.Run("close releases the stream", func(t *testing.T) {
		resp := doHold(t, p, hs)
		_, streams := poolStreams(p)
		assert.EqualValues(t, 1, streams, "headers are in but the body is still streaming")

		hs.releaseAll()
		require.NoError(t, resp.Body.Close())
		_, streams = poolStreams(p)
		assert.EqualValues(t, 0, streams)
	})

	t.Run("reading to EOF releases the stream, and Close afterwards does not release it twice", func(t *testing.T) {
		resp := doHold(t, p, hs) // hs is released by now: this finishes right away
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "done", string(body))
		_, streams := poolStreams(p)
		assert.EqualValues(t, 0, streams, "EOF ends the stream")

		require.NoError(t, resp.Body.Close())
		_, streams = poolStreams(p)
		assert.EqualValues(t, 0, streams, "a second release would drive the count negative")
	})

	t.Run("a failed request releases the stream", func(t *testing.T) {
		pool := p.loadH2Pool()
		require.NotNil(t, pool)
		w := pool.PickConn()
		require.NotNil(t, w)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, hs.URL+"/hold", nil)
		require.NoError(t, err)
		_, err = (&h2ConnSender{wrapper: w, pool: pool}).Do(req)
		require.ErrorIs(t, err, context.Canceled)
		assert.EqualValues(t, 0, w.StreamCount())
		assert.True(t, w.IsActive(), "a request that failed on its own must not evict the connection")
	})
}

func TestScaleUpUnderLoad(t *testing.T) {
	hs := newHoldServer(t)
	// A scale-up threshold of 2 streams (4 * 0.5): the pool grows once every
	// connection carries 2.
	tr := newScalingTransport(t,
		MaxConcurrentStreams(4),
		ScaleUpThreshold(0.5),
		ScaleDownGap(0.1),
		MinConnections(1),
		MaxConnections(2),
	)
	p := newScalingPeer(t, tr, hs)

	open := []*http.Response{doHold(t, p, hs)}
	assert.Equal(t, 1, connCount(p), "one stream is below the threshold")

	open = append(open, doHold(t, p, hs))
	require.Eventually(t, func() bool { return connCount(p) == 2 },
		eventuallyWait, eventuallyTick, "two streams reach the threshold, so the pool dials a second connection")
	assert.EqualValues(t, 2, hs.conns.Load(), "the server saw two distinct TCP connections")

	// New requests go to the least-loaded connection, which is the new one.
	open = append(open, doHold(t, p, hs), doHold(t, p, hs))
	for _, w := range p.loadH2Pool().LoadConns() {
		assert.EqualValues(t, 2, w.StreamCount(), "load is spread across both connections")
	}

	// At MaxConnections, sustained load must not dial a third connection.
	open = append(open, doHold(t, p, hs))
	time.Sleep(50 * testtime.Millisecond)
	assert.Equal(t, 2, connCount(p), "MaxConnections caps the pool")
	assert.EqualValues(t, 2, hs.conns.Load())

	hs.releaseAll()
	finishAll(t, open)
	_, streams := poolStreams(p)
	assert.EqualValues(t, 0, streams)
}

func TestScalingDisabledNeverGrows(t *testing.T) {
	hs := newHoldServer(t)
	tr := newScalingTransport(t,
		WithDynamicConnectionScaling(false),
		MaxConcurrentStreams(2),
		ScaleUpThreshold(0.5),
		MinConnections(3), // ignored while scaling is off: the floor is 1
	)
	p := newScalingPeer(t, tr, hs)

	var open []*http.Response
	for range 5 {
		open = append(open, doHold(t, p, hs))
	}
	time.Sleep(50 * testtime.Millisecond)
	conns, streams := poolStreams(p)
	assert.Equal(t, 1, conns, "scaling off: one connection regardless of load or MinConnections")
	assert.EqualValues(t, 5, streams)
	assert.EqualValues(t, 1, hs.conns.Load())

	hs.releaseAll()
	finishAll(t, open)
}

func TestMinConnectionsFilledAfterFirstDial(t *testing.T) {
	hs := newHoldServer(t)
	tr := newScalingTransport(t, MinConnections(3), MaxConnections(5))
	p := newScalingPeer(t, tr, hs)

	finish(t, doReq(t, p, hs, "/"))
	require.Eventually(t, func() bool { return activeConns(p) == 3 },
		eventuallyWait, eventuallyTick, "the monitor's first pass fills the pool up to MinConnections")
	assert.EqualValues(t, 3, hs.conns.Load(), "exactly MinConnections were dialed, with no duplicate first dial")
}

func TestScaleDownThenIdleClose(t *testing.T) {
	hs := newHoldServer(t)
	tr := newScalingTransport(t,
		MaxConcurrentStreams(4),
		ScaleUpThreshold(0.5),
		ScaleDownGap(0.1),
		MaxConnections(2),
		ConnIdleTimeout(time.Millisecond),
	)
	p := newScalingPeer(t, tr, hs)

	a := doHold(t, p, hs)
	b := doHold(t, p, hs)
	require.Eventually(t, func() bool { return connCount(p) == 2 }, eventuallyWait, eventuallyTick)

	// While both connections are loaded the pool must not shrink: the
	// remaining connection could not absorb the load inside the hysteresis band.
	pool := p.loadH2Pool()
	pool.EvaluateScaling()
	assert.Equal(t, 2, activeConns(p), "load does not fit on one connection, so nothing drains")

	hs.releaseAll()
	finish(t, a)
	finish(t, b)

	// Idle load: the next pass drains one connection (scale-down).
	pool.EvaluateScaling()
	assert.Equal(t, 1, activeConns(p), "one connection is marked draining")
	// The following pass advances draining -> idle, and a later one closes it
	// once the idle timeout has passed.
	pool.EvaluateScaling()
	time.Sleep(5 * testtime.Millisecond)
	pool.EvaluateScaling()
	require.Eventually(t, func() bool { return connCount(p) == 1 },
		eventuallyWait, eventuallyTick, "the drained connection is closed and removed after the idle timeout")
	require.Eventually(t, func() bool { return hs.closed.Load() == 1 },
		eventuallyWait, eventuallyTick, "the server observes the closed TCP connection")
	assert.Equal(t, 1, activeConns(p), "the surviving connection stays active")
}

// TestDrainingConnectionKeepsItsStreamOpen guards the reason streams are
// counted through the body: a connection scaled down while a response is still
// streaming must not be closed under that response.
func TestDrainingConnectionKeepsItsStreamOpen(t *testing.T) {
	hs := newHoldServer(t)
	tr := newScalingTransport(t,
		MaxConcurrentStreams(10),
		ScaleUpThreshold(0.5),
		ScaleDownGap(0.1),
		MaxConnections(2),
		ConnIdleTimeout(time.Millisecond),
	)
	p := newScalingPeer(t, tr, hs)

	streaming := doHold(t, p, hs)
	pool := p.loadH2Pool()
	busy := pool.LoadConns()[0]
	_, err := pool.AddConn() // a second, idle connection
	require.NoError(t, err)

	// One stream fits easily on one connection, so a pass drains a connection:
	// the most loaded one, which is the busy connection.
	pool.EvaluateScaling()
	require.Equal(t, 1, activeConns(p))
	require.False(t, busy.IsActive(), "the busy connection was the one marked draining")

	// With the idle timeout long expired, repeated passes must still leave the
	// draining connection alone, because its stream count is not zero.
	for range 4 {
		time.Sleep(3 * testtime.Millisecond)
		pool.EvaluateScaling()
	}
	assert.EqualValues(t, 0, hs.closed.Load(), "no TCP connection may be closed while a stream is open on it")
	assert.EqualValues(t, 1, busy.StreamCount())

	hs.releaseAll()
	body, err := io.ReadAll(streaming.Body)
	require.NoError(t, err, "the open stream completes even though its connection was scaled down around it")
	assert.Equal(t, "done", string(body))
	require.NoError(t, streaming.Body.Close())

	// Only now that it is drained does the connection go idle and close.
	pool.EvaluateScaling()
	time.Sleep(5 * testtime.Millisecond)
	pool.EvaluateScaling()
	require.Eventually(t, func() bool { return connCount(p) == 1 && hs.closed.Load() == 1 },
		eventuallyWait, eventuallyTick)
}

func TestLiveProviderTogglesScaling(t *testing.T) {
	hs := newHoldServer(t)
	var enabled atomic.Pointer[bool]
	off := false
	enabled.Store(&off)
	tr := newScalingTransport(t,
		MaxConcurrentStreams(2),
		ScaleUpThreshold(0.5),
		MaxConnections(3),
		WithGlobalLiveConnectionPoolProvider(func() ClientConnectionPoolConfig {
			return ClientConnectionPoolConfig{DynamicScalingEnabled: enabled.Load()}
		}),
	)
	p := newScalingPeer(t, tr, hs)

	open := []*http.Response{doHold(t, p, hs), doHold(t, p, hs), doHold(t, p, hs)}
	time.Sleep(50 * testtime.Millisecond)
	assert.Equal(t, 1, connCount(p), "the live provider has scaling off")

	on := true
	enabled.Store(&on)
	open = append(open, doHold(t, p, hs))
	require.Eventually(t, func() bool { return connCount(p) >= 2 },
		eventuallyWait, eventuallyTick, "flipping the live flag on lets the next request scale up")

	hs.releaseAll()
	finishAll(t, open)
}

func TestReleasedPeerPoolIsTornDownByTransportStop(t *testing.T) {
	hs := newHoldServer(t)
	hs.releaseAll()
	tr := NewTransport()
	require.NoError(t, tr.Start())

	id := testIdentifier{hs.addr()}
	sub := idSubscriber{1}
	pi, err := tr.RetainPeer(id, sub)
	require.NoError(t, err)
	p := pi.(*httpPeer)

	finish(t, doHold(t, p, hs))
	pool := p.loadH2Pool()
	require.NotNil(t, pool)
	require.NotNil(t, pool.PickConn())

	require.NoError(t, tr.ReleasePeer(id, sub))
	require.NoError(t, tr.Stop())

	// Stop joined the released peer's pool, so it is fully torn down already.
	assert.Nil(t, pool.PickConn())
	assert.Empty(t, pool.LoadConns())
}

// TestPoolCreatedAfterReleaseIsStopped covers a request racing a release: the
// pool it lazily creates must not outlive the released peer.
func TestPoolCreatedAfterReleaseIsStopped(t *testing.T) {
	hs := newHoldServer(t)
	hs.releaseAll()
	tr := newScalingTransport(t)
	p := newPeer(hs.addr(), tr)

	p.Release() // nothing to stop yet: no request has created the pool

	// The pool is created and stopped straight away. The dial may or may not
	// beat that stop, so the error is not asserted -- only that the pool is
	// stopped and finishes tearing down, which Wait would hang on otherwise.
	_, _ = p.h2Sender()
	pool := p.loadH2Pool()
	require.NotNil(t, pool)
	pool.Wait()
}

// TestMinConnectionsSurviveHTTPIdleConnTimeout: the pool, not the HTTP/2
// client, decides when a connection is idle enough to close, and it never
// closes below MinConnections. A connection that http2 closed on its own idle
// timer would drop the pool below its floor until the monitor re-dialed.
func TestMinConnectionsSurviveHTTPIdleConnTimeout(t *testing.T) {
	hs := newHoldServer(t)
	tr := newScalingTransport(t,
		MinConnections(2),
		MaxConnections(2),
		IdleConnTimeout(20*time.Millisecond),
	)
	p := newScalingPeer(t, tr, hs)

	finish(t, doReq(t, p, hs, "/"))
	require.Eventually(t, func() bool { return activeConns(p) == 2 }, eventuallyWait, eventuallyTick)

	time.Sleep(300 * testtime.Millisecond) // many multiples of the idle timeout
	assert.Equal(t, 2, activeConns(p), "idle connections at the floor must stay open")
	assert.EqualValues(t, 0, hs.closed.Load(), "no connection may have been closed")
	assert.EqualValues(t, 2, hs.conns.Load(), "and none re-dialed")
}
