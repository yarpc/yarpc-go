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
	"go.uber.org/yarpc/v2/yarpcerror"
)

var _ yarpc.UnaryEncodingHandler = (*unaryHandler)(nil)
var _ yarpc.StreamTransportHandler = (*streamHandler)(nil)

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
	Handle func(context.Context, proto.Message) (proto.Message, error)
}

type unaryHandler struct {
	handle func(context.Context, proto.Message) (proto.Message, error)
}

// NewUnaryHandler returns a new UnaryHandler.
func NewUnaryHandler(p UnaryHandlerParams) yarpc.UnaryEncodingHandler {
	return &unaryHandler{handle: p.Handle}
}

// Handle handles a proto.Message and returns a proto.Message
func (u *unaryHandler) Handle(ctx context.Context, reqBody interface{}) (interface{}, error) {
	if reqMessage, ok := reqBody.(proto.Message); ok {
		return u.handle(ctx, reqMessage)
	}
	return nil, yarpcerror.InternalErrorf("tried to handle a non-proto.Message in protobuf handler")
}
