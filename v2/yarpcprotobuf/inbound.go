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

package yarpcprotobuf

import (
	"context"

	"github.com/gogo/protobuf/proto"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcencoding"
)

// StreamHandlerParams contains the parameters for creating a new StreamHandler.
type StreamHandlerParams struct {
	Handle func(*ServerStream) error
}

type streamHandler struct {
	handle func(*ServerStream) error
}

// NewStreamHandler returns a new StreamHandler.
func NewStreamHandler(p StreamHandlerParams) yarpc.StreamHandler {
	return &streamHandler{p.Handle}
}

func (s *streamHandler) HandleStream(stream *yarpc.ServerStream) error {
	ctx, call := yarpc.NewInboundCall(stream.Context(), yarpc.DisableResponseHeaders())
	if err := call.ReadFromRequest(stream.Request()); err != nil {
		return err
	}
	protoStream := &ServerStream{
		ctx:    ctx,
		stream: stream,
	}
	return s.handle(protoStream)
}

// UnaryHandlerParams contains the parameters for creating a new UnaryHandler.
type UnaryHandlerParams struct {
	Handle func(context.Context, proto.Message) (proto.Message, error)
	Create func() proto.Message
}

type unaryHandler struct {
	handle func(context.Context, proto.Message) (proto.Message, error)
	create func() proto.Message
}

// NewUnaryHandler returns a new UnaryHandler.
func NewUnaryHandler(p UnaryHandlerParams) yarpc.UnaryHandler {
	return &unaryHandler{p.Handle, p.Create}
}

func (u *unaryHandler) Handle(ctx context.Context, req *yarpc.Request, buf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	ctx, call, request, err := toProtoRequest(ctx, req, buf, u.create)
	if err != nil {
		return nil, nil, err
	}

	response := &yarpc.Response{}
	responseBuf := &yarpc.Buffer{}
	call.WriteToResponse(response)

	protoResponse, appErr := u.handle(ctx, request)

	// If the proto response is nil, return early
	// so that we don't attempt to marshal a nil
	// object.
	if protoResponse == nil {
		if appErr != nil {
			response.ApplicationError = true
		}
		return response, responseBuf, appErr
	}

	body, cleanup, err := marshal(req.Encoding, protoResponse)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return response, responseBuf, yarpcencoding.ResponseBodyEncodeError(req, err)
	}
	if _, err := responseBuf.Write(body); err != nil {
		return response, responseBuf, err
	}
	if appErr != nil {
		// TODO(apeatsbond): now that we propogate a Response struct back, the
		// Response should hold the actual application error. Errors returned by the
		// handler (not through the Response) could be considered fatal.
		response.ApplicationError = true
	}
	return response, responseBuf, appErr
}

func toProtoRequest(
	ctx context.Context,
	req *yarpc.Request,
	body *yarpc.Buffer,
	create func() proto.Message,
) (context.Context, *yarpc.InboundCall, proto.Message, error) {
	if err := yarpcencoding.ExpectEncodings(req, _protoEncoding, _jsonEncoding); err != nil {
		return nil, nil, nil, err
	}
	ctx, call := yarpc.NewInboundCall(ctx)
	if err := call.ReadFromRequest(req); err != nil {
		return nil, nil, nil, err
	}
	request := create()
	if err := unmarshal(req.Encoding, body, request); err != nil {
		return nil, nil, nil, yarpcencoding.RequestBodyDecodeError(req, err)
	}
	return ctx, call, request, nil
}
