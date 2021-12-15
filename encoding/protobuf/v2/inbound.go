// Copyright (c) 2021 Uber Technologies, Inc.
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

package v2

import (
	"context"

	"github.com/golang/protobuf/proto"
	apiencoding "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/errors"
)

type unaryHandler struct {
	handle     func(context.Context, proto.Message) (proto.Message, error)
	newRequest func() proto.Message
	codec      *codec
}

func newUnaryHandler(
	handle func(context.Context, proto.Message) (proto.Message, error),
	newRequest func() proto.Message,
	codec *codec,
) *unaryHandler {
	return &unaryHandler{
		handle:     handle,
		newRequest: newRequest,
		codec:      codec,
	}
}

func (u *unaryHandler) Handle(ctx context.Context, transportRequest *transport.Request, responseWriter transport.ResponseWriter) error {
	ctx, call, request, err := getProtoRequest(ctx, transportRequest, u.newRequest, u.codec)
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
		responseData, responseCleanup, err = marshal(transportRequest.Encoding, response, u.codec)
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
	return convertToYARPCError(transportRequest.Encoding, appErr, u.codec, responseWriter)
}

type onewayHandler struct {
	handleOneway func(context.Context, proto.Message) error
	newRequest   func() proto.Message
	codec        *codec
}

func newOnewayHandler(
	handleOneway func(context.Context, proto.Message) error,
	newRequest func() proto.Message,
	codec *codec,
) *onewayHandler {
	return &onewayHandler{
		handleOneway: handleOneway,
		newRequest:   newRequest,
		codec:        codec,
	}
}

func (o *onewayHandler) HandleOneway(ctx context.Context, transportRequest *transport.Request) error {
	ctx, _, request, err := getProtoRequest(ctx, transportRequest, o.newRequest, o.codec)
	if err != nil {
		return err
	}
	return convertToYARPCError(transportRequest.Encoding, o.handleOneway(ctx, request), o.codec, nil /*responseWriter*/)
}

type streamHandler struct {
	handle func(*ServerStream) error
	codec  *codec
}

func newStreamHandler(handle func(*ServerStream) error) *streamHandler {
	return &streamHandler{handle, newCodec(nil /*AnyResolver*/)}
}

func (s *streamHandler) HandleStream(stream *transport.ServerStream) error {
	ctx, call := apiencoding.NewInboundCallWithOptions(stream.Context(), apiencoding.DisableResponseHeaders())
	transportRequest := stream.Request()
	if err := call.ReadFromRequestMeta(transportRequest.Meta); err != nil {
		return err
	}
	protoStream := &ServerStream{
		ctx:    ctx,
		stream: stream,
		codec:  s.codec,
	}
	return convertToYARPCError(transportRequest.Meta.Encoding, s.handle(protoStream), s.codec, nil /*responseWriter*/)
}

func getProtoRequest(ctx context.Context, transportRequest *transport.Request, newRequest func() proto.Message, codec *codec) (context.Context, *apiencoding.InboundCall, proto.Message, error) {
	if err := errors.ExpectEncodings(transportRequest, Encoding, JSONEncoding); err != nil {
		return nil, nil, nil, err
	}
	ctx, call := apiencoding.NewInboundCall(ctx)
	if err := call.ReadFromRequest(transportRequest); err != nil {
		return nil, nil, nil, err
	}
	request := newRequest()
	if err := unmarshal(transportRequest.Encoding, transportRequest.Body, request, codec); err != nil {
		return nil, nil, nil, errors.RequestBodyDecodeError(transportRequest, err)
	}
	return ctx, call, request, nil
}
