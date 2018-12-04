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

package yarpcgrpc

import (
	"fmt"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpcprotobuf"
	"go.uber.org/yarpc/v2/yarpcthrift"
	"go.uber.org/yarpc/v2/yarpctracing"
	"go.uber.org/yarpc/v2/yarpctransport"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	// errInvalidGRPCStream is applied before yarpc so it's a raw GRPC error
	errInvalidGRPCStream = status.Error(codes.InvalidArgument, "received grpc request with invalid stream")
	errInvalidGRPCMethod = yarpcerror.New(yarpcerror.CodeInvalidArgument, "invalid stream method name for request")
)

type handler struct {
	i *Inbound
}

func newHandler(i *Inbound) *handler {
	return &handler{i: i}
}

func (h *handler) handle(srv interface{}, serverStream grpc.ServerStream) error {
	start := time.Now()
	ctx := serverStream.Context()

	streamMethod, ok := grpc.MethodFromServerStream(serverStream)
	if !ok {
		return errInvalidGRPCStream
	}

	req, err := requestFromServerStream(serverStream, streamMethod)
	if err != nil {
		return err
	}

	handlerSpec, err := h.i.Router.Choose(ctx, req)
	if err != nil {
		return err
	}

	// extract open tracing span
	tracer := h.i.Tracer
	var parentSpanCtx opentracing.SpanContext
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		parentSpanCtx, _ = tracer.Extract(opentracing.HTTPHeaders, mdReadWriter(md))
	}
	extractOpenTracingSpan := &yarpctracing.ExtractOpenTracingSpan{
		ParentSpanContext: parentSpanCtx,
		Tracer:            tracer,
		TransportName:     transportName,
		StartTime:         start,
		ExtraTags:         yarpctracing.Tags,
	}
	ctx, span := extractOpenTracingSpan.Do(ctx, req)
	defer span.Finish()

	// invoke handler
	switch handlerSpec.Type() {
	case yarpc.Unary:
		err = h.handleUnary(ctx, req, serverStream, start, handlerSpec.Unary())
	case yarpc.Streaming:
		err = h.handleStream(ctx, req, serverStream, start, handlerSpec.Stream())
	default:
		return yarpcerror.New(yarpcerror.CodeUnimplemented, fmt.Sprintf("gRPC inbound does not handle %s handlers", handlerSpec.Type().String()))
	}

	return yarpctracing.UpdateSpanWithErr(span, err)
}

