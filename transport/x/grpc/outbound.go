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

package grpc

import (
	"bytes"
	"context"
	"io/ioutil"
	"sync"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	peerchooser "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UserAgent is the User-Agent that will be set for requests.
// http://www.grpc.io/docs/guides/wire.html#user-agents
const UserAgent = "yarpc-go/" + yarpc.Version

var _ transport.UnaryOutbound = (*Outbound)(nil)

// Outbound is a transport.UnaryOutbound.
type Outbound struct {
	once        *lifecycle.Once
	lock        sync.Mutex
	t           *Transport
	peerChooser peer.Chooser
	options     *outboundOptions
}

func newSingleOutbound(t *Transport, address string, options ...OutboundOption) *Outbound {
	return newOutbound(t, peerchooser.NewSingle(hostport.PeerIdentifier(address), t), options...)
}

func newOutbound(t *Transport, peerChooser peer.Chooser, options ...OutboundOption) *Outbound {
	return &Outbound{
		once:        lifecycle.NewOnce(),
		t:           t,
		peerChooser: peerChooser,
		options:     newOutboundOptions(options),
	}
}

// Start implements transport.Lifecycle#Start.
func (o *Outbound) Start() error {
	return o.once.Start(o.peerChooser.Start)
}

// Stop implements transport.Lifecycle#Stop.
func (o *Outbound) Stop() error {
	return o.once.Stop(o.peerChooser.Stop)
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (o *Outbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Transports implements transport.Inbound#Transports.
func (o *Outbound) Transports() []transport.Transport {
	return []transport.Transport{o.t}
}

// Chooser returns the associated peer.Chooser.
func (o *Outbound) Chooser() peer.Chooser {
	return o.peerChooser
}

// Call implements transport.UnaryOutbound#Call.
func (o *Outbound) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if err := o.once.WaitUntilRunning(ctx); err != nil {
		return nil, err
	}
	var responseBody []byte
	responseMD := metadata.New(nil)
	invokeErr := o.invoke(ctx, request, &responseBody, &responseMD)
	responseHeaders, err := getApplicationHeaders(responseMD)
	if err != nil {
		return nil, err
	}
	return &transport.Response{
		Body:    ioutil.NopCloser(bytes.NewBuffer(responseBody)),
		Headers: responseHeaders,
	}, invokeErrorToYARPCError(invokeErr)
}

func (o *Outbound) invoke(
	ctx context.Context,
	request *transport.Request,
	responseBody *[]byte,
	responseMD *metadata.MD,
) (retErr error) {
	md, err := transportRequestToMetadata(request)
	if err != nil {
		return err
	}
	requestBuffer := bufferpool.Get()
	defer bufferpool.Put(requestBuffer)
	if _, err := requestBuffer.ReadFrom(request.Body); err != nil {
		return err
	}
	fullMethod, err := procedureNameToFullMethod(request.Procedure)
	if err != nil {
		return err
	}
	var callOptions []grpc.CallOption
	if responseMD != nil {
		callOptions = []grpc.CallOption{grpc.Trailer(responseMD)}
	}
	apiPeer, onFinish, err := o.peerChooser.Choose(ctx, request)
	defer func() {
		if onFinish != nil {
			onFinish(retErr)
		}
	}()
	if err != nil {
		return err
	}
	grpcPeer, ok := apiPeer.(*grpcPeer)
	if !ok {
		return peer.ErrInvalidPeerConversion{
			Peer:         apiPeer,
			ExpectedType: "*grpcPeer",
		}
	}
	return grpc.Invoke(
		metadata.NewContext(ctx, md),
		fullMethod,
		requestBuffer.Bytes(),
		responseBody,
		grpcPeer.clientConn,
		callOptions...,
	)
}

func invokeErrorToYARPCError(err error) error {
	if err == nil {
		return nil
	}
	if yarpcerrors.IsYARPCError(err) {
		return err
	}
	status, ok := status.FromError(err)
	// if not a yarpc error or grpc error, just return the error
	if !ok {
		return err
	}
	code, ok := _grpcCodeToCode[status.Code()]
	if !ok {
		code = yarpcerrors.CodeUnknown
	}
	return yarpcerrors.FromHeaders(code, status.Message())
}
