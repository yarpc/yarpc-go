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
	streamtest "go.uber.org/thriftrw/thrifttest/stream"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/testtime"
)

type responseEnveloper struct {
	name         string
	envelopeType wire.EnvelopeType
}

func (re responseEnveloper) MethodName() string              { return re.name }
func (re responseEnveloper) EnvelopeType() wire.EnvelopeType { return re.envelopeType }
func (re responseEnveloper) Encode(stream.Writer) error      { return nil }

type responseWriter struct{ err error }

func (rw responseWriter) WriteResponse(wire.EnvelopeType, io.Writer, stream.Enveloper) error {
	return rw.err
}

type responseHandler struct {
	nwc *NoWireCall
	err error

	body           stream.Enveloper
	responseWriter stream.ResponseWriter
}

func (rh *responseHandler) Handle(_ context.Context, nwc *NoWireCall) (NoWireResponse, error) {
	rh.nwc = nwc
	return NoWireResponse{
		Body:           rh.body,
		ResponseWriter: rh.responseWriter,
	}, rh.err
}

func TestNoWireHandleResponseEnvelopeError(t *testing.T) {
	re := responseEnveloper{name: "caller", envelopeType: wire.Exception}
	rh := responseHandler{body: re, responseWriter: responseWriter{}}
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
	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	rh := responseHandler{body: re, responseWriter: responseWriter{err: fmt.Errorf("write response error")}}
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
	assert.Contains(t, err.Error(), "write response error")
}

func TestNoWireDoRequestExpectEncodingsError(t *testing.T) {
	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	h := thriftNoWireHandler{
		NoWireHandler: &responseHandler{body: re, responseWriter: responseWriter{}},
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

func TestNoWireDoRequestRequestReaderUnary(t *testing.T) {
	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	rh := responseHandler{body: re, responseWriter: responseWriter{}}
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
	require.NotNil(t, rh.nwc)
	// In a call that uses the "RequestReader", the Reader, RequestReader, and
	// EnvelopeType must be set.
	require.NotNil(t, rh.nwc.Reader)
	require.NotNil(t, rh.nwc.RequestReader)
	require.NotNil(t, rh.nwc.EnvelopeType)
	assert.Equal(t, req.Body, rh.nwc.Reader)
	assert.Equal(t, proto, rh.nwc.RequestReader)
	assert.Equal(t, wire.Call, rh.nwc.EnvelopeType) // Unary call
}

func TestNoWireDoRequestRequestReaderOneway(t *testing.T) {
	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	rh := responseHandler{body: re, responseWriter: responseWriter{}}
	proto := binary.Default
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req := request()
	require.NoError(t, h.HandleOneway(ctx, req))
	require.NotNil(t, rh.nwc)
	// In a call that uses the "RequestReader", the Reader, RequestReader, and
	// EnvelopeType must be set.
	require.NotNil(t, rh.nwc.Reader)
	require.NotNil(t, rh.nwc.RequestReader)
	require.NotNil(t, rh.nwc.EnvelopeType)
	assert.Equal(t, req.Body, rh.nwc.Reader)
	assert.Equal(t, proto, rh.nwc.RequestReader)
	assert.Equal(t, wire.OneWay, rh.nwc.EnvelopeType) // OneWay call
}

func TestNoWireDoRequestRequestReaderHandleError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	h := thriftNoWireHandler{
		NoWireHandler: &responseHandler{body: re, responseWriter: responseWriter{}, err: fmt.Errorf("request reader error")},
		Protocol:      binary.Default,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, request(), rw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request reader error")
}

func TestNoWireDoRequestEnvelopingSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Return(stream.EnvelopeHeader{Type: wire.Call}, nil)
	sr.EXPECT().ReadEnvelopeEnd().Return(nil)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	rh := responseHandler{body: re}
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
		Enveloping:    true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	rw := new(transporttest.FakeResponseWriter)
	assert.NoError(t, h.Handle(ctx, request(), rw))
	assert.NotNil(t, rh.nwc.StreamReader)
}

func TestNoWireDoRequestEnvelopingBadEnvelope(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Return(stream.EnvelopeHeader{}, fmt.Errorf("read envelope begin error"))
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	h := thriftNoWireHandler{
		NoWireHandler: &responseHandler{},
		Protocol:      proto,
		Enveloping:    true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, request(), rw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read envelope begin error")
}

func TestNoWireDoRequestEnvelopingBadEnvelopeType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().AnyTimes().Return(stream.EnvelopeHeader{Type: wire.Exception}, nil)
	sr.EXPECT().Close().AnyTimes().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).AnyTimes().Return(sr)

	h := thriftNoWireHandler{
		NoWireHandler: &responseHandler{},
		Protocol:      proto,
		Enveloping:    true,
	}

	t.Run("unary", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		rw := new(transporttest.FakeResponseWriter)
		err := h.Handle(ctx, request(), rw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected envelope type")
	})

	t.Run("oneway", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		err := h.HandleOneway(ctx, request())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected envelope type")
	})
}

func TestNoWireHandleError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Return(stream.EnvelopeHeader{Type: wire.Call}, nil)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	h := thriftNoWireHandler{
		NoWireHandler: &responseHandler{body: re, responseWriter: responseWriter{}, err: fmt.Errorf("bad unary handle")},
		Protocol:      proto,
		Enveloping:    true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, request(), rw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad unary handle")
}

func TestNoWireHandleOnewayError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().AnyTimes().Return(stream.EnvelopeHeader{Type: wire.OneWay}, nil)
	sr.EXPECT().Close().AnyTimes().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).AnyTimes().Return(sr)

	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	h := thriftNoWireHandler{
		NoWireHandler: &responseHandler{body: re, responseWriter: responseWriter{}, err: fmt.Errorf("bad oneway handle")},
		Protocol:      proto,
		Enveloping:    true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	err := h.HandleOneway(ctx, request())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad oneway handle")
}

func TestNoWireDoRequestEnvelopingFalseSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Times(0)
	sr.EXPECT().ReadEnvelopeEnd().Times(0)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	rh := responseHandler{body: re}
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
		Enveloping:    false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	rw := new(transporttest.FakeResponseWriter)
	assert.NoError(t, h.Handle(ctx, request(), rw))
	assert.NotNil(t, rh.nwc)
}

func TestNoWireDoRequestEnvelopingFalseReadHandleError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sr := streamtest.NewMockReader(mockCtrl)
	sr.EXPECT().ReadEnvelopeBegin().Times(0)
	sr.EXPECT().ReadEnvelopeEnd().Times(0)
	sr.EXPECT().Close().Return(nil)

	proto := streamtest.NewMockProtocol(mockCtrl)
	proto.EXPECT().Reader(gomock.Any()).Return(sr)

	re := responseEnveloper{name: "caller", envelopeType: wire.Reply}
	rh := responseHandler{body: re, err: fmt.Errorf("bad handle")}
	h := thriftNoWireHandler{
		NoWireHandler: &rh,
		Protocol:      proto,
		Enveloping:    false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	rw := new(transporttest.FakeResponseWriter)
	err := h.Handle(ctx, request(), rw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad handle")
}
