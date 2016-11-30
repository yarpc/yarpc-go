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

package http

import (
	"net"
	"net/http"

	intnet "go.uber.org/yarpc/internal/net"
	"go.uber.org/yarpc/transport"

	"github.com/opentracing/opentracing-go"
)

// InboundOption is an option for an HTTP inbound.
type InboundOption func(*Inbound)

// WithTracer is a NewInbound option that adds a tracer
func WithTracer(tracer opentracing.Tracer) InboundOption {
	return func(i *Inbound) {
		i.tracer = tracer
	}
}

// Mux specifies the ServeMux that the HTTP server should use and the pattern
// under which the YARPC endpoint should be registered.
func Mux(pattern string, mux *http.ServeMux) InboundOption {
	return func(i *Inbound) {
		i.mux = mux
		i.muxPattern = pattern
	}
}

// NewInbound builds a new HTTP inbound that listens on the given address.
func NewInbound(addr string, opts ...InboundOption) *Inbound {
	i := &Inbound{addr: addr}
	i.tracer = opentracing.GlobalTracer()
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// Inbound represents an HTTP Inbound. It is the same as the transport Inbound
// except it exposes the address on which the system is listening for
// connections.
type Inbound struct {
	addr       string
	mux        *http.ServeMux
	muxPattern string
	server     *intnet.HTTPServer
	tracer     opentracing.Tracer
}

// Start starts the inbound with a given service detail and transport
// dependencies, opening a listening socket.
func (i *Inbound) Start(service transport.ServiceDetail, d transport.Deps) error {

	var httpHandler http.Handler = handler{
		registry: service.Registry,
		tracer:   i.tracer,
	}
	if i.mux != nil {
		i.mux.Handle(i.muxPattern, httpHandler)
		httpHandler = i.mux
	}

	i.server = intnet.NewHTTPServer(&http.Server{
		Addr:    i.addr,
		Handler: httpHandler,
	})
	if err := i.server.ListenAndServe(); err != nil {
		return err
	}

	i.addr = i.server.Listener().Addr().String() // in case it changed
	return nil
}

// Stop the inbound, closing the listening socket.
func (i *Inbound) Stop() error {
	if i.server == nil {
		return nil
	}
	return i.server.Stop()
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
