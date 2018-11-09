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
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/thrifttest"
	"go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc/v2"
)

func TestDecodeAgnosticProto(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
	proto.EXPECT().DecodeRequest(wire.Call, gomock.Any()).Return(
		wire.NewValueStruct(wire.Struct{}), protocol.NoEnvelopeResponder, nil).Times(2)

	testCodec := newCodec(proto, false)
	reqBuf := &yarpc.Buffer{}

	res, err := testCodec.Decode(reqBuf)
	assert.NoError(t, err, "unexpected error")
	resVal, ok := res.(wire.Value)
	assert.True(t, ok, "expected wire.Value")
	assert.Equal(t, wire.NewValueStruct(wire.Struct{}), resVal)

	// calling Decode again should fail, because it attempts to set a new
	// responder.
	res, err = testCodec.Decode(reqBuf)
	assert.EqualError(t, err, "code:internal message:tried to overwrite a responder for thrift codec")
}

func TestDecodeAgnosticProtoError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockEnvelopeAgnosticProtocol(mockCtrl)
	proto.EXPECT().DecodeRequest(wire.Call, gomock.Any()).Return(
		wire.Value{}, nil, errors.New("error decoding request"))

	testCodec := newCodec(proto, false)
	reqBuf := &yarpc.Buffer{}

	res, err := testCodec.Decode(reqBuf)
	assert.EqualError(t, err, "error decoding request")
	assert.Nil(t, res)
}

func TestDecodeEnveloped(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(
		wire.Envelope{Type: wire.Call}, nil).Times(2)

	testCodec := newCodec(proto, true)
	reqBuf := &yarpc.Buffer{}

	res, err := testCodec.Decode(reqBuf)
	assert.NoError(t, err, "unexpected error")
	resVal, ok := res.(wire.Value)
	assert.True(t, ok, "expected wire.Value")
	assert.Equal(t, wire.Envelope{Type: wire.Call}.Value, resVal)

	// calling Decode again should fail, because it attempts to set a new
	// responder.
	res, err = testCodec.Decode(reqBuf)
	assert.EqualError(t, err, "code:internal message:tried to overwrite a responder for thrift codec")
}

func TestDecodeEnvelopedError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(
		wire.Envelope{}, errors.New("error decoding request"))

	testCodec := newCodec(proto, true)
	reqBuf := &yarpc.Buffer{}

	res, err := testCodec.Decode(reqBuf)
	assert.EqualError(t, err, "error decoding request")
	assert.Nil(t, res)
}

func TestDecodeEnvelopedWrongTypeError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(
		wire.Envelope{Type: wire.OneWay}, nil)

	testCodec := newCodec(proto, true)
	reqBuf := &yarpc.Buffer{}

	res, err := testCodec.Decode(reqBuf)
	require.Error(t, err, "expected error")
	assert.Contains(t, err.Error(), "unexpected envelope type: OneWay")
	assert.Nil(t, res)
}

func TestDecodeUnenveloped(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Decode(gomock.Any(), wire.TStruct).Return(
		wire.NewValueStruct(wire.Struct{}), nil).Times(2)

	testCodec := newCodec(proto, false)
	reqBuf := &yarpc.Buffer{}

	res, err := testCodec.Decode(reqBuf)
	assert.NoError(t, err, "unexpected error")
	resVal, ok := res.(wire.Value)
	assert.True(t, ok, "expected wire.Value")
	assert.Equal(t, wire.NewValueStruct(wire.Struct{}), resVal)

	// calling Decode again should fail, because it attempts to set a new
	// responder.
	res, err = testCodec.Decode(reqBuf)
	assert.EqualError(t, err, "code:internal message:tried to overwrite a responder for thrift codec")
}

func TestDecodeUnenvelopedError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Decode(gomock.Any(), wire.TStruct).Return(
		wire.Value{}, errors.New("error decoding request"))

	testCodec := newCodec(proto, false)
	reqBuf := &yarpc.Buffer{}

	res, err := testCodec.Decode(reqBuf)
	assert.EqualError(t, err, "error decoding request")
	assert.Nil(t, res)
}

func TestEncode(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	responder := thrifttest.NewMockResponder(mockCtrl)
	responder.EXPECT().EncodeResponse(gomock.Any(), wire.Reply, gomock.Any()).Return(nil)

	testCodec := newCodec(protocol.Binary, false)
	testCodec.responder = responder

	resBuf, err := testCodec.Encode(wire.Value{})
	assert.NoError(t, err, "unexpected error")
	assert.NotNil(t, resBuf)
}

func TestEncodeError(t *testing.T) {
	testCodec := newCodec(protocol.Binary, false)

	// thrift codec only encodes wire.Value
	_, err := testCodec.Encode(wire.Envelope{})
	require.Error(t, err, "unexpected error")
	assert.Contains(t, err.Error(), "tried to encode a non-wire.Value in thrift codec")
}
