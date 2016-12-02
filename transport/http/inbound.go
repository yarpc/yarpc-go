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

	"go.uber.org/yarpc/internal/errors"
	intnet "go.uber.org/yarpc/internal/net"
	"go.uber.org/yarpc/transport"

	"github.com/opentracing/opentracing-go"
)

// NewInbound builds a new HTTP inbound that listens on the given address.
func NewInbound(addr string) *Inbound {
	return &Inbound{
		addr:   addr,
		tracer: opentracing.GlobalTracer(),
	}
}

// Inbound represents an HTTP Inbound. It is the same as the transport Inbound
// except it exposes the address on which the system is listening for
// connections.
type Inbound struct {
	addr       string
	mux        *http.ServeMux
	muxPattern string
	server     *intnet.HTTPServer
	registry   transport.Registry
	tracer     opentracing.Tracer
}

// WithMux specifies the ServeMux that the HTTP server should use and the
// pattern under which the YARPC endpoint should be registered.
func (i *Inbound) WithMux(pattern string, mux *http.ServeMux) *Inbound {
	i.mux = mux
	i.muxPattern = pattern
	return i
}

// WithTracer configures a tracer on this inbound.
func (i *Inbound) WithTracer(tracer opentracing.Tracer) *Inbound {
	i.tracer = tracer
	return i
}

// WithRegistry configures a registry to handle incoming requests,
// as a chained method for convenience.
func (i *Inbound) WithRegistry(registry transport.Registry) *Inbound {
	i.registry = registry
	return i
}

// SetRegistry configures a registry to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *Inbound) SetRegistry(registry transport.Registry) {
	i.registry = registry
}

// Transports returns the inbound's HTTP transport.
func (i *Inbound) Transports() []transport.Transport {
	// TODO factor out transport and return it here.
	return []transport.Transport{}
}

// Start starts the inbound with a given service detail, opening a listening
// socket.
func (i *Inbound) Start() error {

	if i.registry == nil {
		return errors.NoRegistryError{}
	}

	var httpHandler http.Handler = handler{
		registry: i.registry,
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
