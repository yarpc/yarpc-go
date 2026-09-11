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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
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

// newH2TestServer starts an h2c (unencrypted HTTP/2) test server that
// responds 200 OK to every request.
func newH2TestServer() *httptest.Server {
	server := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))
	server.Config.Protocols = new(http.Protocols)
	server.Config.Protocols.SetHTTP1(true)
	server.Config.Protocols.SetUnencryptedHTTP2(true)
	http2.ConfigureServer(server.Config, &http2.Server{IdleTimeout: defaultIdleConnTimeout})
	server.Start()
	return server
}

func TestH2ConnSenderDo(t *testing.T) {
	server := newH2TestServer()
	defer server.Close()

	tr := NewTransport()
	require.NoError(t, tr.Start())
	t.Cleanup(func() { assert.NoError(t, tr.Stop()) })

	addr := strings.TrimPrefix(server.URL, "http://")
	pool := connpool.NewPool(
		context.Background(),
		connpool.Config{MinConnections: 1, MaxConnections: 1},
		func(ctx context.Context) (*http2.ClientConn, error) { return tr.dialH2Conn(ctx, addr) },
		zap.NewNop(),
		addr,
		nil,
	)
	// AddConn always reserves a connWg slot for a watcher to release via
	// ConnDone (see connpool.Pool.OnConnAdded's doc comment); this mirrors
	// httpPeer.watchH2Conn's teardown-only half so pool.Wait() below
	// doesn't hang on the connections this test adds directly.
	pool.OnConnAdded = func(w *connpool.Wrapper[*http2.ClientConn]) {
		go func() {
			<-w.Context().Done()
			w.Conn.Close()
			pool.Remove(w)
			pool.ConnDone()
		}()
	}
	require.NoError(t, pool.Start(0))
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

	t.Run("RoundTrip error evicts the connection", func(t *testing.T) {
		w, err := pool.AddConn()
		require.NoError(t, err)
		defer w.Cancel()

		sender := &h2ConnSender{wrapper: w}
		// An already-cancelled request context forces RoundTrip to fail
		// fast and deterministically, without relying on real network
		// flakiness to produce an error.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		_, err = sender.Do(req)
		require.Error(t, err)
		assert.EqualValues(t, 0, w.StreamCount())
		assert.Equal(t, connpool.StateDraining, w.GetState(),
			"a failed RoundTrip should transition the connection to draining")
		select {
		case <-w.Context().Done():
		default:
			t.Fatal("a failed RoundTrip should cancel the wrapper's context")
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

	p := newPeer(addr, tr)
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
