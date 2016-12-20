// Copyright (c) 2016 Uber Technologies, Inc.
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
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf/internal"
	"go.uber.org/yarpc/internal/encoding"
	"go.uber.org/yarpc/internal/meta"
)

// UnaryHandler represents a protobuf request handler.
//
// Users should use the server code generated rather than using this directly.
type UnaryHandler interface {
	// response message, application error, metadata, yarpc error
	Handle(ctx context.Context, reqMeta yarpc.ReqMeta, reqMessage proto.Message) (proto.Message, error, yarpc.ResMeta, error)
	NewRequest() proto.Message
}

// NewUnaryHandler returns a new UnaryHandler.
func NewUnaryHandler(
	handle func(context.Context, yarpc.ReqMeta, proto.Message) (proto.Message, error, yarpc.ResMeta, error),
	newRequest func() proto.Message,
) UnaryHandler {
	return newUnaryHandler(handle, newRequest)
}

type unaryHandler struct {
	handle     func(context.Context, yarpc.ReqMeta, proto.Message) (proto.Message, error, yarpc.ResMeta, error)
	newRequest func() proto.Message
}

func newUnaryHandler(
	handle func(context.Context, yarpc.ReqMeta, proto.Message) (proto.Message, error, yarpc.ResMeta, error),
	newRequest func() proto.Message,
) UnaryHandler {
	return &unaryHandler{handle, newRequest}
}

func (u *unaryHandler) Handle(ctx context.Context, reqMeta yarpc.ReqMeta, reqMessage proto.Message) (proto.Message, error, yarpc.ResMeta, error) {
	return u.handle(ctx, reqMeta, reqMessage)
}

func (u *unaryHandler) NewRequest() proto.Message {
	return u.newRequest()
}

type transportUnaryHandler struct {
	unaryHandler UnaryHandler
}

func newTransportUnaryHandler(unaryHandler UnaryHandler) *transportUnaryHandler {
	return &transportUnaryHandler{unaryHandler}
}

func (t *transportUnaryHandler) Handle(ctx context.Context, transportRequest *transport.Request, responseWriter transport.ResponseWriter) error {
	if err := encoding.Expect(transportRequest, Encoding); err != nil {
		return err
	}
	body, err := ioutil.ReadAll(transportRequest.Body)
	if err != nil {
		return err
	}
	request := t.unaryHandler.NewRequest()
	// is this possible?
	if body != nil {
		if err := proto.Unmarshal(body, request); err != nil {
			return encoding.RequestBodyDecodeError(transportRequest, err)
		}
	}
	response, appErr, resMeta, err := t.unaryHandler.Handle(ctx, meta.FromTransportRequest(transportRequest), request)
	if err != nil {
		return err
	}
	if appErr != nil {
		responseWriter.SetApplicationError()
	}
	if resMeta != nil {
		meta.ToTransportResponseWriter(resMeta, responseWriter)
	}
	var resData []byte
	if response != nil {
		resData, err = protoMarshal(response)
		if err != nil {
			return encoding.ResponseBodyEncodeError(transportRequest, err)
		}
	}
	var internalApplicationError *internal.ApplicationError
	if appErr != nil {
		if applicationError, ok := appErr.(*ApplicationError); ok {
			payload, err := protoMarshal(applicationError.message)
			if err != nil {
				return encoding.ResponseBodyEncodeError(transportRequest, err)
			}
			internalApplicationError = &internal.ApplicationError{
				proto.MessageName(applicationError.message),
				payload,
			}
		} else {
			internalApplicationError = &internal.ApplicationError{
				Payload: []byte(appErr.Error()),
			}
		}
	}
	internalResponse := &internal.Response{
		resData,
		internalApplicationError,
	}
	internalResData, err := protoMarshal(internalResponse)
	if err != nil {
		return encoding.ResponseBodyEncodeError(transportRequest, err)
	}
	_, err = responseWriter.Write(internalResData)
	return err
}
