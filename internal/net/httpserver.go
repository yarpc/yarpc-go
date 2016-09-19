// Copyright (c) 2016 Uber Technologies, Inc.
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

package net

import (
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/uber-go/atomic"
)

var (
	errServerStopped    = errors.New("the server has been stopped")
	errAlreadyListening = errors.New("the server is already listening")
)

// HTTPServer wraps an http.Server to listen asynchronously and allow stopping
// it.
type HTTPServer struct {
	*http.Server

	lock     sync.Mutex
	listener net.Listener
	done     chan error
	stopped  atomic.Bool
}

// NewHTTPServer wraps the given http.Server into an HTTPServer.
func NewHTTPServer(s *http.Server) *HTTPServer {
	return &HTTPServer{
		Server: s,
		done:   make(chan error, 1),
	}
}

// Listener returns the listener for this server or nil if the server isn't
// yet listening.
func (h *HTTPServer) Listener() net.Listener {
	return h.listener
}

// ListenAndServe starts the given HTTP server up in the background and
// returns immediately. The server listens on the configured Addr or ":http"
// if unconfigured.
//
// An error is returned if the server failed to start up, if the server was
// already listening, or if the server was stopped with Stop().
func (h *HTTPServer) ListenAndServe() error {
	if h.stopped.Load() {
		return errServerStopped
	}

	addr := h.Server.Addr
	if addr == "" {
		addr = ":http"
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	if h.listener != nil {
		return errAlreadyListening
	}

	listener, err := net.Listen("tcp", h.Server.Addr)
	if err != nil {
		return err
	}

	go func(done chan<- error) {
		// Serve always returns a non-nil error. For us, it's an error only if
		// we didn't call Stop().
		err := h.Server.Serve(listener)
		if !h.stopped.Load() {
			done <- err
		} else {
			done <- nil
		}
	}(h.done)

	h.listener = listener
	return nil
}

// Stop stops the server. An error is returned if the server stopped
// unexpectedly.
//
// Once a server is stopped, it cannot be started again with ListenAndServe.
func (h *HTTPServer) Stop() error {
	if h.stopped.Swap(true) {
		return nil
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	if h.listener == nil {
		return nil
	}

	closeErr := h.listener.Close()
	h.listener = nil
	serveErr := <-h.done // wait until Serve() stops
	if closeErr != nil {
		return closeErr
	}
	return serveErr
}
