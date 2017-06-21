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
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	apiencoding "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/x/protobuf/internal/wirepb"
	"go.uber.org/yarpc/internal/buffer"
	"go.uber.org/yarpc/internal/encoding"
)

var (
	_jsonMarshaler   = &jsonpb.Marshaler{}
	_jsonUnmarshaler = &jsonpb.Unmarshaler{AllowUnknownFields: true}
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
	if appErr != nil {
		responseWriter.SetApplicationError()
	}
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
			return encoding.ResponseBodyEncodeError(transportRequest, err)
		}
	}
	// We have to detect if our transport requires a raw response
	// It is not possible to propagate this information on ctx with the current API
	// we we attach this in the relevant transport (currently only gRPC) on the headers
	// If we are sending a raw response back to a YARPC client, it needs to understand
	// this is happening, so we attach the headers on the response as well
	// Other clients (namely the existing gRPC clients outside of YARPC) understand
	// that the response is the raw response.
	if isRawResponse(transportRequest.Headers) {
		responseWriter.AddHeaders(getRawResponseHeaders())
		_, err := responseWriter.Write(responseData)
		if err != nil {
			return err
		}
		return appErr
	}
	var wireError *wirepb.Error
	if appErr != nil {
		wireError = &wirepb.Error{
			Message: appErr.Error(),
		}
	}
	wireResponse := &wirepb.Response{
		Payload: string(responseData),
		Error:   wireError,
	}
	wireData, wireCleanup, err := marshal(transportRequest.Encoding, wireResponse)
	if wireCleanup != nil {
		defer wireCleanup()
	}
	if err != nil {
		return encoding.ResponseBodyEncodeError(transportRequest, err)
	}
	_, err = responseWriter.Write(wireData)
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
	ctx, _, request, err := getProtoRequest(ctx, transportRequest, o.newRequest)
	if err != nil {
		return err
	}
	return o.handleOneway(ctx, request)
}

func getProtoRequest(ctx context.Context, transportRequest *transport.Request, newRequest func() proto.Message) (context.Context, *apiencoding.InboundCall, proto.Message, error) {
	if err := encoding.Expect(transportRequest, Encoding, JSONEncoding); err != nil {
		return nil, nil, nil, err
	}
	ctx, call := apiencoding.NewInboundCall(ctx)
	if err := call.ReadFromRequest(transportRequest); err != nil {
		return nil, nil, nil, err
	}
	request := newRequest()
	if err := unmarshal(transportRequest.Encoding, transportRequest.Body, request); err != nil {
		return nil, nil, nil, encoding.RequestBodyDecodeError(transportRequest, err)
	}
	return ctx, call, request, nil
}

func unmarshal(encoding transport.Encoding, reader io.Reader, message proto.Message) error {
	buf := buffer.Get()
	defer buffer.Put(buf)
	if _, err := buf.ReadFrom(reader); err != nil {
		return err
	}
	body := buf.Bytes()
	if len(body) == 0 {
		return nil
	}
	switch encoding {
	case Encoding:
		return unmarshalProto(body, message)
	case JSONEncoding:
		return unmarshalJSON(body, message)
	default:
		return fmt.Errorf("encoding.Expect should have handled encoding %q but did not", encoding)
	}
}

func unmarshalProto(body []byte, message proto.Message) error {
	return proto.Unmarshal(body, message)
}

func unmarshalJSON(body []byte, message proto.Message) error {
	return _jsonUnmarshaler.Unmarshal(bytes.NewReader(body), message)
}

func marshal(encoding transport.Encoding, message proto.Message) ([]byte, func(), error) {
	switch encoding {
	case Encoding:
		return marshalProto(message)
	case JSONEncoding:
		return marshalJSON(message)
	default:
		return nil, nil, fmt.Errorf("encoding.Expect should have handled encoding %q but did not", encoding)
	}
}

func marshalProto(message proto.Message) ([]byte, func(), error) {
	protoBuffer := getBuffer()
	cleanup := func() { putBuffer(protoBuffer) }
	if err := protoBuffer.Marshal(message); err != nil {
		cleanup()
		return nil, nil, err
	}
	return protoBuffer.Bytes(), cleanup, nil
}

func marshalJSON(message proto.Message) ([]byte, func(), error) {
	buf := buffer.Get()
	cleanup := func() { buffer.Put(buf) }
	if err := _jsonMarshaler.Marshal(buf, message); err != nil {
		cleanup()
		return nil, nil, err
	}
	return buf.Bytes(), cleanup, nil
}
