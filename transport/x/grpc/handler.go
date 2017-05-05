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
	"fmt"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/x/protobuf"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/request"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type handler struct {
	yarpcServiceName string
	grpcServiceName  string
	grpcMethodName   string
	encoding         transport.Encoding
	router           transport.Router
}

func newHandler(
	yarpcServiceName string,
	grpcServiceName string,
	grpcMethodName string,
	encoding transport.Encoding,
	router transport.Router,
) *handler {
	return &handler{
		yarpcServiceName,
		grpcServiceName,
		grpcMethodName,
		encoding,
		router,
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
					return nil, fmt.Errorf("expected *transport.Request, got %T", request)
				}
				return h.call(ctx, transportRequest)
			},
		)
	}
	return h.call(ctx, transportRequest)
}

func (h *handler) getTransportRequest(ctx context.Context, decodeFunc func(interface{}) error) (*transport.Request, error) {
	md, ok := metadata.FromContext(ctx)
	if md == nil || !ok {
		return nil, fmt.Errorf("cannot get metadata from ctx: %v", ctx)
	}
	transportRequest, err := metadataToTransportRequest(md)
	if err != nil {
		return nil, err
	}
	if transportRequest.Encoding == "" {
		transportRequest.Encoding = h.encoding
	}
	// We must do this to indicate to the protobuf encoding that we
	// need to return the raw response object over this transport.
	//
	// See the commentary within encoding/x/protobuf/inbound.go.
	transportRequest.Headers = protobuf.SetRawResponse(transportRequest.Headers)
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

func (h *handler) call(ctx context.Context, transportRequest *transport.Request) (interface{}, error) {
	handlerSpec, err := h.router.Choose(ctx, transportRequest)
	if err != nil {
		return nil, err
	}
	switch handlerSpec.Type() {
	case transport.Unary:
		return h.callUnary(ctx, transportRequest, handlerSpec.Unary())
	default:
		return nil, errors.UnsupportedTypeError{"grpc", handlerSpec.Type().String()}
	}
}

func (h *handler) callUnary(ctx context.Context, transportRequest *transport.Request, unaryHandler transport.UnaryHandler) (interface{}, error) {
	if err := request.ValidateUnaryContext(ctx); err != nil {
		return nil, err
	}
	responseWriter := newResponseWriter()
	// TODO: do we always want to return the data from responseWriter.Bytes, or return nil for the data if there is an error?
	// For now, we are always returning the data
	err := transport.DispatchUnaryHandler(ctx, unaryHandler, time.Now(), transportRequest, responseWriter)
	err = multierr.Append(err, grpc.SendHeader(ctx, responseWriter.md))
	data := responseWriter.Bytes()
	return data, err
}
