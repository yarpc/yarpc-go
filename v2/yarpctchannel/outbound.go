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

package yarpctchannel

import (
	"context"
	"errors"
	"io"

	"github.com/uber/tchannel-go"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaliopool"
	"go.uber.org/yarpc/v2/internal/internalyarpcerror"
	"go.uber.org/yarpc/v2/yarpcencoding"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcpeer"
)

var (
	errDoNotUseContextWithHeaders = yarpcerror.Newf(yarpcerror.CodeInvalidArgument, "tchannel.ContextWithHeaders is not compatible with YARPC, use yarpc.CallOption instead")

	_ yarpc.UnaryTransportOutbound = (*Outbound)(nil)
)

// Outbound sends YARPC requests over TChannel.
type Outbound struct {
	// Chooser is a peer chooser for outbound requests.
	//
	// The chooser receives the request metadata and returns peers.
	// The chooser is backed by a dialer and a peer list.
	// You can instead use an Addr and Dialer directly, bypassing YARPC peer
	// selection.
	Chooser yarpc.Chooser

	// Addr is the host:port of the peer that will handle outbound requests.
	//
	// Providing an Addr and Dialer is an alternative to using YARPC peer
	// selection.
	// The outbound will dial this exact address for each outbound request.
	Addr string

	// Dialer is a dialer to retain connections to the remote peer.
	//
	// Providing an Addr and Dialer is an alternative to using YARPC peer
	// selection.
	// The outbound will dial this exact address for each outbound request.
	Dialer yarpc.Dialer

	// HeaderCase specifies whether to forward headers with or without
	// canonicalizing them.
	//
	// The YARPC default is to canonicalize all outbound headers, since
	// this is the common denominator between transport protocols including
	// HTTP, and gRPC.
	// Some TChannel services depend on the exact case of the header.
	HeaderCase HeaderCase

	ch *tchannel.Channel
}

// Call makes a TChannel request.
//
// If the outbound has a Chooser, the outbound will use the chooser to obtain a
// peer for the duration of the request.
// Assume that the Chooser ignores the req.Peer identifier unless the Chooser
// specifies otherwise a custom behavior.
// The Chooser implementation is free to interpret the req.Peer as a hint, a
// requirement, or ignore it altogether.
//
// Otherwise, if the request has a specified Peer, the outbound will use the
// Dialer to retain that peer for the duration of the request.
//
// Otherwise, the outbound will use the Dialer to retain the peer identified by
// the outbound's Addr for the duration of the request.
func (o *Outbound) Call(ctx context.Context, req *yarpc.Request, reqBody *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	if req == nil {
		return nil, nil, yarpcerror.InvalidArgumentErrorf("request for tchannel outbound was nil")
	}
	if _, ok := ctx.(tchannel.ContextWithHeaders); ok {
		return nil, nil, errDoNotUseContextWithHeaders
	}
	peer, onFinish, err := o.getPeerForRequest(ctx, req)
	if err != nil {
		return nil, nil, toYARPCError(req, err)
	}

	root := peer.dialer.ch.RootPeers()
	tchannelPeer := root.GetOrAdd(peer.Identifier())
	res, resBody, err := callWithPeer(ctx, req, reqBody, tchannelPeer, o.HeaderCase)
	onFinish(err)

	return res, resBody, toYARPCError(req, err)
}

// callWithPeer sends a request with the chosen peer.
func callWithPeer(ctx context.Context, req *yarpc.Request, reqBody *yarpc.Buffer, peer *tchannel.Peer, headerCase HeaderCase) (*yarpc.Response, *yarpc.Buffer, error) {
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
	call, err = peer.BeginCall(ctx, req.Service, req.Procedure, &callOptions)

	if err != nil {
		return nil, nil, err
	}
	reqHeaders := headerMap(req.Headers, headerCase)

	// baggage headers are transport implementation details that are stripped
	// out (and stored in the context). Users don't interact with it.
	tracingBaggage := tchannel.InjectOutboundSpan(call.Response(), nil)
	if err := writeHeaders(format, reqHeaders, tracingBaggage, call.Arg2Writer); err != nil {
		// TODO(abg): This will wrap IO errors while writing headers as encode
		// errors. We should fix that.
		return nil, nil, yarpcencoding.RequestHeadersEncodeError(req, err)
	}

	if err := writeBody(reqBody, call); err != nil {
		return nil, nil, err
	}

	res := call.Response()
	headers, err := readHeaders(format, res.Arg2Reader)
	if err != nil {
		if err, ok := err.(tchannel.SystemError); ok {
			return nil, nil, fromSystemError(err)
		}
		// TODO(abg): This will wrap IO errors while reading headers as decode
		// errors. We should fix that.
		return nil, nil, yarpcencoding.ResponseHeadersDecodeError(req, err)
	}

	arg3Reader, err := res.Arg3Reader()
	if err != nil {
		if err, ok := err.(tchannel.SystemError); ok {
			return nil, nil, fromSystemError(err)
		}
		return nil, nil, err
	}
	resBody := &yarpc.Buffer{}
	if _, err = internaliopool.Copy(resBody, arg3Reader); err != nil {
		return nil, nil, err
	}

	// service name match validation, return yarpcerror.CodeInternal error if not match
	if match, resSvcName := checkServiceMatchAndDeleteHeaderKey(req.Service, headers); !match {
		return nil, nil, yarpcerror.InternalErrorf("service name sent from the request "+
			"does not match the service name received in the response: sent %q, got: %q", req.Service, resSvcName)
	}

	var appErr error
	if res.ApplicationError() {
		// TODO(mhp): this is a filler error for now to preserve current yarpc
		// error behavior. This should later be more well-defined.
		appErr = errors.New("application error")
	}
	return &yarpc.Response{
		Peer:             getPeerAndDeleteHeaderKeys(headers),
		Headers:          headers,
		ApplicationError: appErr,
	}, resBody, getResponseErrorAndDeleteHeaderKeys(headers)
}

