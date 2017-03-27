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

	apiencoding "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/x/protobuf/internal/wirepb"
	"go.uber.org/yarpc/internal/buffer"
	"go.uber.org/yarpc/internal/encoding"

	"github.com/gogo/protobuf/proto"
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
	if err := encoding.Expect(transportRequest, Encoding); err != nil {
		return err
	}
	ctx, call := apiencoding.NewInboundCall(ctx)
	if err := call.ReadFromRequest(transportRequest); err != nil {
		return err
	}
	buf := buffer.Get()
	defer buffer.Put(buf)
	if _, err := buf.ReadFrom(transportRequest.Body); err != nil {
		return err
	}
	body := buf.Bytes()
	request := u.newRequest()
	// is this possible?
	if body != nil {
		if err := proto.Unmarshal(body, request); err != nil {
			return encoding.RequestBodyDecodeError(transportRequest, err)
		}
	}
	response, appErr := u.handle(ctx, request)
	if err := call.WriteToResponse(responseWriter); err != nil {
		return err
	}
	var responseData []byte
	if response != nil {
		protoBuffer := getBuffer()
		defer putBuffer(protoBuffer)
		if err := protoBuffer.Marshal(response); err != nil {
			return encoding.ResponseBodyEncodeError(transportRequest, err)
		}
		responseData = protoBuffer.Bytes()
	}
	var wireError *wirepb.Error
	if appErr != nil {
		responseWriter.SetApplicationError()
		wireError = &wirepb.Error{
			appErr.Error(),
		}
	}
	wireResponse := &wirepb.Response{
		responseData,
		wireError,
	}
	protoBuffer := getBuffer()
	defer putBuffer(protoBuffer)
	if err := protoBuffer.Marshal(wireResponse); err != nil {
		return encoding.ResponseBodyEncodeError(transportRequest, err)
	}
	_, err := responseWriter.Write(protoBuffer.Bytes())
	return err
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
	if err := encoding.Expect(transportRequest, Encoding); err != nil {
		return err
	}
	ctx, call := apiencoding.NewInboundCall(ctx)
	if err := call.ReadFromRequest(transportRequest); err != nil {
		return err
	}
	buf := buffer.Get()
	defer buffer.Put(buf)
	if _, err := buf.ReadFrom(transportRequest.Body); err != nil {
		return err
	}
	body := buf.Bytes()
	request := o.newRequest()
	// is this possible?
	if body != nil {
		if err := proto.Unmarshal(body, request); err != nil {
			return encoding.RequestBodyDecodeError(transportRequest, err)
		}
	}
	return o.handleOneway(ctx, request)
}
