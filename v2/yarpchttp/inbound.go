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

package yarpchttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/opentracing/opentracing-go"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internalhttp"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/zap"
)

// Inbound receives YARPC requests using an HTTP server.
type Inbound struct {
	// Listener is an open listener for inbound HTTP requests.
	//
	// The listener will be closed on Stop(ctx).
	Listener net.Listener

	// Addr is a host:port on which to listen if no Listener is expressly provided.
	Addr string

	// Router is the router to handle requests.
	Router yarpc.Router

	// Mux specifies that the HTTP server should make the YARPC endpoint available
	// under the MuxPattern on the given ServeMux.
	// By default, the YARPC service is made available on all paths of the HTTP
	// server.
	// By specifying a ServeMux, users can narrow the endpoints under which the
	// YARPC service is available and offer their own non-YARPC endpoints.
	Mux *http.ServeMux

	// MuxPattern is a path prefix that the YARPC inbound will require for all
	// inbound RPC.
	MuxPattern string

	// GrabHeaders specifies additional headers that are not prefixed with
	// ApplicationHeaderPrefix that should be propagated to the caller.
	//
	// All headers given must begin with x- or X- or the Inbound that the
	// returned option is passed to will return an error when Start is called.
	//
	// Headers specified with GrabHeaders are case-insensitive.
	// https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2
	GrabHeaders []string

	// Interceptor specifies a function which can wrap the YARPC handler. If
	// provided, this function will be called with an http.Handler which will
	// route requests through YARPC. The http.Handler returned by this function
	// may delegate requests to the provided YARPC handler to route them through
	// YARPC.
	Interceptor func(http.Handler) http.Handler

	// Tracer configures a tracer for the inbound.
	Tracer opentracing.Tracer

	// Logger configures a tracer for the inbound.
	Logger *zap.Logger

	// legacyResponseError disables the Rpc-Error-Message header and
	// writes the error message to the body instead, even if the handler
	// receives the Rpc-Accepts-Both-Response-Error header with the value
	// "true".
	legacyResponseError bool

	server *internalhttp.HTTPServer
}

// Start starts the inbound with a given service detail, opening a listening
// socket.
func (i *Inbound) Start(_ context.Context) error {
	if i.Router == nil {
		return yarpcerror.Newf(yarpcerror.CodeInternal, "no router configured for HTTP inbound")
	}

	grabHeaders := make(map[string]struct{}, len(i.GrabHeaders))
	for _, header := range i.GrabHeaders {
		if !strings.HasPrefix(header, "x-") {
			return yarpcerror.Newf(yarpcerror.CodeInvalidArgument, "header %s does not begin with 'x-'", header)
		}
		grabHeaders[header] = struct{}{}
	}

	var tracer opentracing.Tracer
	if i.Tracer == nil {
		tracer = opentracing.GlobalTracer()
	} else {
		tracer = i.Tracer
	}

	var logger *zap.Logger
	if i.Logger == nil {
		logger = zap.NewNop()
	} else {
		logger = i.Logger
	}

	handler := handler{
		router:              i.Router,
		grabHeaders:         grabHeaders,
		legacyResponseError: i.legacyResponseError,
		logger:              logger,
		tracer:              tracer,
	}

	var httpHandler http.Handler = handler

	if i.Interceptor != nil {
		httpHandler = i.Interceptor(httpHandler)
	}

	if i.Mux != nil {
		muxPattern := "/"
		if i.MuxPattern != "" {
			muxPattern = i.MuxPattern
		}
		i.Mux.Handle(muxPattern, httpHandler)
		httpHandler = i.Mux
	}

	server := &http.Server{
		Handler: httpHandler,
	}

	if i.Listener == nil {
		var err error
		i.Listener, err = net.Listen("tcp", i.Addr)
		if err != nil {
			return err
		}
	}

	handler.addr = i.Listener.Addr().String()

	i.server = internalhttp.NewHTTPServer(server)
	go i.server.Run(i.Listener)

	logger.Info("started HTTP inbound", zap.Stringer("address", i.Listener.Addr()))
	if len(i.Router.Procedures()) == 0 {
		logger.Warn("no procedures specified for HTTP inbound")
	}
	return nil
}

// Stop the inbound using Shutdown.
func (i *Inbound) Stop(ctx context.Context) error {
	if i.server == nil {
		return fmt.Errorf("HTTP inbounds must be started before they are stopped")
	}
	return i.server.Shutdown(ctx)
}
