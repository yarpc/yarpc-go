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
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/thrifttest/streamtest"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/testtime"
)

const _body = "decoded"

type bodyReader struct {
	body string
	err  error
}

func (br *bodyReader) Decode(sr stream.Reader) error {
	br.body = _body
	return br.err
}

type responseEnveloper struct {
	name         string
	envelopeType wire.EnvelopeType
}

var _ stream.Enveloper = (*responseEnveloper)(nil)

func (re responseEnveloper) MethodName() string              { return re.name }
func (re responseEnveloper) EnvelopeType() wire.EnvelopeType { return re.envelopeType }
func (re responseEnveloper) Encode(stream.Writer) error      { return nil }

type responseWriter struct{ err error }

var _ stream.ResponseWriter = (*responseWriter)(nil)

func (rw responseWriter) WriteResponse(wire.EnvelopeType, io.Writer, stream.Enveloper) error {
	return rw.err
}

type responseHandler struct {
	t   *testing.T
	nwc *NoWireCall

	reqBody stream.BodyReader
	body    stream.Enveloper
}

var _ NoWireHandler = (*responseHandler)(nil)

func (rh *responseHandler) Handle(ctx context.Context, nwc *NoWireCall) (NoWireResponse, error) {
	rh.t.Helper()

	// All calls to Handle must have everything in a NoWireCall set.
	require.NotNil(rh.t, nwc)
	assert.NotNil(rh.t, nwc.Reader)
	assert.NotNil(rh.t, nwc.RequestReader)
	assert.NotNil(rh.t, nwc.EnvelopeType)

	rh.nwc = nwc
	rw, err := nwc.RequestReader.ReadRequest(ctx, nwc.EnvelopeType, nwc.Reader, rh.reqBody)
	return NoWireResponse{
		Body:           rh.body,
		ResponseWriter: rw,
	}, err
}

func TestDecodeNoWireRequestUnary(t *testing.T) {
	rh := responseHandler{
		t:       t,
		reqBody: &bodyReader{},
		body:    responseEnveloper{name: "caller", envelopeType: wire.Reply},
	}
	proto := binary.Default
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req := request()
	rw := new(transporttest.FakeResponseWriter)
	require.NoError(t, h.Handle(ctx, req, rw))
	assert.Equal(t, req.Body, rh.nwc.Reader)
	assert.Equal(t, proto, rh.nwc.RequestReader)
	assert.Equal(t, wire.Call, rh.nwc.EnvelopeType) // Unary call
}

func TestDecodeNoWireRequestOneway(t *testing.T) {
	rh := responseHandler{
		t:       t,
		reqBody: &bodyReader{},
		body:    responseEnveloper{name: "caller", envelopeType: wire.Reply},
	}
	proto := binary.Default
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req := request()
	require.NoError(t, h.HandleOneway(ctx, req))
	assert.Equal(t, req.Body, rh.nwc.Reader)
	assert.Equal(t, proto, rh.nwc.RequestReader)
	assert.Equal(t, wire.OneWay, rh.nwc.EnvelopeType) // OneWay call
}

func TestDecodeNoWireRequestEnveloping(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Return(stream.EnvelopeHeader{Type: wire.Call}, nil)
	sr.EXPECT().ReadEnvelopeEnd().Return(nil)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	br := &bodyReader{}
	rh := responseHandler{
		t:       t,
		reqBody: br,
		body:    responseEnveloper{name: "caller", envelopeType: wire.Reply},
	}
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
		Enveloping:    true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	rw := new(transporttest.FakeResponseWriter)
	require.NoError(t, h.Handle(ctx, request(), rw))
	assert.Equal(t, _body, br.body, "request body expected to be decoded")

	rrp, ok := rh.nwc.RequestReader.(*reqReaderProto)
	require.True(t, ok)
	assert.Equal(t, proto, rrp.Protocol)
}

func TestDecodeNoWireRequestEnvelopingFalse(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	br := &bodyReader{}
	rh := responseHandler{
		t:       t,
		reqBody: br,
		body:    responseEnveloper{name: "caller", envelopeType: wire.Reply},
	}
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
		Enveloping:    false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	rw := new(transporttest.FakeResponseWriter)
	require.NoError(t, h.Handle(ctx, request(), rw))
	assert.Equal(t, _body, br.body, "request body expected to be decoded")

	rrp, ok := rh.nwc.RequestReader.(*reqReaderProto)
	require.True(t, ok)
	assert.Equal(t, proto, rrp.Protocol)
}

func TestNoWireHandleIncorrectResponseEnvelope(t *testing.T) {
	br := &bodyReader{}
	rh := responseHandler{
		t:       t,
		reqBody: br,
		body:    responseEnveloper{name: "caller", envelopeType: wire.Exception},
	}
	proto := binary.Default
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req := request()
	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, req, rw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected envelope type")
}

func TestNoWireHandleWriteResponseError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := streamtest.NewMockRequestReader(mockCtrl)
	proto.EXPECT().ReadRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(responseWriter{err: fmt.Errorf("write response error")}, nil)

	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	br := &bodyReader{}
	rh := responseHandler{t: t, reqBody: br, body: re}
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req := request()
	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, req, rw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write response error")
}

func TestDecodeNoWireRequestExpectEncodingsError(t *testing.T) {
	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	h := thriftNoWireHandler{
		NoWireHandler: &responseHandler{t: t, body: re},
		Protocol:      binary.Default,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req := request()
	req.Encoding = "grpc"

	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, req, rw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `expected encoding "thrift" but got "grpc"`)
}

func TestReqReaderEnvelopingEnvelopeBeginError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Return(stream.EnvelopeHeader{}, fmt.Errorf("read envelope begin error"))
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	_, err := testEnvelopedReadRequest(t, proto, &bodyReader{}, true /* enveloping */)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read envelope begin error")
}

func TestReqReaderEnvelopingBadEnvelopeType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Return(stream.EnvelopeHeader{Type: wire.Exception}, nil)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	_, err := testEnvelopedReadRequest(t, proto, &bodyReader{}, true /* enveloping */)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected envelope type")
}

func TestReqReaderEnvelopingDecodeError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Return(stream.EnvelopeHeader{Type: wire.OneWay}, nil)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	_, err := testEnvelopedReadRequest(t, proto, &bodyReader{err: fmt.Errorf("decode error")}, true /* enveloping */)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode error")
}

func TestReqReaderEnvelopingEnvelopeEndError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Return(stream.EnvelopeHeader{Type: wire.OneWay}, nil)
	sr.EXPECT().ReadEnvelopeEnd().Return(fmt.Errorf("read envelope end error"))
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	_, err := testEnvelopedReadRequest(t, proto, &bodyReader{}, true /* enveloping */)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read envelope end error")
}

func TestReqReaderNotEnvelopingDecodeError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	_, err := testEnvelopedReadRequest(t, proto, &bodyReader{err: fmt.Errorf("another decode error")}, false /* enveloping */)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "another decode error")
}

func testEnvelopedReadRequest(t *testing.T, proto stream.Protocol, body stream.BodyReader, enveloping bool) (stream.ResponseWriter, error) {
	t.Helper()

	req := request()
	rrp := reqReaderProto{
		Protocol:   proto,
		treq:       req,
		enveloping: enveloping,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	return rrp.ReadRequest(ctx, wire.OneWay, req.Body, body)
}
