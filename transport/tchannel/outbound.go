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

package tchannel

import (
	"context"

	"github.com/satori/go.uuid"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/introspection"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
	peerchooser "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
)

var (
	errDoNotUseContextWithHeaders = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "tchannel.ContextWithHeaders is not compatible with YARPC, use yarpc.CallOption instead")

	_ transport.UnaryOutbound              = (*Outbound)(nil)
	_ introspection.IntrospectableOutbound = (*Outbound)(nil)
)

// Outbound sends YARPC requests over TChannel.
// It may be constructed using the NewOutbound or NewSingleOutbound methods on
// the TChannel Transport.
type Outbound struct {
	transport *Transport
	chooser   peer.Chooser
	once      *lifecycle.Once
}

// NewOutbound builds a new TChannel outbound that selects a peer for each
// request using the given peer chooser.
func (t *Transport) NewOutbound(chooser peer.Chooser) *Outbound {
	return &Outbound{
		once:      lifecycle.NewOnce(),
		transport: t,
		chooser:   chooser,
	}
}

// NewSingleOutbound builds a new TChannel outbound always using the peer with
// the given address.
func (t *Transport) NewSingleOutbound(addr string) *Outbound {
	chooser := peerchooser.NewSingle(hostport.PeerIdentifier(addr), t)
	return t.NewOutbound(chooser)
}

// Chooser returns the outbound's peer chooser.
func (o *Outbound) Chooser() peer.Chooser {
	return o.chooser
}

// Call sends an RPC over this TChannel outbound.
func (o *Outbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	if req == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("request for tchannel outbound was nil")
	}
	if err := o.once.WaitUntilRunning(ctx); err != nil {
		return nil, intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "error waiting for tchannel outbound to start for service: %s", req.Service)
	}
	if _, ok := ctx.(tchannel.ContextWithHeaders); ok {
		return nil, errDoNotUseContextWithHeaders
	}
	p, onFinish, err := o.getPeerForRequest(ctx, req)
	if err != nil {
		return nil, toYARPCError(req, err)
	}
	res, err := p.Call(ctx, req)
	onFinish(err)
	return res, toYARPCError(req, err)
}

// Call sends an RPC to this specific peer.
func (p *tchannelPeer) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	root := p.transport.ch.RootPeers()
	tp := root.GetOrAdd(p.HostPort())
	return callWithPeer(ctx, req, tp, p.transport.headerCase)
}

// callWithPeer sends a request with the chosen peer.
func callWithPeer(ctx context.Context, req *transport.Request, peer *tchannel.Peer, headerCase headerCase) (*transport.Response, error) {
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

	// If the hostport is given, we use the BeginCall on the channel
	// instead of the subchannel.
	call, err = peer.BeginCall(
		// TODO(abg): Set TimeoutPerAttempt in the context's retry options if
		// TTL is set.
		// (kris): Consider instead moving TimeoutPerAttempt to an outer
		// layer, just clamp the context on outbound call.
		ctx,
		req.Service,
		req.Procedure,
		&callOptions,
	)

	if err != nil {
		return nil, err
	}
	// append a random per-request uuid
	req.UUID = uuid.NewV4()
	req.Headers.With(RequestUUIDHeaderKey, req.UUID.String())
	reqHeaders := headerMap(req.Headers, headerCase)

	// baggage headers are transport implementation details that are stripped out (and stored in the context). Users don't interact with it
	tracingBaggage := tchannel.InjectOutboundSpan(call.Response(), nil)
	if err := writeHeaders(format, reqHeaders, tracingBaggage, call.Arg2Writer); err != nil {
		// TODO(abg): This will wrap IO errors while writing headers as encode
		// errors. We should fix that.
		return nil, errors.RequestHeadersEncodeError(req, err)
	}

	if err := writeBody(req.Body, call); err != nil {
		return nil, err
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
		return nil, err
	}

	if err := checkResponseHeadersAndDeleteHeaderKey(req, headers); err != nil {
		return nil, err
	}

	return &transport.Response{
		Headers:          headers,
		Body:             resBody,
		ApplicationError: res.ApplicationError(),
	}, getResponseErrorAndDeleteHeaderKeys(headers)
}

func (o *Outbound) getPeerForRequest(ctx context.Context, treq *transport.Request) (*tchannelPeer, func(error), error) {
	p, onFinish, err := o.chooser.Choose(ctx, treq)
	if err != nil {
		return nil, nil, err
	}

	tp, ok := p.(*tchannelPeer)
	if !ok {
		return nil, nil, peer.ErrInvalidPeerConversion{
			Peer:         p,
			ExpectedType: "*tchannelPeer",
		}
	}

	return tp, onFinish, nil
}

// Transports returns the underlying TChannel Transport for this outbound.
func (o *Outbound) Transports() []transport.Transport {
	return []transport.Transport{o.transport}
}

// Start starts the TChannel outbound.
func (o *Outbound) Start() error {
	return o.once.Start(o.chooser.Start)
}

// Stop stops the TChannel outbound.
func (o *Outbound) Stop() error {
	return o.once.Stop(o.chooser.Stop)
}

// IsRunning returns whether the ChannelOutbound is running.
func (o *Outbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Introspect returns basic status about this outbound.
func (o *Outbound) Introspect() introspection.OutboundStatus {
	state := "Stopped"
	if o.IsRunning() {
		state = "Running"
	}
	var chooser introspection.ChooserStatus
	if i, ok := o.chooser.(introspection.IntrospectableChooser); ok {
		chooser = i.Introspect()
	} else {
		chooser = introspection.ChooserStatus{
			Name: "Introspection not available",
		}
	}
	return introspection.OutboundStatus{
		Transport: "tchannel",
		State:     state,
		Chooser:   chooser,
	}
}
