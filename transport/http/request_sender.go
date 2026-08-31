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
	"net/http"

	"go.uber.org/yarpc/transport/internal/connpool"
	"go.uber.org/yarpc/yarpcerrors"
	"golang.org/x/net/http2"
)

var (
	_ sender = (*http.Client)(nil)
	_ sender = (*transportSender)(nil)
	_ sender = (*h2ConnSender)(nil)
	_ sender = (*h2PeerSender)(nil)
)

type sender interface {
	Do(*http.Request) (*http.Response, error)
}

type transportSender struct {
	// This avoids calling http.Client.Do on the same HTTP request twice.
	*http.Client
}

func (t *transportSender) Do(req *http.Request) (*http.Response, error) {
	return t.Client.Transport.RoundTrip(req)
}

// h2ConnSender routes a request through one specific pooled HTTP/2
// connection. Calling wrapper.Conn.RoundTrip directly (rather than via
// http.Client.Do) means requests never get automatic redirect-following or
// http.Client's extra error wrapping, matching transportSender's semantics
// for the HTTP/1 path above.
type h2ConnSender struct {
	wrapper *connpool.Wrapper[*http2.ClientConn]
}

func (s *h2ConnSender) Do(req *http.Request) (*http.Response, error) {
	s.wrapper.IncStreamCount()
	defer s.wrapper.DecStreamCount()

	resp, err := s.wrapper.Conn.RoundTrip(req)
	if err != nil {
		// Evict a broken connection immediately rather than waiting for the
		// next health-poll tick (see httpPeer.watchH2Conn). Transition state
		// first so a concurrent PickConn can't hand this connection to
		// another request in the window before the pool's watcher goroutine
		// observes Cancel and removes it.
		s.wrapper.TransitionState(connpool.StateActive, connpool.StateDraining)
		s.wrapper.Cancel()
	}
	return resp, err
}

// maxH2RetryAttempts bounds replay attempts for a request that failed for a
// reason net/http2's own *http2.Transport would have retried internally on
// a fresh connection (see (*http2.Transport).roundTripViaPool). It mirrors
// that function's own retry limit.
const maxH2RetryAttempts = 6

// h2PeerSender routes a request through httpPeer's HTTP/2 connection pool,
// picking (or lazily dialing) a connection via h2Sender and replaying the
// request on a different connection when it fails for a reason
// *http2.Transport would itself have retried transparently -- a GOAWAY, an
// otherwise-unusable connection, or a refused stream. Talking to a specific
// *http2.ClientConn directly (h2ConnSender), instead of routing every
// request through *http2.Transport's own pool, otherwise loses that
// built-in replay entirely, since (*http2.ClientConn).RoundTrip never
// retries on its own.
type h2PeerSender struct {
	peer *httpPeer
}

func (s *h2PeerSender) Do(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		connSender, err := s.peer.h2Sender()
		if err != nil {
			return nil, yarpcerrors.Newf(yarpcerrors.CodeUnavailable,
				"no HTTP/2 connection available for peer %q: %v", s.peer.addr, err)
		}

		resp, err := connSender.Do(req)
		if err == nil || attempt >= maxH2RetryAttempts {
			return resp, err
		}

		nextReq, retryable := nextH2Request(req, err)
		if !retryable {
			return resp, err
		}
		req = nextReq
	}
}

// nextH2Request reports whether err is a reason *http2.Transport would
// itself retry a request on a new connection, and if so, returns the
// request to replay: err's retryability and the request-rebuilding logic
// here mirror golang.org/x/net/http2's own (unexported) canRetryError and
// shouldRetryRequest.
func nextH2Request(req *http.Request, err error) (*http.Request, bool) {
	if !isRetryableH2Error(err) {
		return nil, false
	}

	if req.Body == nil || req.Body == http.NoBody {
		return req, true
	}
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, false
		}
		newReq := new(http.Request)
		*newReq = *req
		newReq.Body = body
		return newReq, true
	}
	// A connection reported unusable before any part of the request was
	// written is safe to replay unmodified, same as net/http2's own
	// errClientConnUnusable special case.
	if err.Error() == h2ErrClientConnUnusable {
		return req, true
	}
	return nil, false
}

// h2ErrClientConnUnusable and h2ErrClientConnGotGoAway mirror the unexported
// sentinel error strings golang.org/x/net/http2 uses for conditions its own
// Transport retries transparently. They're matched by message because the
// package does not export them, but the message is the stable text a caller
// of (*http2.ClientConn).RoundTrip observes for these conditions.
const (
	h2ErrClientConnUnusable  = "http2: client conn not usable"
	h2ErrClientConnGotGoAway = "http2: Transport received Server's graceful shutdown GOAWAY"
)

func isRetryableH2Error(err error) bool {
	if se, ok := err.(http2.StreamError); ok {
		return se.Code == http2.ErrCodeRefusedStream
	}
	switch err.Error() {
	case h2ErrClientConnUnusable, h2ErrClientConnGotGoAway:
		return true
	}
	return false
}
