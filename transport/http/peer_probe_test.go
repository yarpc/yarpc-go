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
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// newProbeTransport returns a Transport configured for fast probe tests.
// connTimeout is kept short so failure-path tests don't block long.
// Idle connections are closed at test end to avoid goroutine leaks from
// the HTTP/2 client read loops that outlive the test server.
func newProbeTransport(t *testing.T, opts ...TransportOption) *Transport {
	t.Helper()
	opts = append([]TransportOption{ConnTimeout(200 * time.Millisecond)}, opts...)
	tr := NewTransport(opts...)
	t.Cleanup(func() {
		tr.h1Transport.CloseIdleConnections()
		tr.h2Transport.CloseIdleConnections()
	})
	return tr
}

// newProbePeer creates an httpPeer without spawning MaintainConn.
// It calls newPeer directly (package-internal constructor) which does not
// start any background goroutine — only getOrCreatePeer does that.
func newProbePeer(t *testing.T, addr string, opts ...TransportOption) *httpPeer {
	t.Helper()
	return newPeer(addr, newProbeTransport(t, opts...))
}

// newH2CTestServer starts an H2C (HTTP/2 cleartext) server that returns 200
// for any request. Closed automatically when the test ends.
func newH2CTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	h2s := &http2.Server{IdleTimeout: 15 * time.Minute}
	srv := httptest.NewServer(h2c.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		h2s,
	))
	t.Cleanup(srv.Close)
	return srv
}

// listenThenClose opens a TCP listener and immediately closes it, returning
// the address. Used to obtain a port that is guaranteed to be free by the
// time the test needs it to be unresponsive.
func listenThenClose(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

// ── H2Probing() option ────────────────────────────────────────────────────────

func TestH2ProbingOption(t *testing.T) {
	tr := newProbeTransport(t, H2Probing())
	assert.True(t, tr.useH2Probe)
}

func TestDefaultTransport_H2ProbeDisabled(t *testing.T) {
	tr := newProbeTransport(t)
	assert.False(t, tr.useH2Probe)
}

// ── probeTCP ─────────────────────────────────────────────────────────────────

func TestProbeTCP_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	p := newProbePeer(t, ln.Addr().String())
	assert.NoError(t, p.probeTCP())
}

func TestProbeTCP_Failure(t *testing.T) {
	p := newProbePeer(t, listenThenClose(t))
	assert.Error(t, p.probeTCP())
}

// ── probeH2 ──────────────────────────────────────────────────────────────────

func TestProbeH2_Success(t *testing.T) {
	srv := newH2CTestServer(t)
	addr := strings.TrimPrefix(srv.URL, "http://")

	p := newProbePeer(t, addr)
	assert.NoError(t, p.probeH2())
}

func TestProbeH2_Failure(t *testing.T) {
	p := newProbePeer(t, listenThenClose(t))
	assert.Error(t, p.probeH2())
}

// ── isAvailable dispatch ──────────────────────────────────────────────────────

func TestIsAvailable_TCPProbe_Available(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	// useH2Probe=false (default) → probeTCP path
	p := newProbePeer(t, ln.Addr().String())
	assert.True(t, p.isAvailable())
}

func TestIsAvailable_TCPProbe_Unavailable(t *testing.T) {
	// useH2Probe=false → probeTCP fails → isAvailable returns false
	p := newProbePeer(t, listenThenClose(t))
	assert.False(t, p.isAvailable())
}

func TestIsAvailable_H2Probe_Available(t *testing.T) {
	srv := newH2CTestServer(t)
	addr := strings.TrimPrefix(srv.URL, "http://")

	// useH2Probe=true → probeH2 path
	p := newProbePeer(t, addr, H2Probing())
	assert.True(t, p.isAvailable())
}

func TestIsAvailable_H2Probe_Unavailable(t *testing.T) {
	// useH2Probe=true → probeH2 fails → isAvailable returns false + logs debug
	p := newProbePeer(t, listenThenClose(t), H2Probing())
	assert.False(t, p.isAvailable())
}
