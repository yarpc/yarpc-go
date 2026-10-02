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
	"errors"
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
	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// *http.Client do more than what a RoundTrip is supposed to do:
// - It tries to handle higher-level protocol details such as redirects, authentications, or cookies.
// - It requires a client request(requestURI can't be set) so it is not possible to proxy a server request transparently.
// We want to make sure transportSender can proxy server requests transparently.
func TestSender(t *testing.T) {
	const data = "dummy server response body"

	var (
		server = httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				io.WriteString(w, data)
			},
		))
		clientReq, _ = http.NewRequest("GET", server.URL, nil)
		serverReq    = httptest.NewRequest("GET", server.URL, nil)
		client       = &http.Client{
			Transport: http.DefaultTransport,
		}
	)
	defer server.Close()

	tests := []struct {
		msg            string
		sender         sender
		req            *http.Request
		wantStatusCode int
		wantBody       string
		wantError      string
	}{
		{
			msg:            "http.Client sender, http client request",
			req:            clientReq,
			sender:         http.DefaultClient,
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
		{
			msg:       "http.Client sender, http server request",
			req:       serverReq,
			sender:    http.DefaultClient,
			wantError: "http: Request.RequestURI can't be set in client requests",
		},
		{
			msg:            "transportSender, http client request",
			req:            clientReq,
			sender:         &transportSender{Client: client},
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
		{
			msg:            "transportSender, http server request",
			req:            serverReq,
			sender:         &transportSender{Client: client},
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			resp, err := tt.sender.Do(tt.req)
			if tt.wantError != "" {
				require.Error(t, err, "expect error when we use http.Client to send a server request")
				assert.Contains(t, err.Error(), tt.wantError, "error body mismatch")
				return
			}
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode, "status code does not match")
			body, _ := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantBody, string(body), "response body does not match")
		})
	}
}

func TestIsRetryableH2Error(t *testing.T) {
	tests := []struct {
		msg  string
		err  error
		want bool
	}{
		{
			msg:  "refused stream",
			err:  http2.StreamError{Code: http2.ErrCodeRefusedStream},
			want: true,
		},
		{
			msg:  "other stream error code",
			err:  http2.StreamError{Code: http2.ErrCodeProtocol},
			want: false,
		},
		{
			msg:  "client conn unusable",
			err:  errors.New(h2ErrClientConnUnusable),
			want: true,
		},
		{
			msg:  "client conn got goaway",
			err:  errors.New(h2ErrClientConnGotGoAway),
			want: true,
		},
		{
			msg:  "unrelated error",
			err:  errors.New("boom"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetryableH2Error(tt.err))
		})
	}
}

func TestNextH2Request(t *testing.T) {
	unusableErr := errors.New(h2ErrClientConnUnusable)
	goAwayErr := errors.New(h2ErrClientConnGotGoAway)
	notRetryableErr := errors.New("boom")

	t.Run("not a retryable error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		got, retryable := nextH2Request(req, notRetryableErr)
		assert.False(t, retryable)
		assert.Nil(t, got)
	})

	t.Run("nil body replays the same request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		req.Body = nil
		got, retryable := nextH2Request(req, unusableErr)
		assert.True(t, retryable)
		assert.Same(t, req, got)
	})

	t.Run("http.NoBody replays the same request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		req.Body = http.NoBody
		got, retryable := nextH2Request(req, unusableErr)
		assert.True(t, retryable)
		assert.Same(t, req, got)
	})

	t.Run("GetBody succeeds rebuilds the request body", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("payload"))
		require.NoError(t, err)
		require.NotNil(t, req.GetBody, "http.NewRequest should set GetBody for a strings.Reader body")

		got, retryable := nextH2Request(req, unusableErr)
		assert.True(t, retryable)
		require.NotNil(t, got)
		assert.NotSame(t, req, got, "a fresh request should be returned, not the original")

		body, err := io.ReadAll(got.Body)
		require.NoError(t, err)
		assert.Equal(t, "payload", string(body))
	})

	t.Run("GetBody failing is not retryable", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("payload"))
		require.NoError(t, err)
		req.GetBody = func() (io.ReadCloser, error) { return nil, errors.New("cannot rebuild body") }

		got, retryable := nextH2Request(req, unusableErr)
		assert.False(t, retryable)
		assert.Nil(t, got)
	})

	t.Run("body with no GetBody replays unmodified only for an unusable conn", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("payload"))
		require.NoError(t, err)
		req.GetBody = nil

		got, retryable := nextH2Request(req, unusableErr)
		assert.True(t, retryable)
		assert.Same(t, req, got)
	})

	t.Run("body with no GetBody is not retryable for other retryable errors", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("payload"))
		require.NoError(t, err)
		req.GetBody = nil

		got, retryable := nextH2Request(req, goAwayErr)
		assert.False(t, retryable)
		assert.Nil(t, got)
	})
}

