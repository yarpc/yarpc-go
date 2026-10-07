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
	"bufio"
	"context"
	"net"
	"net/http"
	"sync"

	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// h2cInbound serves HTTP/2 cleartext (h2c) connections for an Inbound, and
// shuts them down gracefully.
//
// h2c.NewHandler hijacks every h2c connection from the http.Server, and
// net/http stops tracking a connection once it is hijacked:
// http.Server.Shutdown neither waits for nor closes it. Shutdown does run the
// HTTP/2 graceful shutdown hook that sends GOAWAY, but in a goroutine it does
// not wait for either (golang/go#26682). Left to net/http, Inbound.Stop would
// return with h2c calls still in flight and their GOAWAY racing the process
// exit. h2cInbound sends the GOAWAY and waits for these connections itself.
//
// TODO: Once the module requires Go 1.24 or later, serve h2c natively by
// enabling UnencryptedHTTP2 in http.Server.Protocols, and delete h2cInbound.
// net/http then serves h2c connections without hijacking them, so Shutdown
// sends them GOAWAY and waits for them like for any other connection.
type h2cInbound struct {
	h2 *http2.Server

	// notifier is never served. http2.ConfigureServer registers h2's
	// graceful shutdown hook on it, so that notifier.Shutdown sends GOAWAY on
	// every h2c connection without also closing the Inbound's listener.
	notifier *http.Server

	mu      sync.Mutex
	active  int                   // h2c connections being served
	conns   map[net.Conn]struct{} // hijacked connections being served
	drained chan struct{}         // if non-nil, closed once active drops to 0
	closed  bool                  // set once drain gave up and closed conns
}

// newH2CInbound builds an h2cInbound configured like base.
func newH2CInbound(base *http.Server) (*h2cInbound, error) {
	s := &h2cInbound{
		h2: &http2.Server{},
		// ConfigureServer derives the HTTP/2 idle timeout from these.
		notifier: &http.Server{
			IdleTimeout: base.IdleTimeout,
			ReadTimeout: base.ReadTimeout,
		},
		conns: make(map[net.Conn]struct{}),
	}
	if err := http2.ConfigureServer(s.notifier, s.h2); err != nil {
		return nil, err
	}
	return s, nil
}

// Handler returns an http.Handler that serves h2c connections, and passes
// any other request to h.
func (s *h2cInbound) Handler(h http.Handler) http.Handler {
	h2cHandler := h2c.NewHandler(h, s.h2)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isH2C(r) {
			h2cHandler.ServeHTTP(w, r)
			return
		}

		// h2cHandler serves the whole connection before returning. Counting it
		// here, before the hijack, while net/http still tracks the connection,
		// ensures http.Server.Shutdown cannot return before it is counted.
		s.begin()
		hw := &h2cHijackRecorder{ResponseWriter: w, s: s}
		defer func() { s.end(hw.conn) }()
		h2cHandler.ServeHTTP(hw, r)
	})
}

// NotifyShutdown sends GOAWAY on every h2c connection. It does not block.
func (s *h2cInbound) NotifyShutdown() {
	// No listener or connection is ever registered on notifier, so this only
	// runs the shutdown hooks, in goroutines, and returns.
	_ = s.notifier.Shutdown(context.Background())
}

// Drain waits for every h2c connection to close. If ctx ends first, it closes
// the remaining connections and returns ctx's error.
func (s *h2cInbound) Drain(ctx context.Context) error {
	s.mu.Lock()
	if s.active == 0 {
		s.mu.Unlock()
		return nil
	}
	if s.drained == nil {
		s.drained = make(chan struct{})
	}
	drained := s.drained
	s.mu.Unlock()

	select {
	case <-drained:
		return nil
	case <-ctx.Done():
	}

	s.mu.Lock()
	s.closed = true
	conns := make([]net.Conn, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()

	for _, c := range conns {
		_ = c.Close()
	}
	return ctx.Err()
}

func (s *h2cInbound) begin() {
	s.mu.Lock()
	s.active++
	s.mu.Unlock()
}

func (s *h2cInbound) hijacked(c net.Conn) {
	s.mu.Lock()
	closed := s.closed
	if !closed {
		s.conns[c] = struct{}{}
	}
	s.mu.Unlock()

	if closed {
		// Drain already gave up on h2c connections; don't serve a new one.
		_ = c.Close()
	}
}

func (s *h2cInbound) end(c net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c != nil {
		delete(s.conns, c)
	}
	s.active--
	if s.active == 0 && s.drained != nil {
		close(s.drained)
		s.drained = nil
	}
}

// h2cHijackRecorder registers the connection that h2c.NewHandler hijacks with
// the h2cInbound, so that Drain can close it.
type h2cHijackRecorder struct {
	http.ResponseWriter

	s    *h2cInbound
	conn net.Conn
}

func (w *h2cHijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, rw, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, err
	}
	w.conn = c
	w.s.hijacked(c)
	return c, rw, nil
}

// isH2C reports whether h2c.NewHandler takes over the connection of r. It
// mirrors the checks h2c.NewHandler makes: HTTP/2 with prior knowledge
// (RFC 7540 §3.4), or an upgrade from HTTP/1.1 (RFC 7540 §3.2).
func isH2C(r *http.Request) bool {
	if r.Method == "PRI" && len(r.Header) == 0 && r.URL.Path == "*" && r.Proto == "HTTP/2.0" {
		return true
	}
	return httpguts.HeaderValuesContainsToken(r.Header["Upgrade"], "h2c") &&
		httpguts.HeaderValuesContainsToken(r.Header["Connection"], "HTTP2-Settings")
}
