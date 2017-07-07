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
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/x/grpc/grpcheader"
	"go.uber.org/yarpc/yarpcerrors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type handler struct {
	grpcServiceName string
	grpcMethodName  string
	router          transport.Router
}

func newHandler(
	grpcServiceName string,
	grpcMethodName string,
	router transport.Router,
) *handler {
	return &handler{
		grpcServiceName: grpcServiceName,
		grpcMethodName:  grpcMethodName,
		router:          router,
	}
}

// the grpc-go handler does not put the context.Context as the first argument
// so we must ignore this file for linting

func (h *handler) handle(
	server interface{},
	ctx context.Context,
	decodeFunc func(interface{}) error,
	interceptor grpc.UnaryServerInterceptor,
) (interface{}, error) {
	responseMD := metadata.New(nil)
	response, err := h.handleBeforeErrorConversion(server, ctx, decodeFunc, interceptor, responseMD)
	err = handlerErrorToGRPCError(err, responseMD)
	// TODO: what to do with error?
	_ = grpc.SetTrailer(ctx, responseMD)
	return response, err
}

func (h *handler) handleBeforeErrorConversion(
	server interface{},
	ctx context.Context,
	decodeFunc func(interface{}) error,
	interceptor grpc.UnaryServerInterceptor,
	responseMD metadata.MD,
) (interface{}, error) {
	transportRequest, err := h.getTransportRequest(ctx, decodeFunc)
	if err != nil {
		return nil, err
	}
	if interceptor != nil {
		return interceptor(
			ctx,
			transportRequest,
			&grpc.UnaryServerInfo{
				noopGrpcStruct{},
				toFullMethod(h.grpcServiceName, h.grpcMethodName),
			},
			func(ctx context.Context, request interface{}) (interface{}, error) {
				transportRequest, ok := request.(*transport.Request)
				if !ok {
					return nil, yarpcerrors.InternalErrorf("expected *transport.Request, got %T", request)
				}
				return h.call(ctx, transportRequest, responseMD)
			},
		)
	}
	return h.call(ctx, transportRequest, responseMD)
}

func (h *handler) getTransportRequest(ctx context.Context, decodeFunc func(interface{}) error) (*transport.Request, error) {
	md, ok := metadata.FromContext(ctx)
	if md == nil || !ok {
		return nil, yarpcerrors.InternalErrorf("cannot get metadata from ctx: %v", ctx)
	}
	transportRequest, err := metadataToTransportRequest(md)
	if err != nil {
		return nil, err
	}
	var data []byte
	if err := decodeFunc(&data); err != nil {
		return nil, err
	}
	transportRequest.Body = bytes.NewBuffer(data)
	procedure, err := procedureToName(h.grpcServiceName, h.grpcMethodName)
	if err != nil {
		return nil, err
	}
	transportRequest.Procedure = procedure
	if err := transport.ValidateRequest(transportRequest); err != nil {
		return nil, err
	}
	return transportRequest, nil
}

func (h *handler) call(ctx context.Context, transportRequest *transport.Request, responseMD metadata.MD) (interface{}, error) {
	handlerSpec, err := h.router.Choose(ctx, transportRequest)
	if err != nil {
		return nil, err
	}
	switch handlerSpec.Type() {
	case transport.Unary:
		return h.callUnary(ctx, transportRequest, handlerSpec.Unary(), responseMD)
	default:
		return nil, yarpcerrors.UnimplementedErrorf("transport grpc does not handle %s handlers", handlerSpec.Type().String())
	}
}

func (h *handler) callUnary(ctx context.Context, transportRequest *transport.Request, unaryHandler transport.UnaryHandler, responseMD metadata.MD) (interface{}, error) {
	if err := transport.ValidateUnaryContext(ctx); err != nil {
		return nil, err
	}
	responseWriter := newResponseWriter(responseMD)
	// TODO: do we always want to return the data from responseWriter.Bytes, or return nil for the data if there is an error?
	// For now, we are always returning the data
	err := transport.DispatchUnaryHandler(ctx, unaryHandler, time.Now(), transportRequest, responseWriter)
	// TODO: use pooled buffers
	// we have to return the data up the stack, but we can probably do something complicated
	// with the Codec where we put the buffer back on Marshal
	data := responseWriter.Bytes()
	return data, err
}

func handlerErrorToGRPCError(err error, responseMD metadata.MD) error {
	if err == nil {
		return nil
	}
	// if this is an error created from grpc-go, return the error
	if _, ok := status.FromError(err); ok {
		return err
	}
	// if this is not a yarpc error, return the error
	// this will result in the error being a grpc-go error with codes.Unknown
	if !yarpcerrors.IsYARPCError(err) {
		return err
	}
	name := yarpcerrors.ErrorName(err)
	message := yarpcerrors.ErrorMessage(err)
	// if the yarpc error has a name, set the header
	if name != "" {
		// TODO: what to do with error?
		_ = addToMetadata(responseMD, grpcheader.ErrorNameHeader, name)
		if message == "" {
			// if the message is empty, set the message to the name for grpc compatibility
			message = name
		} else {
			// else, we set the name as the prefix for grpc compatibility
			// we parse this off the front if the name header is set on the client-side
			message = name + ": " + message
		}
	}
	grpcCode, ok := _codeToGRPCCode[yarpcerrors.ErrorCode(err)]
	// should only happen if _codeToGRPCCode does not cover all codes
	if !ok {
		grpcCode = codes.Unknown
	}
	return status.Error(grpcCode, message)
}