// newH2CServer starts an h2c (unencrypted HTTP/2) test server serving
// handler. connState, if non-nil, observes the server's connection lifecycle
// (e.g. to count dials).
func newH2CServer(handler http.Handler, connState func(net.Conn, http.ConnState)) *httptest.Server {
	server := httptest.NewUnstartedServer(h2c.NewHandler(handler, &http2.Server{IdleTimeout: defaultIdleConnTimeout}))
	server.Config.ConnState = connState
	server.Start()
	return server
}

// newH2TestServer is newH2CServer with a handler that responds 200 OK to
// every request.
func newH2TestServer(connState func(net.Conn, http.ConnState)) *httptest.Server {
	return newH2CServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	), connState)
}

func TestH2ConnSenderDo(t *testing.T) {
	server := newH2TestServer(nil)
	defer server.Close()

	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	addr := strings.TrimPrefix(server.URL, "http://")
	pool := connpool.NewPool(
		context.Background(),
		func() connpool.Config { return connpool.Config{MinConnections: 1, MaxConnections: 1} },
		func(ctx context.Context) (*http2.ClientConn, error) { return tr.dialH2Conn(ctx, addr) },
		zap.NewNop(),
		addr,
		nil,
	)
	// AddConn always reserves a connWg slot for a watcher to release via
	// ConnDone (see connpool.Pool.OnConnAdded's doc comment); this mirrors
	// httpPeer.watchH2Conn's teardown-only half so pool.Wait() below
	// doesn't hang on the connections this test adds directly. AddConn runs
	// OnConnAdded in its own goroutine, so it can block here.
	pool.OnConnAdded = func(w *connpool.Wrapper[*http2.ClientConn]) {
		<-w.Context().Done()
		w.Conn.Close()
		pool.Remove(w)
		close(w.StoppedC)
		pool.ConnDone()
	}
	require.NoError(t, pool.Start(0, false))
	t.Cleanup(func() { pool.Stop(); pool.Wait() })

	t.Run("successful RoundTrip leaves the connection active", func(t *testing.T) {
		w, err := pool.AddConn()
		require.NoError(t, err)
		defer w.Cancel()

		sender := &h2ConnSender{wrapper: w}
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := sender.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, resp.Body.Close())

		assert.EqualValues(t, 0, w.StreamCount(), "stream count should be decremented after Do returns")
		assert.True(t, w.IsActive(), "a successful RoundTrip must not evict the connection")
	})

	t.Run("a failed request on a healthy connection leaves it active", func(t *testing.T) {
		w, err := pool.AddConn()
		require.NoError(t, err)
		defer w.Cancel()

		sender := &h2ConnSender{wrapper: w}
		// An already-cancelled request context makes RoundTrip fail fast and
		// deterministically. The failure belongs to the request, not the
		// connection, so the connection must stay in service.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		_, err = sender.Do(req)
		require.Error(t, err)
		assert.EqualValues(t, 0, w.StreamCount())
		assert.True(t, w.IsActive(), "a request-level failure must not evict the connection")
		assert.NoError(t, w.Context().Err(), "a request-level failure must not cancel the connection")
	})

	t.Run("a failure on a broken connection evicts it", func(t *testing.T) {
		w, err := pool.AddConn()
		require.NoError(t, err)
		defer w.Cancel()

		sender := &h2ConnSender{wrapper: w}
		// Closing the underlying connection makes it unusable, so the next
		// RoundTrip fails because of the connection itself.
		require.NoError(t, w.Conn.Close())
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		_, err = sender.Do(req)
		require.Error(t, err)
		assert.EqualValues(t, 0, w.StreamCount())
		assert.Equal(t, connpool.StateDraining, w.GetState(),
			"a failure on a closed connection should transition it to draining")
		select {
		case <-w.Context().Done():
		default:
			t.Fatal("a failure on a closed connection should cancel the wrapper's context")
		}
	})
}

