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

package grpchttp

import (
	"net"
	"net/http"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/soheilhy/cmux"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycle"
	yarpchttp "go.uber.org/yarpc/transport/http"
	yarpcgrpc "go.uber.org/yarpc/transport/x/grpc"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

var (
	errRouterNotSet = yarpcerrors.InternalErrorf("router not set")

	_ transport.Inbound = (*Inbound)(nil)
)

// InboundOption is an option for an inbound.
type InboundOption func(*inboundOptions)

// Tracer specifies the tracer to use.
//
// By default, opentracing.GlobalTracer() is used.
func Tracer(tracer opentracing.Tracer) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.httpTransportOptions = append(inboundOptions.httpTransportOptions, yarpchttp.Tracer(tracer))
		inboundOptions.grpcTransportOptions = append(inboundOptions.grpcTransportOptions, yarpcgrpc.Tracer(tracer))
	}
}

// HTTPMux specifies that the HTTP server should make the YARPC endpoint available
// under the given pattern on the given ServeMux. By default, the YARPC
// service is made available on all paths of the HTTP server. By specifying a
// ServeMux, users can narrow the endpoints under which the YARPC service is
// available and offer their own non-YARPC endpoints.
func HTTPMux(pattern string, serveMux *http.ServeMux) InboundOption {
	return func(inboundOptions *inboundOptions) {
		inboundOptions.httpInboundOptions = append(inboundOptions.httpInboundOptions, yarpchttp.Mux(pattern, serveMux))
	}
}

// NewInbound returns a new Inbound.
func NewInbound(listener net.Listener, options ...InboundOption) *Inbound {
	return newInbound(listener, options...)
}

// Inbound is a transport.Inbound.
type Inbound struct {
	once     *lifecycle.Once
	lock     sync.Mutex
	listener net.Listener
	options  *inboundOptions
	router   transport.Router

	grpcTransport *yarpcgrpc.Transport
	grpcInbound   *yarpcgrpc.Inbound
	httpTransport *yarpchttp.Transport
	httpInbound   *yarpchttp.Inbound
	sinkServer    *http.Server
}

func newInbound(listener net.Listener, options ...InboundOption) *Inbound {
	return &Inbound{
		once:     lifecycle.NewOnce(),
		listener: listener,
		options:  newInboundOptions(options),
	}
}

// Start implements transport.Lifecycle#Start.
func (i *Inbound) Start() error {
	return i.once.Start(i.start)
}

// Stop implements transport.Lifecycle#Stop.
func (i *Inbound) Stop() error {
	return i.once.Stop(i.stop)
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (i *Inbound) IsRunning() bool {
	return i.once.IsRunning()
}

// SetRouter implements transport.Inbound#SetRouter.
func (i *Inbound) SetRouter(router transport.Router) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.router = router
}

// Transports implements transport.Inbound#Transports.
func (i *Inbound) Transports() []transport.Transport {
	return []transport.Transport{}
}

func (i *Inbound) start() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.router == nil {
		return errRouterNotSet
	}

	mux := cmux.New(i.listener)
	grpcListener := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
	httpListener := mux.Match(cmux.HTTP1Fast("POST"))
	sinkListener := mux.Match(cmux.Any())
	mux.HandleError(func(err error) bool {
		if err != cmux.ErrListenerClosed {
			zap.L().Error(err.Error())
		}
		return false
	})

	grpcTransport := yarpcgrpc.NewTransport(i.options.grpcTransportOptions...)
	grpcInbound := grpcTransport.NewInbound(grpcListener, i.options.grpcInboundOptions...)
	grpcInbound.SetRouter(i.router)

	httpTransport := yarpchttp.NewTransport(i.options.httpTransportOptions...)
	httpInbound := httpTransport.NewInboundForListener(httpListener, i.options.httpInboundOptions...)
	httpInbound.SetRouter(i.router)

	sinkServer := &http.Server{Handler: http.HandlerFunc(func(responseWriter http.ResponseWriter, _ *http.Request) {
		responseWriter.WriteHeader(http.StatusInternalServerError)
	})}

	if err := grpcTransport.Start(); err != nil {
		_ = i.listener.Close()
		i.listener = nil
		return err
	}
	if err := grpcInbound.Start(); err != nil {
		_ = grpcTransport.Stop()
		_ = i.listener.Close()
		i.listener = nil
		return err
	}
	if err := httpTransport.Start(); err != nil {
		_ = grpcInbound.Stop()
		_ = grpcTransport.Stop()
		_ = i.listener.Close()
		i.listener = nil
		return err
	}
	if err := httpInbound.Start(); err != nil {
		_ = httpTransport.Stop()
		_ = grpcInbound.Stop()
		_ = grpcTransport.Stop()
		_ = i.listener.Close()
		i.listener = nil
		return err
	}

	// TODO: there should be some way to block until serving
	go func() { _ = sinkServer.Serve(sinkListener) }()
	go func() { _ = mux.Serve() }()

	i.grpcTransport = grpcTransport
	i.grpcInbound = grpcInbound
	i.httpTransport = httpTransport
	i.httpInbound = httpInbound
	i.sinkServer = sinkServer
	return nil
}

func (i *Inbound) stop() error {
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.listener != nil {
		_ = i.listener.Close()
		i.listener = nil
	}
	if i.sinkServer != nil {
		_ = i.sinkServer.Close()
		i.sinkServer = nil
	}
	var err error
	// TODO: what is going on
	//if i.httpInbound != nil {
	//err = multierr.Append(err, i.httpInbound.Stop())
	//}
	if i.httpTransport != nil {
		err = multierr.Append(err, i.httpTransport.Stop())
	}
	if i.grpcInbound != nil {
		err = multierr.Append(err, i.grpcInbound.Stop())
	}
	if i.grpcTransport != nil {
		err = multierr.Append(err, i.grpcTransport.Stop())
	}
	return err
}

type inboundOptions struct {
	grpcTransportOptions []yarpcgrpc.TransportOption
	grpcInboundOptions   []yarpcgrpc.InboundOption
	httpTransportOptions []yarpchttp.TransportOption
	httpInboundOptions   []yarpchttp.InboundOption
}

func newInboundOptions(options []InboundOption) *inboundOptions {
	inboundOptions := &inboundOptions{}
	for _, option := range options {
		option(inboundOptions)
	}
	return inboundOptions
}
