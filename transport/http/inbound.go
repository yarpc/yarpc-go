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
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/api/x/introspection"
	intnet "go.uber.org/yarpc/internal/net"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/transport/internal/tls/muxlistener"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	// We want a value that's around 5 seconds, but slightly higher than how
	// long a successful HTTP shutdown can take.
	// There's a specific path in the HTTP shutdown path that can take 5 seconds:
	// https://golang.org/src/net/http/server.go?s=83923:83977#L2710
	// This avoids timeouts in shutdown caused by new idle connections, without
	// making the timeout too large.
	defaultShutdownTimeout = 6 * time.Second
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
// Interceptors are applied in LIFO order, leading to an earlier interceptor's
// handler being executed before latter interceptor handlers
func Interceptor(interceptor func(yarpcHandler http.Handler) http.Handler) InboundOption {
	return func(i *Inbound) {
		i.interceptors = append(i.interceptors, interceptor)
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

// ShutdownTimeout specifies the maximum duration the inbound should wait for
// closing idle connections, and pending calls to complete.
//
// Set to 0 to wait for a complete drain.
//
// Defaults to 5 seconds.
func ShutdownTimeout(timeout time.Duration) InboundOption {
	return func(i *Inbound) {
		i.shutdownTimeout = timeout
	}
}

// InboundTLSConfiguration returns an InboundOption that provides the TLS
// confiugration used for setting up TLS inbound.
func InboundTLSConfiguration(tlsConfig *tls.Config) InboundOption {
	return func(i *Inbound) {
		i.tlsConfig = tlsConfig
	}
}

// InboundTLSMode returns an InboundOption that sets inbound TLS mode.
// It must be noted that TLS configuration must be passed separately using inbound
// option InboundTLSConfiguration.
func InboundTLSMode(mode yarpctls.Mode) InboundOption {
	return func(i *Inbound) {
		i.tlsMode = mode
	}
}

// DisableHTTP2 returns an InboundOption that disables HTTP/2 support.
func DisableHTTP2(flag bool) InboundOption {
	return func(i *Inbound) {
		i.disableHTTP2 = flag
	}
}

// ReadHeaderTimeout returns an InboundOption that sets the http.Server ReadHeaderTimeout
func ReadHeaderTimeout(timeout time.Duration) InboundOption {
	return func(i *Inbound) {
		i.server.ReadHeaderTimeout = timeout
	}
}

// ReadTimeout returns an InboundOption that sets the http.Server ReadTimeout
func ReadTimeout(timeout time.Duration) InboundOption {
	return func(i *Inbound) {
		i.server.ReadTimeout = timeout
	}
}

// WriteTimeout returns an InboundOption that sets the http.Server WriteTimeout
func WriteTimeout(timeout time.Duration) InboundOption {
	return func(i *Inbound) {
		i.server.WriteTimeout = timeout
	}
}

// IdleTimeout returns an InboundOption that sets the http.Server IdleTimeout
func IdleTimeout(timeout time.Duration) InboundOption {
	return func(i *Inbound) {
		i.server.IdleTimeout = timeout
	}
}

// NewInbound builds a new HTTP inbound that listens on the given address and
// sharing this transport.
func (t *Transport) NewInbound(addr string, opts ...InboundOption) *Inbound {
	i := &Inbound{
		once:              lifecycle.NewOnce(),
		addr:              addr,
		shutdownTimeout:   defaultShutdownTimeout,
		tracer:            t.tracer,
		logger:            t.logger,
		transport:         t,
		grabHeaders:       make(map[string]struct{}),
		bothResponseError: true,
		disableHTTP2:      false,
	}
	server := &http.Server{
		Addr: i.addr,
	}
	i.server = intnet.NewHTTPServer(server)

	for _, opt := range opts {
		opt(i)
	}
	return i
}

// Inbound receives YARPC requests using an HTTP server. It may be constructed
// using the NewInbound method on the Transport.
type Inbound struct {
	addr            string
	mux             *http.ServeMux
	muxPattern      string
	server          *intnet.HTTPServer
	shutdownTimeout time.Duration
	router          transport.Router
	tracer          opentracing.Tracer
	logger          *zap.Logger
	transport       *Transport
	grabHeaders     map[string]struct{}
	interceptors    []func(http.Handler) http.Handler

	once *lifecycle.Once

	// should only be false in testing
	bothResponseError bool

	tlsConfig *tls.Config
	tlsMode   yarpctls.Mode

	disableHTTP2 bool
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
		transport:         i.transport,
		grabHeaders:       i.grabHeaders,
		bothResponseError: i.bothResponseError,
		logger:            i.logger,
	}

	// reverse iterating because we want the last from options to wrap the
	// the underlying yarpc http handlers.
	// This way, the first from the option will be the
	// outermost wrapper http.Handler and it will be invoked first during request handling.
	for j := len(i.interceptors) - 1; j >= 0; j-- {
		httpHandler = i.interceptors[j](httpHandler)
	}
	if i.mux != nil {
		i.mux.Handle(i.muxPattern, httpHandler)
		httpHandler = i.mux
	}

	i.server.Handler = httpHandler
	if !i.disableHTTP2 {
		h2s := &http2.Server{}
		i.server.Handler = h2c.NewHandler(i.server.Handler, h2s)
		err := http2.ConfigureServer(i.server.Server, h2s)
		if err != nil {
			return fmt.Errorf("failed to configure HTTP/2 server: %w", err)
		}
	}

	addr := i.addr
	if addr == "" {
		addr = ":http"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if i.tlsMode != yarpctls.Disabled {
		if i.tlsConfig == nil {
			return errors.New("HTTP TLS enabled but configuration not provided")
		}

		listener = muxlistener.NewListener(muxlistener.Config{
			Listener:      listener,
			TLSConfig:     i.tlsConfig,
			ServiceName:   i.transport.serviceName,
			TransportName: TransportName,
			Meter:         i.transport.meter,
			Logger:        i.logger,
			Mode:          i.tlsMode,
		})
	}

	if err := i.server.Serve(listener); err != nil {
		return err
	}

	i.addr = i.server.Listener().Addr().String() // in case it changed
	i.logger.Info("started HTTP inbound", zap.String("address", i.addr))
	if len(i.router.Procedures()) == 0 {
		i.logger.Warn("no procedures specified for HTTP inbound")
	}
	return nil
}

// Stop the inbound using Shutdown.
func (i *Inbound) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), i.shutdownTimeout)
	defer cancel()

	return i.shutdown(ctx)
}

// shutdown the inbound, closing the listening socket, closing idle
// connections, and waiting for all pending calls to complete.
func (i *Inbound) shutdown(ctx context.Context) error {
	return i.once.Stop(func() error {
		if i.server == nil {
			return nil
		}

		return i.server.Shutdown(ctx)
	})
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