// TestH2PeerSenderNoConnectionAvailable covers h2PeerSender.Do's error path:
// when the peer's HTTP/2 pool can't dial (here, nothing listens on the
// target address), h2PeerSender must wrap the error as CodeUnavailable
// rather than propagating the raw dial error.
func TestH2PeerSenderNoConnectionAvailable(t *testing.T) {
	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	p := mustNewPeer(t, addr, tr)
	// h2Sender's cold path still creates and starts a pool before its dial
	// fails (see h2Sender in peer.go) -- this peer was never registered
	// with the Transport (no RetainPeer/getOrCreatePeer), so nothing else
	// would stop that pool.
	t.Cleanup(func() {
		if pool := p.loadH2Pool(); pool != nil {
			pool.Stop()
			pool.Wait()
		}
	})
	sender := &h2PeerSender{peer: p}

	req, err := http.NewRequest(http.MethodGet, "http://"+addr, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	_, err = sender.Do(req.WithContext(ctx))
	require.Error(t, err)
	assert.Equal(t, yarpcerrors.CodeUnavailable, yarpcerrors.FromError(err).Code())
}

// newTestH2Pool builds a pool of HTTP/2 connections to addr whose per-
// connection watcher only does the teardown half of httpPeer.watchH2Conn, so
// the tests below control connection health themselves.
func newTestH2Pool(t *testing.T, tr *Transport, addr string) *connpool.Pool[*http2.ClientConn] {
	t.Helper()
	pool := connpool.NewPool(
		context.Background(),
		func() connpool.Config { return connpool.Config{MinConnections: 1, MaxConnections: 1} },
		func(ctx context.Context) (*http2.ClientConn, error) { return tr.dialH2Conn(ctx, addr) },
		zap.NewNop(),
		addr,
		nil,
	)
	pool.OnConnAdded = func(w *connpool.Wrapper[*http2.ClientConn]) {
		<-w.Context().Done()
		_ = w.Conn.Close()
		pool.Remove(w)
		close(w.StoppedC)
		pool.ConnDone()
	}
	require.NoError(t, pool.Start(0, false))
	t.Cleanup(func() { pool.Stop(); pool.Wait() })
	return pool
}

// h2InFlightServer serves "/slow", which blocks until released, "/hang",
// which blocks until the client abandons the request, and 200 OK for any
// other path. It counts the connections it accepts.
type h2InFlightServer struct {
	*httptest.Server
	slowStarted chan struct{}
	conns       atomic.Int32
	releaseOnce sync.Once
	release     chan struct{}
}

func newH2InFlightServer(t *testing.T) *h2InFlightServer {
	t.Helper()
	s := &h2InFlightServer{
		slowStarted: make(chan struct{}, 1),
		release:     make(chan struct{}),
	}
	s.Server = newH2CServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow":
			s.slowStarted <- struct{}{}
			<-s.release
		case "/hang":
			<-r.Context().Done()
			return
		}
		w.WriteHeader(http.StatusOK)
	}), func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			s.conns.Add(1)
		}
	})
	t.Cleanup(func() {
		s.releaseSlow()
		s.Server.Close()
	})
	return s
}

func (s *h2InFlightServer) releaseSlow() { s.releaseOnce.Do(func() { close(s.release) }) }

type h2Result struct {
	resp *http.Response
	err  error
}

// doAsync sends a GET for path through sender in the background.
func doAsync(t *testing.T, sender sender, url string) <-chan h2Result {
	t.Helper()
	out := make(chan h2Result, 1)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	go func() {
		resp, err := sender.Do(req)
		out <- h2Result{resp, err}
	}()
	return out
}

