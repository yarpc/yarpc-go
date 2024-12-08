// Copyright (c) 2024 Uber Technologies, Inc.
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

package tchannel

import (
	"errors"
	"go.uber.org/yarpc/internal/interceptor/outboundinterceptor"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/inboundmiddleware"
	"go.uber.org/yarpc/internal/interceptor"
	"go.uber.org/yarpc/internal/tracinginterceptor"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
)

var errChannelOrServiceNameIsRequired = errors.New(
	"cannot instantiate tchannel.ChannelTransport: " +
		"please provide a service name with the ServiceName option " +
		"or an existing Channel with the WithChannel option")

// NewChannelTransport is a YARPC transport that facilitates sending and
// receiving YARPC requests through TChannel. It uses a shared TChannel
// Channel for both, incoming and outgoing requests, ensuring reuse of
// connections and other resources.
//
// Either the local service name (with the ServiceName option) or a user-owned
// TChannel (with the WithChannel option) MUST be specified.
//
// ChannelTransport uses the underlying TChannel Channel for load balancing
// and peer managament.
// Use NewTransport and its NewOutbound to support YARPC peer.Choosers.
func NewChannelTransport(opts ...TransportOption) (*ChannelTransport, error) {
	var options transportOptions
	options.tracer = opentracing.GlobalTracer()
	for _, opt := range opts {
		opt(&options)
	}

	// Attempt to construct a channel on behalf of the caller if none given.
	// Defer the error until Start since NewChannelTransport does not have
	// an error return.
	var err error
	ch := options.ch

	if ch == nil {
		if options.name == "" {
			err = errChannelOrServiceNameIsRequired
		} else {
			tracer := options.tracer
			if options.tracingInterceptorEnabled {
				tracer = opentracing.NoopTracer{}
			}
			opts := tchannel.ChannelOptions{Tracer: tracer}
			ch, err = tchannel.NewChannel(options.name, &opts)
			options.ch = ch
		}
	}

	if err != nil {
		return nil, err
	}

	return options.newChannelTransport(), nil
}

func (options transportOptions) newChannelTransport() *ChannelTransport {
	var (
		unaryInbounds  []interceptor.UnaryInbound
		unaryOutbounds []interceptor.DirectUnaryOutbound
	)
	logger := options.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	tracer := options.tracer

	if options.tracingInterceptorEnabled {
		ti := tracinginterceptor.New(tracinginterceptor.Params{
			Tracer:    tracer,
			Transport: TransportName,
		})
		unaryInbounds = append(unaryInbounds, ti)
		unaryOutbounds = append(unaryOutbounds, ti)

		tracer = opentracing.NoopTracer{}
	}
	return &ChannelTransport{
		once:                     lifecycle.NewOnce(),
		ch:                       options.ch,
		addr:                     options.addr,
		tracer:                   tracer,
		logger:                   logger.Named("tchannel"),
		originalHeaders:          options.originalHeaders,
		newResponseWriter:        newHandlerWriter,
		unaryInboundInterceptor:  inboundmiddleware.UnaryChain(unaryInbounds...),
		unaryOutboundInterceptor: outboundinterceptor.UnaryChain(unaryOutbounds...),
	}
}

// ChannelTransport maintains TChannel peers and creates inbounds and outbounds for
// TChannel.
// If you have a YARPC peer.Chooser, use the unqualified tchannel.Transport
// instead.
type ChannelTransport struct {
	once                     *lifecycle.Once
	ch                       Channel
	addr                     string
	tracer                   opentracing.Tracer
	logger                   *zap.Logger
	router                   transport.Router
	originalHeaders          bool
	newResponseWriter        func(inboundCallResponse, tchannel.Format, headerCase) responseWriter
	unaryInboundInterceptor  interceptor.UnaryInbound
	unaryOutboundInterceptor interceptor.DirectUnaryOutbound
}

// Channel returns the underlying TChannel "Channel" instance.
func (t *ChannelTransport) Channel() Channel {
	return t.ch
}

// ListenAddr exposes the listen address of the transport.
func (t *ChannelTransport) ListenAddr() string {
	return t.addr
}

// Start starts the TChannel transport. This starts making connections and
// accepting inbound requests. All inbounds must have been assigned a router
// to accept inbound requests before this is called.
func (t *ChannelTransport) Start() error {
	return t.once.Start(t.start)
}

func (t *ChannelTransport) start() error {

	if t.router != nil {
		// Set up handlers. This must occur after construction because the
		// dispatcher, or its equivalent, calls SetRouter before Start.
		// This also means that SetRouter should be called on every inbound
		// before calling Start on any transport or inbound.
		services := make(map[string]struct{})
		for _, p := range t.router.Procedures() {
			services[p.Service] = struct{}{}
		}

		for s := range services {
			sc := t.ch.GetSubChannel(s)
			existing := sc.GetHandlers()
			sc.SetHandler(handler{
				existing:                 existing,
				router:                   t.router,
				tracer:                   t.tracer,
				logger:                   t.logger,
				newResponseWriter:        t.newResponseWriter,
				unaryOutboundInterceptor: t.unaryOutboundInterceptor,
				unaryInboundInterceptor:  t.unaryInboundInterceptor},
			)
		}
	}

	if t.ch.State() == tchannel.ChannelListening {
		// Channel.Start() was called before RPC.Start(). We still want to
		// update the Handler and what t.addr means, but nothing else.
		t.addr = t.ch.PeerInfo().HostPort
		return nil
	}

	// Default to ListenIP if addr wasn't given.
	addr := t.addr
	if addr == "" {
		listenIP, err := tchannel.ListenIP()
		if err != nil {
			return err
		}

		addr = listenIP.String() + ":0"
		// TODO(abg): Find a way to export this to users
	}

	// TODO(abg): If addr was just the port (":4040"), we want to use
	// ListenIP() + ":4040" rather than just ":4040".

	if err := t.ch.ListenAndServe(addr); err != nil {
		return err
	}

	t.addr = t.ch.PeerInfo().HostPort
	return nil
}

// Stop stops the TChannel transport. It starts rejecting incoming requests
// and draining connections before closing them.
// In a future version of YARPC, Stop will block until the underlying channel
// has closed completely.
func (t *ChannelTransport) Stop() error {
	return t.once.Stop(t.stop)
}

func (t *ChannelTransport) stop() error {
	t.ch.Close()
	return nil
}

// IsRunning returns whether the ChannelTransport is running.
func (t *ChannelTransport) IsRunning() bool {
	return t.once.IsRunning()
}
