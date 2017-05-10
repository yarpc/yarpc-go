// Copyright (c) 2017 Uber Technologies, Inc.
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

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/introspection"
	intnet "go.uber.org/yarpc/internal/net"
	"go.uber.org/yarpc/internal/sync"

	"github.com/opentracing/opentracing-go"
)

// InboundOption customizes the behavior of an HTTP Inbound constructed with
// NewInbound.
type InboundOption func(*Inbound)

func (InboundOption) httpOption() {}

// Mux specifies that the HTTP server should make the YARPC endpoint available
// under the given pattern on the given ServeMux. By default, the YARPC
// service is made available on all paths of the HTTP server. By specifying a
// ServeMux, users can narrow the endpoints under which the YARPC service is
// available and offer their own non-YARPC endpoints.
func Mux(pattern string, mux *http.ServeMux) InboundOption {
	return func(i *Inbound) {
		i.mux = mux
		i.muxPattern = pattern
	}
}

// NewInbound builds a new HTTP inbound that listens on the given address and
// sharing this transport.
//
// Note that this will change to take a net.Listener in 2.0.
func (t *Transport) NewInbound(addr string, opts ...InboundOption) *Inbound {
	i := &Inbound{
		once:      sync.Once(),
		addr:      addr,
		tracer:    t.tracer,
		transport: t,
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// NewInboundForListener builds a new HTTP inbound that listens on the given listener.
//
// Note that this will be NewInbound in 2.0.
func (t *Transport) NewInboundForListener(listener net.Listener, opts ...InboundOption) *Inbound {
	i := &Inbound{
		once:     sync.Once(),
		listener: listener,
		tracer:   t.tracer,
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// Inbound receives YARPC requests using an HTTP server. It may be constructed
// using the NewInbound method on the Transport.
type Inbound struct {
	addr       string
	listener   net.Listener
	mux        *http.ServeMux
	muxPattern string
	server     *intnet.HTTPServer
	router     transport.Router
	tracer     opentracing.Tracer
	transport  *Transport

	once sync.LifecycleOnce
}

// Tracer configures a tracer on this inbound.
func (i *Inbound) Tracer(tracer opentracing.Tracer) *Inbound {
	i.tracer = tracer
	return i
}

// SetRouter configures a router to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *Inbound) SetRouter(router transport.Router) {
	i.router = router
}

// Transports returns the inbound's HTTP transport.
func (i *Inbound) Transports() []transport.Transport {
	return []transport.Transport{i.transport}
}

// Start starts the inbound with a given service detail, opening a listening
// socket.
func (i *Inbound) Start() error {
	return i.once.Start(i.start)
}

func (i *Inbound) start() error {
	if i.router == nil {
		return errors.ErrNoRouter
	}

	var httpHandler http.Handler = handler{
		router: i.router,
		tracer: i.tracer,
	}
	if i.mux != nil {
		i.mux.Handle(i.muxPattern, httpHandler)
		httpHandler = i.mux
	}

	i.server = intnet.NewHTTPServer(&http.Server{
		Handler: httpHandler,
	})
	if i.listener != nil {
		if err := i.server.Serve(i.listener); err != nil {
			return err
		}
	} else {
		i.server.Addr = i.addr
		if err := i.server.ListenAndServe(); err != nil {
			return err
		}
	}

	i.addr = i.server.Listener().Addr().String() // in case it changed
	return nil
}

// Stop the inbound, closing the listening socket.
func (i *Inbound) Stop() error {
	return i.once.Stop(i.stop)
}

func (i *Inbound) stop() error {
	if i.server == nil {
		return nil
	}
	return i.server.Stop()
}

// IsRunning returns whether the inbound is currently running
func (i *Inbound) IsRunning() bool {
	return i.once.IsRunning()
}

// Addr returns the address on which the server is listening. Returns nil if
// Start has not been called yet.
func (i *Inbound) Addr() net.Addr {
	if i.server == nil {
		return nil
	}

	listener := i.server.Listener()
	if listener == nil {
		return nil
	}

	return listener.Addr()
}

// Introspect returns the state of the inbound for introspection purposes.
func (i *Inbound) Introspect() introspection.InboundStatus {
	state := "Stopped"
	if i.IsRunning() {
		state = "Started"
	}
	return introspection.InboundStatus{
		Transport: "http",
		Endpoint:  i.Addr().String(),
		State:     state,
	}
}
