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

package tchannel

import (
	"context"
	"io"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/errors"

	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/internal/sync"
)

// NewOutbound builds a new TChannel outbound using the transport's shared
// channel to make requests to any connected peer.
func (t *ChannelTransport) NewOutbound() *ChannelOutbound {
	return &ChannelOutbound{
		channel:   t.ch,
		transport: t,
	}
}

// NewSingleOutbound builds a new TChannel outbound using the transport's shared
// channel to a specific peer.
func (t *ChannelTransport) NewSingleOutbound(addr string) *ChannelOutbound {
	return &ChannelOutbound{
		channel:   t.ch,
		transport: t,
		addr:      addr,
	}
}

// ChannelOutbound is an outbound transport using a shared TChannel.
type ChannelOutbound struct {
	channel   Channel
	transport *ChannelTransport

	// If specified, this is the address to which requests will be made.
	// Otherwise, the global peer list of the Channel will be used.
	addr string

	once sync.LifecycleOnce
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
	if !o.IsRunning() {
		// TODO replace with "panicInDebug"
		return nil, errOutboundNotStarted
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
		return nil, err
	}

	// Inject tracing system baggage
	reqHeaders := tchannel.InjectOutboundSpan(call.Response(), req.Headers.Items())

	if err := writeRequestHeaders(ctx, format, reqHeaders, call.Arg2Writer); err != nil {
		// TODO(abg): This will wrap IO errors while writing headers as encode
		// errors. We should fix that.
		return nil, encoding.RequestHeadersEncodeError(req, err)
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
		return nil, encoding.ResponseHeadersDecodeError(req, err)
	}

	resBody, err := res.Arg3Reader()
	if err != nil {
		if err, ok := err.(tchannel.SystemError); ok {
			return nil, fromSystemError(err)
		}
		return nil, err
	}

	return &transport.Response{
		Headers:          headers,
		Body:             resBody,
		ApplicationError: res.ApplicationError(),
	}, nil
}

func writeBody(body io.Reader, call *tchannel.OutboundCall) error {
	w, err := call.Arg3Writer()
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, body); err != nil {
		return err
	}

	return w.Close()
}

func fromSystemError(err tchannel.SystemError) error {
	switch err.Code() {
	case tchannel.ErrCodeCancelled, tchannel.ErrCodeBusy, tchannel.ErrCodeBadRequest:
		return errors.RemoteBadRequestError(err.Message())
	case tchannel.ErrCodeTimeout:
		return errors.RemoteTimeoutError(err.Message())
	default:
		return errors.RemoteUnexpectedError(err.Message())
	}
}
