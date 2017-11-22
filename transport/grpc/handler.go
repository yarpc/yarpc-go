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
	"io"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/yarpcerrors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	gtransport "google.golang.org/grpc/transport"
)

var (
	// errInvalidGRPCStream is applied before yarpc so it's a raw GRPC error
	errInvalidGRPCStream = status.Error(codes.InvalidArgument, "received grpc request with invalid stream")
	errInvalidGRPCMethod = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "invalid stream method name for request")
)

type handler struct {
	i *Inbound
}

func newHandler(i *Inbound) *handler {
	return &handler{i: i}
}

func (h *handler) handle(srv interface{}, serverStream grpc.ServerStream) error {
	start := time.Now()
	// Grab context information from the stream's context.
	ctx := serverStream.Context()
	stream, ok := gtransport.StreamFromContext(ctx)
	if !ok {
		return errInvalidGRPCStream
	}

	var requestData []byte
	if err := serverStream.RecvMsg(&requestData); err != nil {
		return err
	}
	requestBuffer := bufferpool.Get()
	defer bufferpool.Put(requestBuffer)
	// Buffers are documented to always return a nil error.
	_, _ = requestBuffer.Write(requestData)

	responseWriter := newResponseWriter()
	defer responseWriter.Close()

	err := h.handleBeforeErrorConversion(ctx, stream.Method(), requestBuffer, responseWriter, start)
	err = handlerErrorToGRPCError(err, responseWriter)

	// Send the response attributes back and end the stream.
	if sendErr := serverStream.SendMsg(responseWriter.Bytes()); sendErr != nil {
		// We couldn't send the response.
		return sendErr
	}
	if responseWriter.md != nil {
		serverStream.SetTrailer(responseWriter.md)
	}
	if responseWriter.headerErr != nil {
		err = handlerErrorToGRPCError(responseWriter.headerErr, responseWriter)
	}
	return err
}

func (h *handler) handleBeforeErrorConversion(
	ctx context.Context,
	streamMethod string,
	requestReader io.Reader,
	responseWriter *responseWriter,
	start time.Time,
) error {
	transportRequest, err := h.getTransportRequest(ctx, streamMethod, requestReader)
	if err != nil {
		return err
	}

	tracer := h.i.t.options.tracer
	if tracer == nil {
		tracer = opentracing.GlobalTracer()
	}
	var parentSpanCtx opentracing.SpanContext
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		parentSpanCtx, _ = tracer.Extract(opentracing.HTTPHeaders, mdReadWriter(md))
	}
	extractOpenTracingSpan := &transport.ExtractOpenTracingSpan{
		ParentSpanContext: parentSpanCtx,
		Tracer:            tracer,
		TransportName:     transportName,
		StartTime:         start,
	}
	ctx, span := extractOpenTracingSpan.Do(ctx, transportRequest)
	defer span.Finish()

	err = h.call(ctx, transportRequest, responseWriter)
	return transport.UpdateSpanWithErr(span, err)
}

func (h *handler) getTransportRequest(ctx context.Context, streamMethod string, requestReader io.Reader) (*transport.Request, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if md == nil || !ok {
		return nil, yarpcerrors.Newf(yarpcerrors.CodeInternal, "cannot get metadata from ctx: %v", ctx)
	}
	transportRequest, err := metadataToTransportRequest(md)
	if err != nil {
		return nil, err
	}
	transportRequest.Body = requestReader
	procedure, err := procedureFromStreamMethod(streamMethod)
	if err != nil {
		return nil, err
	}
	transportRequest.Procedure = procedure
	if err := transport.ValidateRequest(transportRequest); err != nil {
		return nil, err
	}
	return transportRequest, nil
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

func (h *handler) call(ctx context.Context, transportRequest *transport.Request, responseWriter *responseWriter) error {
	handlerSpec, err := h.i.router.Choose(ctx, transportRequest)
	if err != nil {
		return err
	}
	switch handlerSpec.Type() {
	case transport.Unary:
		return h.callUnary(ctx, transportRequest, handlerSpec.Unary(), responseWriter)
	default:
		return yarpcerrors.Newf(yarpcerrors.CodeUnimplemented, "transport grpc does not handle %s handlers", handlerSpec.Type().String())
	}
}

func (h *handler) callUnary(ctx context.Context, transportRequest *transport.Request, unaryHandler transport.UnaryHandler, responseWriter *responseWriter) error {
	if err := transport.ValidateUnaryContext(ctx); err != nil {
		return err
	}
	return transport.DispatchUnaryHandler(ctx, unaryHandler, time.Now(), transportRequest, responseWriter)
}

func handlerErrorToGRPCError(err error, responseWriter *responseWriter) error {
	if err == nil {
		return nil
	}
	// if this is an error created from grpc-go, return the error
	if _, ok := status.FromError(err); ok {
		return err
	}
	// if this is not a yarpc error, return the error
	// this will result in the error being a grpc-go error with codes.Unknown
	if !yarpcerrors.IsStatus(err) {
		return err
	}
	// we now know we have a yarpc error
	yarpcStatus := yarpcerrors.FromError(err)
	name := yarpcStatus.Name()
	message := yarpcStatus.Message()
	// if the yarpc error has a name, set the header
	if name != "" {
		responseWriter.AddSystemHeader(ErrorNameHeader, name)
		if message == "" {
			// if the message is empty, set the message to the name for grpc compatibility
			message = name
		} else {
			// else, we set the name as the prefix for grpc compatibility
			// we parse this off the front if the name header is set on the client-side
			message = name + ": " + message
		}
	}
	grpcCode, ok := _codeToGRPCCode[yarpcStatus.Code()]
	// should only happen if _codeToGRPCCode does not cover all codes
	if !ok {
		grpcCode = codes.Unknown
	}
	return status.Error(grpcCode, message)
}
