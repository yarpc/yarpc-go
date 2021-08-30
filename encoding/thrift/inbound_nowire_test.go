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
	sr   stream.Reader
	body string
	err  error
}

func (br *bodyReader) Decode(sr stream.Reader) error {
	br.sr = sr
	br.body = _body
	return br.err
}

type responseHandler struct {
	t   *testing.T
	nwc *NoWireCall

	reqBody  stream.BodyReader
	body     stream.Enveloper
	appError bool
}

var _ NoWireHandler = (*responseHandler)(nil)

func (rh *responseHandler) HandleNoWire(ctx context.Context, nwc *NoWireCall) (NoWireResponse, error) {
	rh.t.Helper()

	// All calls to Handle must have everything in a NoWireCall set.
	require.NotNil(rh.t, nwc)
	assert.NotNil(rh.t, nwc.Reader)
	assert.NotNil(rh.t, nwc.RequestReader)
	assert.NotNil(rh.t, nwc.EnvelopeType)

	rh.nwc = nwc
	rw, err := nwc.RequestReader.ReadRequest(ctx, nwc.EnvelopeType, nwc.Reader, rh.reqBody)
	return NoWireResponse{
		Body:               rh.body,
		ResponseWriter:     rw,
		IsApplicationError: rh.appError,
	}, err
}

func TestDecodeNoWireRequestUnary(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	env := streamtest.NewMockEnveloper(mockCtrl)
	env.EXPECT().EnvelopeType().Return(wire.Reply).Times(1)
	env.EXPECT().Encode(gomock.Any()).Return(nil).Times(1)

	rh := responseHandler{
		t:       t,
		reqBody: &bodyReader{},
		body:    env,
	}
	proto := binary.Default
	h := thriftNoWireHandler{
		Handler:       &rh,
		RequestReader: proto,
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// OneWay calls have no calls to the response body
	env := streamtest.NewMockEnveloper(mockCtrl)
	rh := responseHandler{
		t:       t,
		reqBody: &bodyReader{},
		body:    env,
	}
	proto := binary.Default
	h := thriftNoWireHandler{
		Handler:       &rh,
		RequestReader: proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req := request()
	require.NoError(t, h.HandleOneway(ctx, req))
	assert.Equal(t, req.Body, rh.nwc.Reader)
	assert.Equal(t, proto, rh.nwc.RequestReader)
	assert.Equal(t, wire.OneWay, rh.nwc.EnvelopeType) // OneWay call
}

func TestNoWireHandleIncorrectResponseEnvelope(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	env := streamtest.NewMockEnveloper(mockCtrl)
	env.EXPECT().EnvelopeType().Return(wire.Exception).Times(1)

	br := &bodyReader{}
	rh := responseHandler{
		t:       t,
		reqBody: br,
		body:    env,
	}
	proto := binary.Default
	h := thriftNoWireHandler{
		Handler:       &rh,
		RequestReader: proto,
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

	env := streamtest.NewMockEnveloper(mockCtrl)
	env.EXPECT().EnvelopeType().Return(wire.Reply).Times(1)

	rw := new(transporttest.FakeResponseWriter)
	streamRw := streamtest.NewMockResponseWriter(mockCtrl)
	streamRw.EXPECT().WriteResponse(wire.Reply, rw, env).Return(fmt.Errorf("write response error")).Times(1)

	req := request()
	br := &bodyReader{}
	proto := streamtest.NewMockRequestReader(mockCtrl)
	proto.EXPECT().ReadRequest(gomock.Any(), wire.Call, req.Body, br).
		Return(streamRw, nil)

	rh := responseHandler{t: t, reqBody: br, body: env}
	h := thriftNoWireHandler{
		Handler:       &rh,
		RequestReader: proto,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	err := h.Handle(ctx, req, rw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write response error")
}

func TestDecodeNoWireRequestExpectEncodingsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// incorrect encoding in response should result in no calls to the response body
	env := streamtest.NewMockEnveloper(mockCtrl)
	h := thriftNoWireHandler{
		Handler:       &responseHandler{t: t, body: env},
		RequestReader: binary.Default,
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

func TestDecodeNoWireAppliationError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	env := streamtest.NewMockEnveloper(mockCtrl)
	env.EXPECT().EnvelopeType().Return(wire.Reply).Times(1)
	env.EXPECT().Encode(gomock.Any()).Return(nil).Times(1)

	br := &bodyReader{}
	h := thriftNoWireHandler{
		Handler: &responseHandler{
			t:        t,
			reqBody:  br,
			body:     env,
			appError: true,
		},
		RequestReader: binary.Default,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req := request()
	rw := new(transporttest.FakeResponseWriter)
	require.NoError(t, h.Handle(ctx, req, rw))
	assert.True(t, rw.IsApplicationError)
}
