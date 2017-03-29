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

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/x/protobuf"
	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/procedure"
	"go.uber.org/yarpc/internal/request"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type methodHandler struct {
	procedureServiceName string
	serviceName          string
	methodName           string
	router               transport.Router
}

func newMethodHandler(
	procedureServiceName string,
	serviceName string,
	methodName string,
	router transport.Router,
) *methodHandler {
	return &methodHandler{procedureServiceName, serviceName, methodName, router}
}

func (m *methodHandler) handle(
	server interface{},
	ctx context.Context,
	decodeFunc func(interface{}) error,
	interceptor grpc.UnaryServerInterceptor,
) (interface{}, error) {
	transportRequest, err := m.getTransportRequest(ctx, decodeFunc)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v\n", transportRequest)
	if interceptor != nil {
		return interceptor(
			ctx,
			transportRequest,
			&grpc.UnaryServerInfo{
				noopGrpcStruct{},
				m.getFullMethod(),
			},
			func(ctx context.Context, request interface{}) (interface{}, error) {
				transportRequest, ok := request.(*transport.Request)
				if !ok {
					return nil, fmt.Errorf("expected *transport.Request, got %T", request)
				}
				return m.call(ctx, transportRequest)
			},
		)
	}
	return m.call(ctx, transportRequest)
}

func (m *methodHandler) getTransportRequest(ctx context.Context, decodeFunc func(interface{}) error) (*transport.Request, error) {
	md, ok := metadata.FromContext(ctx)
	if md == nil || !ok {
		return nil, fmt.Errorf("cannot get metadata from ctx: %v", ctx)
	}
	caller, err := getCaller(md)
	if err != nil {
		return nil, err
	}
	if caller == "" {
		caller = m.serviceName
	}
	encoding, err := getEncoding(md)
	if err != nil {
		return nil, err
	}
	if encoding == "" {
		encoding = protobuf.Encoding
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
		Service:   m.procedureServiceName,
		Procedure: procedure.ToName(m.serviceName, m.methodName),
		Headers:   headers,
		Body:      bytes.NewBuffer(data),
	}
	if err := transport.ValidateRequest(transportRequest); err != nil {
		return nil, err
	}
	return transportRequest, nil
}

func (m *methodHandler) getFullMethod() string {
	return fmt.Sprintf("/%s/%s", m.serviceName, m.methodName)
}

func (m *methodHandler) call(ctx context.Context, transportRequest *transport.Request) (interface{}, error) {
	handlerSpec, err := m.router.Choose(ctx, transportRequest)
	if err != nil {
		return nil, err
	}
	switch handlerSpec.Type() {
	case transport.Unary:
		return m.callUnary(ctx, transportRequest, handlerSpec.Unary())
	default:
		return nil, errors.UnsupportedTypeError{"grpc", handlerSpec.Type().String()}
	}
}

func (m *methodHandler) callUnary(ctx context.Context, transportRequest *transport.Request, unaryHandler transport.UnaryHandler) (interface{}, error) {
	if err := request.ValidateUnaryContext(ctx); err != nil {
		return nil, err
	}
	responseWriter := newResponseWriter()
	// TODO: always return data?
	err := transport.DispatchUnaryHandler(ctx, unaryHandler, time.Now(), transportRequest, responseWriter)
	err = errors.CombineErrors(err, grpc.SendHeader(ctx, responseWriter.md))
	data := responseWriter.Bytes()
	//log.Printf("%s %v\n", string(data), err)
	return &data, err
}
