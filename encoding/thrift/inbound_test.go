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

package thrift

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/thrifttest"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpcerrors"
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
	h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler}

	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, request(), rw)

	assert.NoError(t, err, "unexpected error")
	assert.False(t, rw.IsApplicationError, "application error bit set")
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
	h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler, Enveloping: true}
	err := unaryCall(ctx, h)

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

	codeNotFound := yarpcerrors.CodeNotFound

	handler := func(ctx context.Context, w wire.Value) (Response, error) {
		// XXX setting application error bit
		return Response{
			Body:                 fakeEnveloper(wire.Reply),
			IsApplicationError:   true,
			ApplicationErrorName: "thrift-defined-error",
			ApplicationErrorCode: &codeNotFound,
		}, nil
	}
	h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler}

	// XXX checking error bit
	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, request(), rw)
	assert.True(t, rw.IsApplicationError, "application error bit unset")
	assert.Equal(t, "thrift-defined-error", rw.ApplicationErrorMeta.Name,
		"application error name mismatch")
	assert.Equal(t, &codeNotFound, rw.ApplicationErrorMeta.Code,
		"application error code mismatch")

	assert.NoError(t, err, "unexpected error")
}

func TestDecodeRequestEncodingError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	// XXX handler and protocol superfluous for this case
	h := thriftUnaryHandler{}
	rw := new(transporttest.FakeResponseWriter)
	req := request()
	// XXX bogus encoding override
	req.Encoding = transport.Encoding("bogus")
	err := h.Handle(ctx, req, rw)

	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), `expected encoding "thrift" but got "bogus"`)
	}
}

func TestDecodeRequestReadError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	// XXX handler and protocol superfluous for this case
	h := thriftUnaryHandler{}
	rw := new(transporttest.FakeResponseWriter)
	req := request()
	// XXX override body with a bad reader, returns error on read
	req.Body = &badReader{}
	err := h.Handle(ctx, req, rw)

	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "bad reader error")
	}
}

type badReader struct{}

func (badReader) Read(buf []byte) (int, error) {
	return 0, fmt.Errorf("bad reader error")
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

	h := thriftUnaryHandler{Protocol: proto}
	err := unaryCall(ctx, h)

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
	h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler}
	err := unaryCall(ctx, h)

	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "encode response error")
	}
}

type closeWrapper struct {
	io.Reader
	closeErr error
}

func (c closeWrapper) Close() error {
	return c.closeErr
}

func TestDecodeRequestClose(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
		proto.EXPECT().DecodeRequest(wire.Call, gomock.Any()).Return(
			wire.Value{}, protocol.NoEnvelopeResponder, nil)

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		handler := func(ctx context.Context, w wire.Value) (Response, error) {
			return Response{Body: fakeEnveloper(wire.Reply)}, nil
		}
		h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler}
		req := request()

		// Add close method to the body.
		req.Body = closeWrapper{req.Body, nil /* close error */}
		err := h.Handle(ctx, req, new(transporttest.FakeResponseWriter))
		require.NoError(t, err)
	})

	t.Run("close error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		// Proto and Handler won't get used because of the close error.
		h := thriftUnaryHandler{}
		req := request()

		// Add close method to the body that returns an error.
		req.Body = closeWrapper{req.Body, errors.New("close failed")}
		err := h.Handle(ctx, req, new(transporttest.FakeResponseWriter))
		require.Error(t, err)
	})
}

func TestDecodeEnvelopedError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	// XXX DecodeEnveloped returns error
	proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(wire.Envelope{}, fmt.Errorf("decode enveloped error"))

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	h := thriftUnaryHandler{Protocol: proto, Enveloping: true}
	err := unaryCall(ctx, h)

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

	h := thriftUnaryHandler{Protocol: proto, Enveloping: true}
	err := unaryCall(ctx, h)

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

	h := thriftUnaryHandler{Protocol: proto}
	err := unaryCall(ctx, h)

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
	h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler}
	err := unaryCall(ctx, h)

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
	h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler}
	err := unaryCall(ctx, h)

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
	h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler}
	err := unaryCall(ctx, h)

	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "to wire error")
	}
}

func TestOnewayHandler(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
	// XXX expecting OneWay request instead of Call
	proto.EXPECT().DecodeRequest(wire.OneWay, gomock.Any()).Return(
		wire.NewValueStruct(wire.Struct{}), protocol.NoEnvelopeResponder, nil)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, v wire.Value) error {
		return nil
	}
	h := thriftOnewayHandler{Protocol: proto, OnewayHandler: handler}
	err := h.HandleOneway(ctx, request())

	// XXX expecting success in this case
	assert.NoError(t, err, "unexpected error")
}

func TestOnewayHandlerError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
	// XXX mock returns decode request error, to induce error path out of handleRequest in HandleOneway
	proto.EXPECT().DecodeRequest(wire.OneWay, gomock.Any()).Return(
		wire.Value{}, protocol.NoEnvelopeResponder, fmt.Errorf("decode request error"))

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	handler := func(ctx context.Context, v wire.Value) error {
		return nil
	}
	h := thriftOnewayHandler{Protocol: proto, OnewayHandler: handler}
	err := h.HandleOneway(ctx, request())

	if assert.Error(t, err, "expected an error") {
		assert.Contains(t, err.Error(), "decode request error")
	}
}

func request() *transport.Request {
	return &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  "thrift",
		Procedure: "MyService::someMethod",
		Body:      bytes.NewReader([]byte("irrelevant")),
	}
}

func unaryCall(ctx context.Context, h thriftUnaryHandler) error {
	rw := new(transporttest.FakeResponseWriter)
	return h.Handle(ctx, request(), rw)
}
