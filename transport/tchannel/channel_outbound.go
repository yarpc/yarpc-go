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
	"context"

	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/x/introspection"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
)

var (
	_ transport.Namer                      = (*ChannelOutbound)(nil)
	_ transport.UnaryOutbound              = (*ChannelOutbound)(nil)
	_ introspection.IntrospectableOutbound = (*ChannelOutbound)(nil)
)

// NewOutbound builds a new TChannel outbound using the transport's shared
// channel to make requests to any connected peer.
func (t *ChannelTransport) NewOutbound() *ChannelOutbound {
	return &ChannelOutbound{
		once:      lifecycle.NewOnce(),
		channel:   t.ch,
		transport: t,
	}
}

// NewSingleOutbound builds a new TChannel outbound using the transport's shared
// channel to a specific peer.
func (t *ChannelTransport) NewSingleOutbound(addr string) *ChannelOutbound {
	return &ChannelOutbound{
		once:      lifecycle.NewOnce(),
		channel:   t.ch,
		transport: t,
		addr:      addr,
	}
}

// ChannelOutbound sends YARPC requests over TChannel. It may be constructed
// using the NewOutbound or NewSingleOutbound methods on the
// tchannel.ChannelTransport.
// If you have a YARPC peer.Chooser, use the unqualified tchannel.Transport
// instead (instead of the tchannel.ChannelTransport).
type ChannelOutbound struct {
	channel   Channel
	transport *ChannelTransport

	// If specified, this is the address to which requests will be made.
	// Otherwise, the global peer list of the Channel will be used.
	addr string

	once *lifecycle.Once
}

// TransportName is the transport name that will be set on `transport.Request`
// struct.
func (o *ChannelOutbound) TransportName() string {
	return TransportName
}

// Transports returns the underlying TChannel Transport for this outbound.
func (o *ChannelOutbound) Transports() []transport.Transport {
	return []transport.Transport{o.transport}
}

// Start starts the TChannel outbound.
func (o *ChannelOutbound) Start() error {
	// TODO: Should we create the connection to HostPort (if specified) here or
	// wait for the first call?
	return o.once.Start(nil)
}

// Stop stops the TChannel outbound.
func (o *ChannelOutbound) Stop() error {
	return o.once.Stop(o.stop)
}

func (o *ChannelOutbound) stop() error {
	o.channel.Close()
	return nil
}

// IsRunning returns whether the ChannelOutbound is running.
func (o *ChannelOutbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Call sends an RPC over this TChannel outbound.
func (o *ChannelOutbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	if req == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("request for tchannel channel outbound was nil")
	}
	if err := o.once.WaitUntilRunning(ctx); err != nil {
		return nil, intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "error waiting for tchannel channel outbound to start for service: %s", req.Service)
	}
	if _, ok := ctx.(tchannel.ContextWithHeaders); ok {
		return nil, errDoNotUseContextWithHeaders
	}

	// NB(abg): Under the current API, the local service's name is required
	// twice: once when constructing the TChannel and then again when
	// constructing the RPC.
	var call *tchannel.OutboundCall
	var err error

	format := tchannel.Format(req.Encoding)
	callOptions := tchannel.CallOptions{
		Format:          format,
		ShardKey:        req.ShardKey,
		RoutingKey:      req.RoutingKey,
		RoutingDelegate: req.RoutingDelegate,
	}
	if o.addr != "" {
		// If the hostport is given, we use the BeginCall on the channel
		// instead of the subchannel.
		call, err = o.channel.BeginCall(
			// TODO(abg): Set TimeoutPerAttempt in the context's retry options if
			// TTL is set.
			// (kris): Consider instead moving TimeoutPerAttempt to an outer
			// layer, just clamp the context on outbound call.
			ctx,
			o.addr,
			req.Service,
			req.Procedure,
			&callOptions,
		)
	} else {
		call, err = o.channel.GetSubChannel(req.Service).BeginCall(
			// TODO(abg): Set TimeoutPerAttempt in the context's retry options if
			// TTL is set.
			ctx,
			req.Procedure,
			&callOptions,
		)
	}

	if err != nil {
		return nil, toYARPCError(req, err)
	}

	reqHeaders := req.Headers.Items()
	if o.transport.originalHeaders {
		reqHeaders = req.Headers.OriginalItems()
	}
	// baggage headers are transport implementation details that are stripped out (and stored in the context). Users don't interact with it
	tracingBaggage := tchannel.InjectOutboundSpan(call.Response(), nil)
	if err := writeHeaders(format, reqHeaders, tracingBaggage, call.Arg2Writer); err != nil {
		// TODO(abg): This will wrap IO errors while writing headers as encode
		// errors. We should fix that.
		return nil, errors.RequestHeadersEncodeError(req, err)
	}

	if err := writeBody(req.Body, call); err != nil {
		return nil, toYARPCError(req, err)
	}

	res := call.Response()
	headers, err := readHeaders(format, res.Arg2Reader)
	if err != nil {
		if err, ok := err.(tchannel.SystemError); ok {
			return nil, fromSystemError(err)
		}
		// TODO(abg): This will wrap IO errors while reading headers as decode
		// errors. We should fix that.
		return nil, errors.ResponseHeadersDecodeError(req, err)
	}

	resBody, err := res.Arg3Reader()
	if err != nil {
		if err, ok := err.(tchannel.SystemError); ok {
			return nil, fromSystemError(err)
		}
		return nil, toYARPCError(req, err)
	}

	respService, _ := headers.Get(ServiceHeaderKey) // validateServiceName handles empty strings
	if err := validateServiceName(req.Service, respService); err != nil {
		return nil, err
	}

	err = getResponseError(headers)
	deleteReservedHeaders(headers)

	resp := &transport.Response{
		Headers:          headers,
		Body:             resBody,
		ApplicationError: res.ApplicationError(),
	}
	return resp, err
}

// Introspect returns basic status about this outbound.
func (o *ChannelOutbound) Introspect() introspection.OutboundStatus {
	state := "Stopped"
	if o.IsRunning() {
		state = "Running"
	}
	return introspection.OutboundStatus{
		Transport: "tchannel",
		Endpoint:  o.addr,
		State:     state,
	}
}