func (o *Outbound) getPeerForRequest(ctx context.Context, req *yarpc.Request) (*tchannelPeer, func(error), error) {
	var (
		peer     yarpc.Peer
		onFinish func(error)
		err      error
	)
	if o.Chooser != nil {
		peer, onFinish, err = o.Chooser.Choose(ctx, req)
	} else if req.Peer != nil {
		peer, onFinish, err = o.getEphemeralPeer(req.Peer)
	} else if o.Addr != "" {
		id := yarpc.Address(o.Addr)
		peer, onFinish, err = o.getEphemeralPeer(id)
	} else {
		return nil, nil, yarpcerror.FailedPreconditionErrorf("TChannel outbound must have a chooser or address, or request must address a specific peer")
	}

	if err != nil {
		return nil, nil, err
	}

	tchannelPeer, ok := peer.(*tchannelPeer)
	if !ok {
		return nil, nil, yarpcpeer.ErrInvalidPeerConversion{
			Peer:         peer,
			ExpectedType: "*tchannelPeer",
		}
	}

	return tchannelPeer, onFinish, nil
}

func (o *Outbound) getEphemeralPeer(id yarpc.Identifier) (yarpc.Peer, func(error), error) {
	if o.Dialer == nil {
		return nil, nil, yarpcpeer.ErrMissingDialer{Transport: "tchannel"}
	}
	peer, err := o.Dialer.RetainPeer(id, yarpc.NopSubscriber)
	if err != nil {
		return nil, nil, err
	}
	err = o.Dialer.ReleasePeer(id, yarpc.NopSubscriber)
	if err != nil {
		return nil, nil, err
	}
	return peer, nopFinish, nil
}

func nopFinish(error) {}

func writeBody(body io.Reader, call *tchannel.OutboundCall) error {
	w, err := call.Arg3Writer()
	if err != nil {
		return err
	}

	if _, err := internaliopool.Copy(w, body); err != nil {
		return err
	}

	return w.Close()
}

func fromSystemError(err tchannel.SystemError) error {
	code, ok := _tchannelCodeToCode[err.Code()]
	if !ok {
		// This should be unreachable.
		return yarpcerror.Newf(yarpcerror.CodeInternal, "got tchannel.SystemError %v which did not have a matching YARPC code", err)
	}
	return yarpcerror.Newf(code, err.Message())
}

// ServiceHeaderKey is internal key used by YARPC, we need to remove it before give response to client
// only does verification when there is a response service header.
func checkServiceMatchAndDeleteHeaderKey(reqSvcName string, resHeaders yarpc.Headers) (bool, string) {
	if resSvcName, ok := resHeaders.Get(ServiceHeaderKey); ok {
		resHeaders.Del(ServiceHeaderKey)
		return reqSvcName == resSvcName, resSvcName
	}
	return true, ""
}

func getPeerAndDeleteHeaderKeys(headers yarpc.Headers) yarpc.Identifier {
	peer, ok := popHeader(headers, PeerHeaderKey)
	if !ok || peer == "" {
		return nil
	}
	return yarpc.Address(peer)
}

func getResponseErrorAndDeleteHeaderKeys(headers yarpc.Headers) error {
	errorCodeString, ok := popHeader(headers, ErrorCodeHeaderKey)
	if !ok {
		return nil
	}
	var errorCode yarpcerror.Code
	if err := errorCode.UnmarshalText([]byte(errorCodeString)); err != nil {
		return err
	}
	if errorCode == yarpcerror.CodeOK {
		return yarpcerror.Newf(yarpcerror.CodeInternal, "got CodeOK from error header")
	}
	errorName, _ := popHeader(headers, ErrorNameHeaderKey)
	errorMessage, _ := popHeader(headers, ErrorMessageHeaderKey)
	return internalyarpcerror.NewWithNamef(errorCode, errorName, errorMessage)
}

func popHeader(h yarpc.Headers, n string) (string, bool) {
	v, ok := h.Get(n)
	if ok {
		h.Del(n)
	}
	return v, ok
}