// TestH2ConnSenderInFlightRequestSurvivesAnotherRequestFailing checks that a
// request failing on its own (here, hitting its deadline) does not disturb a
// different request that is in flight on the same connection, and that the
// connection keeps serving afterwards without a re-dial.
func TestH2ConnSenderInFlightRequestSurvivesAnotherRequestFailing(t *testing.T) {
	server := newH2InFlightServer(t)
	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	pool := newTestH2Pool(t, tr, strings.TrimPrefix(server.URL, "http://"))
	w, err := pool.AddConn()
	require.NoError(t, err)
	sender := &h2ConnSender{wrapper: w}

	inFlight := doAsync(t, sender, server.URL+"/slow")
	<-server.slowStarted

	ctx, cancel := context.WithTimeout(context.Background(), 100*testtime.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/hang", nil)
	require.NoError(t, err)
	_, err = sender.Do(req)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	assert.True(t, w.IsActive(), "the connection must stay in service")
	assert.NoError(t, w.Context().Err())
	assert.EqualValues(t, 1, w.StreamCount(), "only the in-flight request should still be counted")

	server.releaseSlow()
	res := <-inFlight
	require.NoError(t, res.err, "the in-flight request must not be aborted by the other request's failure")
	assert.Equal(t, http.StatusOK, res.resp.StatusCode)
	require.NoError(t, res.resp.Body.Close())

	reuse := <-doAsync(t, sender, server.URL+"/")
	require.NoError(t, reuse.err)
	require.NoError(t, reuse.resp.Body.Close())
	assert.EqualValues(t, 1, server.conns.Load(), "everything should have used the one connection")
}

// TestH2ConnSenderBrokenConnectionFailsInFlightRequests is the counterpart:
// when the connection itself breaks, requests in flight on it fail, and the
// connection is evicted so no new request is sent on it.
func TestH2ConnSenderBrokenConnectionFailsInFlightRequests(t *testing.T) {
	server := newH2InFlightServer(t)
	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	pool := newTestH2Pool(t, tr, strings.TrimPrefix(server.URL, "http://"))
	w, err := pool.AddConn()
	require.NoError(t, err)
	sender := &h2ConnSender{wrapper: w}

	inFlight := doAsync(t, sender, server.URL+"/slow")
	<-server.slowStarted

	require.NoError(t, w.Conn.Close())

	res := <-inFlight
	require.Error(t, res.err, "a request in flight on a broken connection must fail")

	other, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	require.NoError(t, err)
	_, err = sender.Do(other)
	require.Error(t, err)

	assert.Equal(t, connpool.StateDraining, w.GetState())
	assert.Error(t, w.Context().Err(), "the broken connection must be scheduled for removal")
}

func newBackoffTestPeer(t *testing.T, addr string) *httpPeer {
	t.Helper()
	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	p := mustNewPeer(t, addr, tr)
	t.Cleanup(func() {
		if pool := p.loadH2Pool(); pool != nil {
			pool.Stop()
			pool.Wait()
		}
	})
	return p
}

// TestH2PeerSenderRetryBackoff drives h2PeerSender against a server that
// rejects the first N requests with GOAWAY, forcing replays.
func TestH2PeerSenderRetryBackoff(t *testing.T) {
	tests := []struct {
		name         string
		rejects      int32
		wantErr      bool
		wantBackoffs []int
	}{
		{name: "no replay needs no backoff", rejects: 0, wantBackoffs: nil},
		{name: "the first replay is immediate", rejects: 1, wantBackoffs: nil},
		{name: "later replays back off", rejects: 3, wantBackoffs: []int{1, 2}},
		{name: "gives up after the retry limit", rejects: -1, wantErr: true, wantBackoffs: []int{1, 2, 3, 4, 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newH2CGoAwayServer(t, tt.rejects)
			p := newBackoffTestPeer(t, server.addr())

			var backoffs []int
			sender := &h2PeerSender{peer: p, retryBackoff: func(retry int) time.Duration {
				backoffs = append(backoffs, retry)
				return time.Millisecond
			}}
			req, err := http.NewRequest(http.MethodGet, "http://"+server.addr(), nil)
			require.NoError(t, err)

			resp, err := sender.Do(req)
			if tt.wantErr {
				require.Error(t, err)
				assert.NotErrorIs(t, err, context.DeadlineExceeded)
			} else {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				require.NoError(t, resp.Body.Close())
			}
			assert.Equal(t, tt.wantBackoffs, backoffs)
		})
	}
}

// TestH2PeerSenderBackoffEndsWithRequestContext checks that a request is not
// held for the whole backoff once its context is done.
func TestH2PeerSenderBackoffEndsWithRequestContext(t *testing.T) {
	server := newH2CGoAwayServer(t, -1)
	p := newBackoffTestPeer(t, server.addr())
	sender := &h2PeerSender{peer: p, retryBackoff: func(int) time.Duration { return time.Hour }}

	ctx, cancel := context.WithTimeout(context.Background(), 300*testtime.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+server.addr(), nil)
	require.NoError(t, err)

	start := time.Now()
	_, err = sender.Do(req)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 10*time.Second, "must not wait out the backoff")
}

func TestH2RetryBackoff(t *testing.T) {
	// 1s, 2s, 4s, ... each with up to 10% jitter.
	for retry, base := range map[int]time.Duration{1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second, 5: 16 * time.Second} {
		for range 50 {
			got := h2RetryBackoff(retry)
			assert.GreaterOrEqual(t, got, base, "retry %d", retry)
			assert.LessOrEqual(t, got, base+base/10, "retry %d", retry)
		}
	}
}

func TestSleepContext(t *testing.T) {
	t.Run("waits the full duration", func(t *testing.T) {
		start := time.Now()
		require.NoError(t, sleepContext(context.Background(), 20*time.Millisecond))
		assert.GreaterOrEqual(t, time.Since(start), 20*time.Millisecond)
	})
	t.Run("returns early with the context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.ErrorIs(t, sleepContext(ctx, time.Hour), context.Canceled)
	})
}
