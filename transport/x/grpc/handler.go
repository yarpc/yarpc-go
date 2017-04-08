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
	procedureServiceName string
	serviceName          string
	methodName           string
	router               transport.Router
	onewayErrorHandler   func(error)
}

func newHandler(
	procedureServiceName string,
	serviceName string,
	methodName string,
	router transport.Router,
	onewayErrorHandler func(error),
) *handler {
	return &handler{
		procedureServiceName,
		serviceName,
		methodName,
		router,
		onewayErrorHandler,
	}
}

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
				h.getFullMethod(),
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
	caller, err := getCaller(md)
	if err != nil {
		return nil, err
	}
	if caller == "" {
		caller = h.serviceName
	}
	encoding, err := getEncoding(md)
	if err != nil {
		return nil, err
	}
	if encoding == "" {
		encoding = protobuf.Encoding
	}
	service, err := getService(md)
	if err != nil {
		return nil, err
	}
	if service == "" {
		service = h.procedureServiceName
	}
	headers, err := getApplicationHeaders(md)
	if err != nil {
		return nil, err
	}
	var data []byte
	if err := decodeFunc(&data); err != nil {
		return nil, err
	}
	transportRequest := &transport.Request{
		Caller:    caller,
		Encoding:  encoding,
		Service:   service,
		Procedure: procedureToName(h.serviceName, h.methodName),
		Headers:   headers,
		Body:      bytes.NewBuffer(data),
	}
	if err := transport.ValidateRequest(transportRequest); err != nil {
		return nil, err
	}
	return transportRequest, nil
}

func (h *handler) getFullMethod() string {
	return fmt.Sprintf("/%s/%s", h.serviceName, h.methodName)
}

func (h *handler) call(ctx context.Context, transportRequest *transport.Request) (interface{}, error) {
	handlerSpec, err := h.router.Choose(ctx, transportRequest)
	if err != nil {
		return nil, err
	}
	switch handlerSpec.Type() {
	case transport.Unary:
		return h.callUnary(ctx, transportRequest, handlerSpec.Unary())
	case transport.Oneway:
		return nil, h.callOneway(ctx, transportRequest, handlerSpec.Oneway())
	default:
		return nil, errors.UnsupportedTypeError{"grpc", handlerSpec.Type().String()}
	}
}

func (h *handler) callUnary(ctx context.Context, transportRequest *transport.Request, unaryHandler transport.UnaryHandler) (interface{}, error) {
	if err := request.ValidateUnaryContext(ctx); err != nil {
		return nil, err
	}
	protobuf.SetRawResponse(transportRequest.Headers)
	responseWriter := newResponseWriter()
	// TODO: always return data?
	err := transport.DispatchUnaryHandler(ctx, unaryHandler, time.Now(), transportRequest, responseWriter)
	err = multierr.Append(err, grpc.SendHeader(ctx, responseWriter.md))
	data := responseWriter.Bytes()
	return &data, err
}

func (h *handler) callOneway(ctx context.Context, transportRequest *transport.Request, onewayHandler transport.OnewayHandler) error {
	go func() {
		// TODO: http propagates this on a span
		// TODO: spinning up a new goroutine for every request
		// is potentially a memory leak
		// TODO: have to use context.Background() because context is cancelled in crossdock
		// other transport implementation seem to create their own context for calls, need to understand better
		// This will not propagate opentracing, for example
		// Right now just letting context propagation test fail
		h.onewayErrorHandler(transport.DispatchOnewayHandler(context.Background(), onewayHandler, transportRequest))
	}()
	return nil
}
