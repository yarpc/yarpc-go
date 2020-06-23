// Copyright (c) 2020 Uber Technologies, Inc.
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
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/grpcerrorcodes"
	intyarpcerrors "go.uber.org/yarpc/internal/yarpcerrors"
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

// TransportName is the transport name that will be set on `transport.Request`
// struct.
func (o *Outbound) TransportName() string {
	return TransportName
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

// Chooser returns the peer.Chooser associated with this Outbound.
func (o *Outbound) Chooser() peer.Chooser {
	return o.peerChooser
}

// Call implements transport.UnaryOutbound#Call.
func (o *Outbound) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if request == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("request for grpc outbound was nil")
	}
	if err := o.once.WaitUntilRunning(ctx); err != nil {
		return nil, intyarpcerrors.AnnotateWithInfo(yarpcerrors.FromError(err), "error waiting for grpc outbound to start for service: %s", request.Service)
	}
	start := time.Now()

	var responseBody []byte
	var responseMD metadata.MD
	invokeErr := o.invoke(ctx, request, &responseBody, &responseMD, start)

	responseHeaders, err := getApplicationHeaders(responseMD)
	if err != nil {
		return nil, err
	}
	return &transport.Response{
		Body:             ioutil.NopCloser(bytes.NewBuffer(responseBody)),
		Headers:          responseHeaders,
		ApplicationError: metadataToIsApplicationError(responseMD),
	}, invokeErr
}

func (o *Outbound) invoke(
	ctx context.Context,
	request *transport.Request,
	responseBody *[]byte,
	responseMD *metadata.MD,
	start time.Time,
) (retErr error) {
	md, err := transportRequestToMetadata(request)
	if err != nil {
		return err
	}

	bytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
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
	if err != nil {
		return err
	}
	defer func() { onFinish(retErr) }()
	grpcPeer, ok := apiPeer.(*grpcPeer)
	if !ok {
		return peer.ErrInvalidPeerConversion{
			Peer:         apiPeer,
			ExpectedType: "*grpcPeer",
		}
	}

	tracer := o.t.options.tracer
	createOpenTracingSpan := &transport.CreateOpenTracingSpan{
		Tracer:        tracer,
		TransportName: TransportName,
		StartTime:     start,
		ExtraTags:     yarpc.OpentracingTags,
	}
	ctx, span := createOpenTracingSpan.Do(ctx, request)
	defer span.Finish()

	if err := tracer.Inject(span.Context(), opentracing.HTTPHeaders, mdReadWriter(md)); err != nil {
		return err
	}

	err = transport.UpdateSpanWithErr(
		span,
		grpcPeer.clientConn.Invoke(
			metadata.NewOutgoingContext(ctx, md),
			fullMethod,
			bytes,
			responseBody,
			callOptions...,
		),
	)
	if err != nil {
		return invokeErrorToYARPCError(err, *responseMD)
	}
	// Service name match validation, return yarpcerrors.CodeInternal error if not match
	if match, resSvcName := checkServiceMatch(request.Service, *responseMD); !match {
		// If service doesn't match => we got response => span must not be nil
		return transport.UpdateSpanWithErr(span, yarpcerrors.InternalErrorf("service name sent from the request "+
			"does not match the service name received in the response: sent %q, got: %q", request.Service, resSvcName))
	}
	return nil
}

func metadataToIsApplicationError(responseMD metadata.MD) bool {
	if responseMD == nil {
		return false
	}
	value, ok := responseMD[ApplicationErrorHeader]
	return ok && len(value) > 0 && len(value[0]) > 0
}

func invokeErrorToYARPCError(err error, responseMD metadata.MD) error {
	if err == nil {
		return nil
	}
	if yarpcerrors.IsStatus(err) {
		return err
	}
	status, ok := status.FromError(err)
	// if not a yarpc error or grpc error, just return a wrapped error
	if !ok {
		return yarpcerrors.FromError(err)
	}
	code, ok := grpcerrorcodes.GRPCCodeToYARPCCode[status.Code()]
	if !ok {
		code = yarpcerrors.CodeUnknown
	}
	var name string
	if responseMD != nil {
		value, ok := responseMD[ErrorNameHeader]
		// TODO: what to do if the length is > 1?
		if ok && len(value) == 1 {
			name = value[0]
		}
	}
	message := status.Message()
	// we put the name as a prefix for grpc compatibility
	// if there was no message, the message will be the name, so we leave it as the message
	if name != "" && message != "" && message != name {
		message = strings.TrimPrefix(message, name+": ")
	} else if name != "" && message == name {
		message = ""
	}

	yarpcErr := intyarpcerrors.NewWithNamef(code, name, message)
	if details, err := marshalError(status); err != nil {
		return err
	} else if details != nil {
		yarpcErr = yarpcErr.WithDetails(details)
	}
	return yarpcErr
}

// CallStream implements transport.StreamOutbound#CallStream.
func (o *Outbound) CallStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	if err := o.once.WaitUntilRunning(ctx); err != nil {
		return nil, err
	}
	return o.stream(ctx, request, time.Now())
}

func (o *Outbound) stream(
	ctx context.Context,
	req *transport.StreamRequest,
	start time.Time,
) (_ *transport.ClientStream, err error) {
	if req.Meta == nil {
		return nil, yarpcerrors.InvalidArgumentErrorf("stream request requires a request metadata")
	}
	treq := req.Meta.ToRequest()
	md, err := transportRequestToMetadata(treq)
	if err != nil {
		return nil, err
	}

	fullMethod, err := procedureNameToFullMethod(req.Meta.Procedure)
	if err != nil {
		return nil, err
	}

	apiPeer, onFinish, err := o.peerChooser.Choose(ctx, treq)
	if err != nil {
		return nil, err
	}

	grpcPeer, ok := apiPeer.(*grpcPeer)
	if !ok {
		err := peer.ErrInvalidPeerConversion{
			Peer:         apiPeer,
			ExpectedType: "*grpcPeer",
		}
		onFinish(err)
		return nil, err
	}

	tracer := o.t.options.tracer
	createOpenTracingSpan := &transport.CreateOpenTracingSpan{
		Tracer:        tracer,
		TransportName: TransportName,
		StartTime:     start,
		ExtraTags:     yarpc.OpentracingTags,
	}
	_, span := createOpenTracingSpan.Do(ctx, treq)

	if err := tracer.Inject(span.Context(), opentracing.HTTPHeaders, mdReadWriter(md)); err != nil {
		span.Finish()
		onFinish(err)
		return nil, err
	}

	streamCtx := metadata.NewOutgoingContext(ctx, md)
	clientStream, err := grpcPeer.clientConn.NewStream(
		streamCtx,
		&grpc.StreamDesc{
			ClientStreams: true,
			ServerStreams: true,
		},
		fullMethod,
	)
	if err != nil {
		span.Finish()
		onFinish(err)
		return nil, err
	}
	stream := newClientStream(streamCtx, req, clientStream, span, onFinish)
	tClientStream, err := transport.NewClientStream(stream)
	if err != nil {
		onFinish(err)
		span.Finish()
		return nil, err
	}
	return tClientStream, nil
}

// Only does verification when there is a response service header key
func checkServiceMatch(reqSvcName string, responseMD metadata.MD) (bool, string) {
	if resSvcName, ok := responseMD[ServiceHeader]; ok {
		return reqSvcName == resSvcName[0], resSvcName[0]
	}
	return true, ""
}
