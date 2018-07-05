// Copyright (c) 2018 Uber Technologies, Inc.
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
	"strings"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
	intnet "go.uber.org/yarpc/internal/net"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
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

// Interceptor specifies a function which can wrap the YARPC handler. If
// provided, this function will be called with an http.Handler which will
// route requests through YARPC. The http.Handler returned by this function
// may delegate requests to the provided YARPC handler to route them through
// YARPC.
func Interceptor(interceptor func(yarpcHandler http.Handler) http.Handler) InboundOption {
	return func(i *Inbound) {
		i.interceptor = interceptor
	}
}

// GrabHeaders specifies additional headers that are not prefixed with
// ApplicationHeaderPrefix that should be propagated to the caller.
//
// All headers given must begin with x- or X- or the Inbound that the
// returned option is passed to will return an error when Start is called.
//
// Headers specified with GrabHeaders are case-insensitive.
// https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2
func GrabHeaders(headers ...string) InboundOption {
	return func(i *Inbound) {
		for _, header := range headers {
			i.grabHeaders[strings.ToLower(header)] = struct{}{}
		}
	}
}

// NewInbound builds a new HTTP inbound that listens on the given address and
// sharing this transport.
func (t *Transport) NewInbound(addr string, opts ...InboundOption) *Inbound {
	i := &Inbound{
		once:              lifecycle.NewOnce(),
		addr:              addr,
		tracer:            t.tracer,
		logger:            t.logger,
		transport:         t,
		grabHeaders:       make(map[string]struct{}),
		bothResponseError: true,
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// Inbound receives YARPC requests using an HTTP server. It may be constructed
// using the NewInbound method on the Transport.
type Inbound struct {
	addr        string
	mux         *http.ServeMux
	muxPattern  string
	server      *intnet.HTTPServer
	router      transport.Router
	tracer      opentracing.Tracer
	logger      *zap.Logger
	transport   *Transport
	grabHeaders map[string]struct{}
	interceptor func(http.Handler) http.Handler

	once *lifecycle.Once

	// should only be false in testing
	bothResponseError bool
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
		return yarpcerrors.Newf(yarpcerrors.CodeInternal, "no router configured for transport inbound")
	}
	for header := range i.grabHeaders {
		if !strings.HasPrefix(header, "x-") {
			return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "header %s does not begin with 'x-'", header)
		}
	}

	var httpHandler http.Handler = handler{
		router:            i.router,
		tracer:            i.tracer,
		grabHeaders:       i.grabHeaders,
		bothResponseError: i.bothResponseError,
		logger:            i.logger,
	}
	if i.interceptor != nil {
		httpHandler = i.interceptor(httpHandler)
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
	i.logger.Info("started HTTP inbound", zap.String("address", i.addr))
	if len(i.router.Procedures()) == 0 {
		i.logger.Warn("no procedures specified for HTTP inbound")
	}
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
	var addrString string
	if addr := i.Addr(); addr != nil {
		addrString = addr.String()
	}
	return introspection.InboundStatus{
		Transport: "http",
		Endpoint:  addrString,
		State:     state,
	}
}
