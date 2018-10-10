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

package yarpcthrift

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/thrifttest"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/internal/testtime"
	yarpc "go.uber.org/yarpc/v2"
)

func TestDecodeRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
	proto.EXPECT().DecodeRequest(wire.Call, gomock.Any()).Return(
		wire.NewValueStruct(wire.Struct{}), protocol.NoEnvelopeResponder, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, w wire.Value) (Response, error) {
		return Response{Body: fakeEnveloper(wire.Reply)}, nil
	}
	h := UnaryTransportHandler{Protocol: proto, ThriftHandler: handler}

	res, _, err := h.Handle(ctx, request(), requestBody())

	assert.NoError(t, err, "unexpected error")
	assert.False(t, res.ApplicationError, "application error bit set")
}

func TestDecodeEnveloped(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	// XXX DecodeEnveloped instead of DecodeRequest or Decode(TStruct, ...)
	proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(wire.Envelope{
		Name:  "someMethod",
		SeqID: 42,
		Type:  wire.Call,
		Value: wire.NewValueStruct(wire.Struct{}),
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, w wire.Value) (Response, error) {
		assert.True(t, wire.ValuesAreEqual(wire.NewValueStruct(wire.Struct{}), w), "request body did not match")
		return Response{Body: fakeEnveloper(wire.Reply)}, nil
	}
	// XXX Enveloping true for this case, to induce DecodeEnveloped instead of
	// DecodeRequest or Decode.
	h := UnaryTransportHandler{Protocol: proto, ThriftHandler: handler, Enveloping: true}
	res, resBody, err := unaryCall(ctx, h)

	assert.NotNil(t, res)
	assert.NotNil(t, resBody)
	assert.NoError(t, err, "unexpected error")
}

func TestDecodeRequestApplicationError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
	proto.EXPECT().DecodeRequest(wire.Call, gomock.Any()).Return(
		wire.NewValueStruct(wire.Struct{}), protocol.NoEnvelopeResponder, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, w wire.Value) (Response, error) {
		// XXX setting application error bit
		return Response{Body: fakeEnveloper(wire.Reply), IsApplicationError: true}, nil
	}
	h := UnaryTransportHandler{Protocol: proto, ThriftHandler: handler}

	// XXX checking error bit
	res, resBody, err := h.Handle(ctx, request(), requestBody())
	assert.NotNil(t, res)
	assert.NotNil(t, resBody)
	assert.True(t, res.ApplicationError, "application error bit unset")
	assert.NoError(t, err, "unexpected error")
}

func TestDecodeRequestEncodingError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	// XXX handler and protocol superfluous for this case
	h := UnaryTransportHandler{}
	req := request()
	// XXX bogus encoding override
	req.Encoding = yarpc.Encoding("bogus")
	res, resBody, err := h.Handle(ctx, req, requestBody())

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), `expected encoding "thrift" but got "bogus"`)
	}
}

func TestDecodeRequestError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
	// XXX DecodeRequest returns decode request error
	proto.EXPECT().DecodeRequest(wire.Call, gomock.Any()).Return(
		wire.Value{}, protocol.NoEnvelopeResponder, fmt.Errorf("decode request error"))

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	h := UnaryTransportHandler{Protocol: proto}
	res, resBody, err := unaryCall(ctx, h)

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "decode request error")
	}
}

func TestDecodeRequestResponseError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
	// XXX threads a fake error responder, which returns an error on
	// EncodeResponse in thriftUnaryHandler.Handle.
	proto.EXPECT().DecodeRequest(wire.Call, gomock.Any()).Return(
		wire.Value{}, errorResponder{fmt.Errorf("encode response error")}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, w wire.Value) (Response, error) {
		return Response{Body: fakeEnveloper(wire.Reply)}, nil
	}
	h := UnaryTransportHandler{Protocol: proto, ThriftHandler: handler}
	res, resBody, err := unaryCall(ctx, h)

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "encode response error")
	}
}

func TestDecodeEnvelopedError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	// XXX DecodeEnveloped returns error
	proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(wire.Envelope{}, fmt.Errorf("decode enveloped error"))

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	h := UnaryTransportHandler{Protocol: proto, Enveloping: true}
	res, resBody, err := unaryCall(ctx, h)

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "decode enveloped error")
	}
}

func TestDecodeEnvelopedEnvelopeTypeError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	// XXX DecodeEnveloped returns OneWay instead of expected Call
	proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(wire.Envelope{Type: wire.OneWay}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	h := UnaryTransportHandler{Protocol: proto, Enveloping: true}
	res, resBody, err := unaryCall(ctx, h)

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "unexpected envelope type: OneWay")
	}
}

func TestDecodeNotEnvelopedError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	// XXX Mocked decode returns decode error
	proto.EXPECT().Decode(gomock.Any(), wire.TStruct).Return(wire.Value{}, fmt.Errorf("decode error"))

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	h := UnaryTransportHandler{Protocol: proto}
	res, resBody, err := unaryCall(ctx, h)

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "decode error")
	}
}

func TestUnaryHandlerError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Decode(gomock.Any(), wire.TStruct).Return(wire.Value{}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, w wire.Value) (Response, error) {
		// XXX returns error
		return Response{}, fmt.Errorf("unary handler error")
	}
	h := UnaryTransportHandler{Protocol: proto, ThriftHandler: handler}
	res, resBody, err := unaryCall(ctx, h)

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "unary handler error")
	}
}

func TestUnaryHandlerResponseEnvelopeTypeError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Decode(gomock.Any(), wire.TStruct).Return(wire.Value{}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, w wire.Value) (Response, error) {
		// XXX OneWay instead of Reply
		return Response{Body: fakeEnveloper(wire.OneWay)}, nil
	}
	h := UnaryTransportHandler{Protocol: proto, ThriftHandler: handler}
	res, resBody, err := unaryCall(ctx, h)

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "unexpected envelope type: OneWay")
	}
}

func TestUnaryHandlerBodyToWireError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Decode(gomock.Any(), wire.TStruct).Return(wire.Value{}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, w wire.Value) (Response, error) {
		// XXX Body.ToWire returns error
		return Response{Body: errorEnveloper{wire.Reply, fmt.Errorf("to wire error")}}, nil
	}
	h := UnaryTransportHandler{Protocol: proto, ThriftHandler: handler}
	res, resBody, err := unaryCall(ctx, h)

	assert.Nil(t, res)
	assert.Nil(t, resBody)
	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "to wire error")
	}
}

func request() *yarpc.Request {
	return &yarpc.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  "thrift",
		Procedure: "MyService::someMethod",
	}
}

func requestBody() *yarpc.Buffer {
	return yarpc.NewBufferBytes([]byte("irrelevant"))
}

func unaryCall(ctx context.Context, h UnaryTransportHandler) (*yarpc.Response, *yarpc.Buffer, error) {
	return h.Handle(ctx, request(), requestBody())
}
