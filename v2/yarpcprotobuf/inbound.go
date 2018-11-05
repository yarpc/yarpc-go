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
	"go.uber.org/yarpc/v2/yarpcjson"
)

// StreamHandlerParams contains the parameters for creating a new StreamTransportHandler.
type StreamHandlerParams struct {
	Handle func(*ServerStream) error
}

type streamHandler struct {
	handle func(*ServerStream) error
}

// NewStreamHandler returns a new StreamTransportHandler.
func NewStreamHandler(p StreamHandlerParams) yarpc.StreamTransportHandler {
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

// UnaryHandlerParams contains the parameters for creating a new UnaryTransportHandler.
type UnaryHandlerParams struct {
	Handle      func(context.Context, proto.Message) (proto.Message, error)
	RequestType proto.Message
}

type unaryHandler struct {
	handle      func(context.Context, proto.Message) (proto.Message, error)
	requestType proto.Message
}

// NewUnaryHandler returns a new UnaryHandler.
func NewUnaryHandler(p UnaryHandlerParams) yarpc.UnaryTransportHandler {
	return &unaryHandler{p.Handle, p.RequestType}
}

func (u *unaryHandler) Handle(ctx context.Context, req *yarpc.Request, buf *yarpc.Buffer) (*yarpc.Response, *yarpc.Buffer, error) {
	ctx, call, protoReq, err := toProtoRequest(ctx, req, buf, u.requestType)
	if err != nil {
		return nil, nil, err
	}

	res := &yarpc.Response{}
	resBuf := &yarpc.Buffer{}

	protoRes, appErr := u.handle(ctx, protoReq)
	call.WriteToResponse(res)

	// If the application error is not nil, return
	// early so that we don't attempt to marshal a nil
	// object.
	if appErr != nil {
		// TODO: This is a bit odd; we set the error in response AND return it.
		// However, to preserve the current behavior of YARPC, this is
		// necessary. This is most likely where the error details will be added,
		// so we expect this to change.
		res.ApplicationError = appErr
		return res, resBuf, appErr
	}

	body, cleanup, err := marshal(req.Encoding, protoRes)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return res, resBuf, yarpcencoding.ResponseBodyEncodeError(req, err)
	}
	if _, err := resBuf.Write(body); err != nil {
		return res, resBuf, err
	}
	return res, resBuf, appErr
}

func toProtoRequest(
	ctx context.Context,
	req *yarpc.Request,
	body *yarpc.Buffer,
	protoReq proto.Message,
) (context.Context, *yarpc.InboundCall, proto.Message, error) {
	if err := yarpcencoding.ExpectEncodings(req, Encoding, yarpcjson.Encoding); err != nil {
		return nil, nil, nil, err
	}
	ctx, call := yarpc.NewInboundCall(ctx)
	if err := call.ReadFromRequest(req); err != nil {
		return nil, nil, nil, err
	}
	if err := unmarshal(req.Encoding, body, protoReq); err != nil {
		return nil, nil, nil, yarpcencoding.RequestBodyDecodeError(req, err)
	}
	return ctx, call, protoReq, nil
}
