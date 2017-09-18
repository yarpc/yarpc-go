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

package protobuf

import (
	"context"

	"github.com/gogo/protobuf/proto"
	apiencoding "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/errors"
)

type unaryHandler struct {
	handle     func(context.Context, proto.Message) (proto.Message, error)
	newRequest func() proto.Message
}

func newUnaryHandler(
	handle func(context.Context, proto.Message) (proto.Message, error),
	newRequest func() proto.Message,
) *unaryHandler {
	return &unaryHandler{handle, newRequest}
}

func (u *unaryHandler) Handle(ctx context.Context, transportRequest *transport.Request, responseWriter transport.ResponseWriter) error {
	ctx, call, request, err := getProtoRequest(ctx, transportRequest, u.newRequest)
	if err != nil {
		return err
	}

	response, appErr := u.handle(ctx, request)

	if err := call.WriteToResponse(responseWriter); err != nil {
		return err
	}
	var responseData []byte
	var responseCleanup func()
	if response != nil {
		responseData, responseCleanup, err = marshal(transportRequest.Encoding, response)
		if responseCleanup != nil {
			defer responseCleanup()
		}
		if err != nil {
			return errors.ResponseBodyEncodeError(transportRequest, err)
		}
	}
	_, err = responseWriter.Write(responseData)
	if err != nil {
		return err
	}
	if appErr != nil {
		responseWriter.SetApplicationError()
	}
	return appErr
}

type onewayHandler struct {
	handleOneway func(context.Context, proto.Message) error
	newRequest   func() proto.Message
}

func newOnewayHandler(
	handleOneway func(context.Context, proto.Message) error,
	newRequest func() proto.Message,
) *onewayHandler {
	return &onewayHandler{handleOneway, newRequest}
}

func (o *onewayHandler) HandleOneway(ctx context.Context, transportRequest *transport.Request) error {
	ctx, _, request, err := getProtoRequest(ctx, transportRequest, o.newRequest)
	if err != nil {
		return err
	}
	return o.handleOneway(ctx, request)
}

func getProtoRequest(ctx context.Context, transportRequest *transport.Request, newRequest func() proto.Message) (context.Context, *apiencoding.InboundCall, proto.Message, error) {
	if err := errors.ExpectEncodings(transportRequest, Encoding, JSONEncoding); err != nil {
		return nil, nil, nil, err
	}
	ctx, call := apiencoding.NewInboundCall(ctx)
	if err := call.ReadFromRequest(transportRequest); err != nil {
		return nil, nil, nil, err
	}
	request := newRequest()
	if err := unmarshal(transportRequest.Encoding, transportRequest.Body, request); err != nil {
		return nil, nil, nil, errors.RequestBodyDecodeError(transportRequest, err)
	}
	return ctx, call, request, nil
}

type streamHandler struct {
	handle func(ServerStream) error
}

func newStreamHandler(
	handle func(ServerStream) error,
) *streamHandler {
	return &streamHandler{handle}
}

func (s *streamHandler) HandleStream(stream transport.ServerStream) error {
	return s.handle(stream)
}
