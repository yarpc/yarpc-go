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

package grpc

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/v2"
	intyarpcerror "go.uber.org/yarpc/v2/internal/internalyarpcerror"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/yarpc/v2/yarpctracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UserAgent is the User-Agent that will be set for requests.
// http://www.grpc.io/docs/guides/wire.html#user-agents
const UserAgent = "yarpc-go/" + yarpc.Version

var _ yarpc.UnaryOutbound = (*Outbound)(nil)

// Outbound sends YARPC requests over gRPC. It is recommended that services use
// a single HTTP dialer to construct all HTTP outbounds, ensuring efficient
// sharing of resources across the different outbounds.
type Outbound struct {
	// Chooser is a peer chooser for outbound requests.
	Chooser yarpc.Chooser

	// Dialer is an alternative to specifying a Chooser.
	// The outbound will dial the address specified in the URL.
	Dialer yarpc.Dialer

	// URL specifies the template for the URL this outbound makes requests to.
	// For yarpc.Chooser-based outbounds, the peer (host:port) spection of the
	// URL may vary from call to call but the REST will remain unchanged.
	// For single-peer outbounds, the URL will be used as-is.
	URL *url.URL

	// Tracer attaches a tracer for the outbound.
	Tracer opentracing.Tracer
}

// Call implements yarpc.UnaryOutbound#Call.
func (o *Outbound) Call(ctx context.Context, req *yarpc.Request, reqBuf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	if req == nil {
		return nil, nil, yarpcerror.InvalidArgumentErrorf("request for grpc outbound was nil")
	}

	start := time.Now()

	var responseBody []byte
	var responseMD metadata.MD
	invokeErr := o.invoke(ctx, req, reqBuf, &responseBody, &responseMD, start)

	responseHeaders, err := getApplicationHeaders(responseMD)
	if err != nil {
		return nil, nil, err
	}
	return &yarpc.Response{
			Headers:          responseHeaders,
			ApplicationError: metadataToIsApplicationError(responseMD),
		},
		yarpc.NewBufferBytes(responseBody),
		invokeErr
}

func (o *Outbound) invoke(
	ctx context.Context,
	req *yarpc.Request,
	reqBuf *yarpc.Buffer,
	responseBody *[]byte,
	responseMD *metadata.MD,
	start time.Time,
) (retErr error) {
	md, err := requestToMetadata(req)
	if err != nil {
		return err
	}

	fullMethod, err := procedureNameToFullMethod(req.Procedure)
	if err != nil {
		return err
	}
	var callOptions []grpc.CallOption
	if responseMD != nil {
		callOptions = []grpc.CallOption{grpc.Trailer(responseMD)}
	}
	apiPeer, onFinish, err := o.Chooser.Choose(ctx, req)
	if err != nil {
		return err
	}
	defer func() { onFinish(retErr) }()
	grpcPeer, ok := apiPeer.(*grpcPeer)
	if !ok {
		return yarpcpeer.ErrInvalidPeerConversion{
			Peer:         apiPeer,
			ExpectedType: "*grpcPeer",
		}
	}

	createOpenTracingSpan := &yarpctracing.CreateOpenTracingSpan{
		Tracer:        o.Tracer,
		TransportName: transportName,
		StartTime:     start,
		ExtraTags:     yarpctracing.Tags,
	}
	ctx, span := createOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	if err := o.Tracer.Inject(span.Context(), opentracing.HTTPHeaders, mdReadWriter(md)); err != nil {
		return err
	}

	err = yarpctracing.UpdateSpanWithErr(
		span,
		grpcPeer.clientConn.Invoke(
			metadata.NewOutgoingContext(ctx, md),
			fullMethod,
			reqBuf.Bytes(),
			responseBody,
			callOptions...,
		),
	)
	if err != nil {
		return invokeErrorToYARPCError(err, *responseMD)
	}
	// Service name match validation, return yarpcerror.CodeInternal error if not match
	if match, resService := checkServiceMatch(req.Service, *responseMD); !match {
		// If service doesn't match => we got response => span must not be nil
		return yarpctracing.UpdateSpanWithErr(span, yarpcerror.InternalErrorf("service name sent from the request "+
			"does not match the service name received in the response: sent %q, got: %q", req.Service, resService))
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
	if yarpcerror.IsStatus(err) {
		return err
	}
	status, ok := status.FromError(err)
	// if not a yarpc error or grpc error, just return a wrapped error
	if !ok {
		return yarpcerror.FromError(err)
	}
	code, ok := _grpcCodeToCode[status.Code()]
	if !ok {
		code = yarpcerror.CodeUnknown
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
	return intyarpcerror.NewWithNamef(code, name, message)
}

// CallStream implements yarpc.StreamOutbound#CallStream.
func (o *Outbound) CallStream(ctx context.Context, request *yarpc.Request) (*yarpc.ClientStream, error) {
	return o.stream(ctx, request, time.Now())
}

func (o *Outbound) stream(
	ctx context.Context,
	req *yarpc.Request,
	start time.Time,
) (_ *yarpc.ClientStream, err error) {
	if req == nil {
		return nil, yarpcerror.InvalidArgumentErrorf("stream request requires a request metadata")
	}
	md, err := requestToMetadata(req)
	if err != nil {
		return nil, err
	}

	fullMethod, err := procedureNameToFullMethod(req.Procedure)
	if err != nil {
		return nil, err
	}

	apiPeer, onFinish, err := o.Chooser.Choose(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { onFinish(err) }()

	grpcPeer, ok := apiPeer.(*grpcPeer)
	if !ok {
		return nil, yarpcpeer.ErrInvalidPeerConversion{
			Peer:         apiPeer,
			ExpectedType: "*grpcPeer",
		}
	}

	createOpenTracingSpan := &yarpctracing.CreateOpenTracingSpan{
		Tracer:        o.Tracer,
		TransportName: transportName,
		StartTime:     start,
		ExtraTags:     yarpctracing.Tags,
	}
	_, span := createOpenTracingSpan.Do(ctx, req)

	if err := o.Tracer.Inject(span.Context(), opentracing.HTTPHeaders, mdReadWriter(md)); err != nil {
		span.Finish()
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
		return nil, err
	}
	stream := newClientStream(streamCtx, req, clientStream, span)
	tClientStream, err := yarpc.NewClientStream(stream)
	if err != nil {
		span.Finish()
		return nil, err
	}
	return tClientStream, nil
}

// Only does verification when there is a response service header key
func checkServiceMatch(reqServiceName string, responseMD metadata.MD) (bool, string) {
	if resService, ok := responseMD[ServiceHeader]; ok {
		return reqServiceName == resService[0], resService[0]
	}
	return true, ""
}