// requestFromServerStream converts the grpc request metadata into a yarpc.Request
func requestFromServerStream(stream grpc.ServerStream, streamMethod string) (*yarpc.Request, error) {
	ctx := stream.Context()

	md, ok := metadata.FromIncomingContext(ctx)
	if md == nil || !ok {
		return nil, yarpcerror.New(yarpcerror.CodeInternal, fmt.Sprintf("cannot get metadata from ctx: %v", ctx))
	}
	req, err := metadataToRequest(md)
	if err != nil {
		return nil, err
	}
	req.Transport = transportName

	procedure, err := procedureFromStreamMethod(streamMethod)
	if err != nil {
		return nil, err
	}

	req.Procedure = procedure
	if err := yarpc.ValidateRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// procedureFromStreamMethod converts a GRPC stream method into a yarpc
// procedure name.  This is mostly copied from the GRPC-go server processing
// logic here:
// https://github.com/grpc/grpc-go/blob/d6723916d2e73e8824d22a1ba5c52f8e6255e6f8/server.go#L931-L956
func procedureFromStreamMethod(streamMethod string) (string, error) {
	if streamMethod != "" && streamMethod[0] == '/' {
		streamMethod = streamMethod[1:]
	}
	pos := strings.LastIndex(streamMethod, "/")
	if pos == -1 {
		return "", errInvalidGRPCMethod
	}
	service := streamMethod[:pos]
	method := streamMethod[pos+1:]
	return procedureToName(service, method)
}

func (h *handler) handleStream(
	ctx context.Context,
	req *yarpc.Request,
	gServerStream grpc.ServerStream,
	start time.Time,
	streamHandler yarpc.StreamTransportHandler,
) error {
	serverStream, err := yarpc.NewServerStream(newServerStream(ctx, req, gServerStream))
	if err != nil {
		return toGRPCStreamError(err)
	}

	err = yarpctransport.InvokeStreamHandler(yarpctransport.StreamInvokeRequest{
		Stream:  serverStream,
		Handler: streamHandler,
		Logger:  h.i.Logger,
	})

	return toGRPCStreamError(err)
}

func (h *handler) handleUnary(
	ctx context.Context,
	req *yarpc.Request,
	serverStream grpc.ServerStream,
	start time.Time,
	handler yarpc.UnaryTransportHandler,
) error {
	var requestData []byte
	if err := serverStream.RecvMsg(&requestData); err != nil {
		return err
	}

	mdWriter := newMetadataWriter()

	// Add address of handler.
	mdWriter.AddSystemHeader(PeerHeader, h.i.Listener.Addr().String())
	// Echo accepted rpc-service in response header
	mdWriter.AddSystemHeader(ServiceHeader, req.Service)

	if err := yarpc.ValidateRequestContext(ctx); err != nil {
		return err
	}
	res, resBuf, err := yarpctransport.InvokeUnaryHandler(yarpctransport.UnaryInvokeRequest{
		Context:   ctx,
		StartTime: time.Now(),
		Request:   req,
		Buffer:    yarpc.NewBufferBytes(requestData),
		Handler:   handler,
		Logger:    h.i.Logger,
	})

	if err != nil {
		err = handlerErrorToGRPCError(err, mdWriter)
		serverStream.SetTrailer(mdWriter.MD())
		return err
	}

	if err := handleResponse(req.Encoding, res.ApplicationErrorInfo, resBuf, serverStream, mdWriter); err != nil {
		return err
	}

	mdWriter.SetResponseHeaders(res)
	serverStream.SetTrailer(mdWriter.MD())
	return nil
}

func handlerErrorToGRPCError(err error, mdWriter *metadataWriter) error {
	// if this is an error created from grpc-go, return the error
	if _, ok := status.FromError(err); ok {
		return err
	}
	// we now know we have a yarpc error
	errorInfo := yarpcerror.ExtractInfo(err)
	name := errorInfo.Name
	message := errorInfo.Message
	// if the yarpc error has a name, set the header
	if name != "" {
		mdWriter.AddSystemHeader(ErrorNameHeader, name)
		if message == "" {
			// if the message is empty, set the message to the name for grpc compatibility
			message = name
		} else {
			// else, we set the name as the prefix for grpc compatibility
			// we parse this off the front if the name header is set on the client-side
			message = name + ": " + message
		}
	}
	grpcCode, ok := _codeToGRPCCode[errorInfo.Code]
	// should only happen if _codeToGRPCCode does not cover all codes
	if !ok {
		grpcCode = codes.Unknown
	}
	return status.Error(grpcCode, message)
}

func handleResponse(
	encoding yarpc.Encoding,
	errInfo *yarpcerror.Info,
	resBuf *yarpc.Buffer,
	serverStream grpc.ServerStream,
	mdWriter *metadataWriter,
) error {
	// This is a regular response that we should send
	if errInfo == nil && resBuf != nil {
		return serverStream.SendMsg(resBuf.Bytes())
	}

	// This is an application error
	if errInfo != nil {
		if errInfo.Name != "" {
			mdWriter.AddSystemHeader(ErrorNameHeader, errInfo.Name)
		}
		if resBuf != nil {
			switch encoding {
			case yarpcprotobuf.Encoding:
				mdWriter.AddSystemHeader(ErrorDetailsHeader, resBuf.String())
			case yarpcjson.Encoding, yarpcthrift.Encoding:
				mdWriter.AddSystemHeader(ApplicationErrorHeader, resBuf.String())
			}
		}
	}
	return nil
}
