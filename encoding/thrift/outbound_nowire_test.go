// Copyright (c) 2025 Uber Technologies, Inc.
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
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol"
	tbinary "go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/thrifttest"
	"go.uber.org/thriftrw/thrifttest/streamtest"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/testtime"
)

const _response = "response"

func TestNoWireClientCall(t *testing.T) {
	tests := []struct {
		desc             string
		giveRequestBody  stream.Enveloper
		giveResponseBody string
		clientOptions    []ClientOption

		expectCall       bool
		wantRequestBody  string
		wantResponseBody string
		wantError        string
	}{
		{
			desc:             "positive case, without enveloping",
			giveRequestBody:  fakeEnveloper(wire.Call),
			giveResponseBody: encodeThriftString(t, _response),
			expectCall:       true,
			wantRequestBody:  encodeThriftString(t, _irrelevant),
			wantResponseBody: _response,
		},
		{
			desc:            "positive case, with enveloping",
			giveRequestBody: fakeEnveloper(wire.Call),
			giveResponseBody: encodeEnvelopeType(t, wire.Reply) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _response),
			clientOptions: []ClientOption{Enveloped},
			expectCall:    true,
			wantRequestBody: encodeEnvelopeType(t, wire.Call) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _irrelevant),
			wantResponseBody: _response,
		},
		{
			desc:            "unexpected request envelope type",
			giveRequestBody: fakeEnveloper(wire.Exception),
			wantError:       `failed to encode "thrift" request body for procedure "MyService::someMethod" of service "service": unexpected envelope type: Exception`,
		},
		{
			desc:            "response envelope exception (TApplicationException) decoding error",
			giveRequestBody: fakeEnveloper(wire.Call),
			clientOptions:   []ClientOption{Enveloped},
			giveResponseBody: encodeEnvelopeType(t, wire.Exception) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1),
			expectCall: true,
			wantRequestBody: encodeEnvelopeType(t, wire.Call) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _irrelevant),
			wantResponseBody: _response,
			wantError:        `failed to decode "thrift" response body for procedure "MyService::someMethod" of service "service": unexpected EOF`,
		},
		{
			desc:            "response envelope exception (TApplicationException) error",
			giveRequestBody: fakeEnveloper(wire.Call),
			clientOptions:   []ClientOption{Enveloped},
			giveResponseBody: encodeEnvelopeType(t, wire.Exception) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _response),
			expectCall: true,
			wantRequestBody: encodeEnvelopeType(t, wire.Call) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _irrelevant),
			wantResponseBody: _response,
			wantError:        "encountered an internal failure: TApplicationException{}",
		},
		{
			desc:            "unexpected response envelope type",
			giveRequestBody: fakeEnveloper(wire.Call),
			giveResponseBody: encodeEnvelopeType(t, wire.Call) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _response),
			clientOptions: []ClientOption{Enveloped},
			expectCall:    true,
			wantRequestBody: encodeEnvelopeType(t, wire.Call) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _irrelevant),
			wantResponseBody: _response,
			wantError:        `failed to decode "thrift" response body for procedure "MyService::someMethod" of service "service": unexpected envelope type: Call`,
		},
	}

	// This type aliasing is needed because it's not possible to embed two types
	// with the same name without collision.
	type streamProtocol = stream.Protocol
	type fakeProtocol struct {
		protocol.Protocol
		streamProtocol
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			sp := streamtest.NewMockProtocol(mockCtrl)
			if tt.wantRequestBody != "" {
				sp.EXPECT().Writer(gomock.Any()).
					DoAndReturn(func(w io.Writer) stream.Writer {
						return tbinary.Default.Writer(w)
					})
			}

			if tt.wantResponseBody != "" {
				sp.EXPECT().Reader(gomock.Any()).
					DoAndReturn(func(r io.Reader) stream.Reader {
						return tbinary.Default.Reader(r)
					})
			}

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			trans := transporttest.NewMockUnaryOutbound(mockCtrl)
			if tt.expectCall {
				trans.EXPECT().Call(gomock.Any(),
					transporttest.NewRequestMatcher(t, &transport.Request{
						Caller:    "caller",
						Service:   "service",
						Encoding:  Encoding,
						Procedure: "MyService::someMethod",
						Body:      strings.NewReader(tt.wantRequestBody),
					}),
				).Return(&transport.Response{
					Body: io.NopCloser(strings.NewReader(tt.giveResponseBody)),
				}, nil)
			}

			opts := tt.clientOptions
			opts = append(opts, Protocol(&fakeProtocol{streamProtocol: sp}))
			nwc := NewNoWire(Config{
				Service: "MyService",
				ClientConfig: clientconfig.MultiOutbound("caller", "service",
					transport.Outbounds{
						Unary: trans,
					}),
			}, opts...)

			br := fakeBodyReader{}
			err := nwc.Call(ctx, tt.giveRequestBody, &br)
			if tt.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantResponseBody, br.body)
		})
	}
}

func TestNoWireClientOneway(t *testing.T) {
	tests := []struct {
		msg             string
		giveRequestBody stream.Enveloper
		clientOptions   []ClientOption

		expectCall      bool
		wantRequestBody string
		wantError       string
	}{
		{
			msg:             "positive case, without enveloping",
			giveRequestBody: fakeEnveloper(wire.OneWay),
			expectCall:      true,
			wantRequestBody: encodeThriftString(t, _irrelevant),
		},
		{
			msg:             "positive case, with enveloping",
			giveRequestBody: fakeEnveloper(wire.OneWay),
			clientOptions:   []ClientOption{Enveloped},
			expectCall:      true,
			wantRequestBody: encodeEnvelopeType(t, wire.OneWay) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _irrelevant),
		},
		{
			msg:             "unexpected request envelope type",
			giveRequestBody: fakeEnveloper(wire.Exception),
			wantError:       `failed to encode "thrift" request body for procedure "MyService::someMethod" of service "service": unexpected envelope type: Exception`,
		},
		{
			msg:             "oneway call error",
			giveRequestBody: fakeEnveloper(wire.OneWay),
			clientOptions:   []ClientOption{Enveloped},
			expectCall:      true,
			wantRequestBody: encodeEnvelopeType(t, wire.OneWay) +
				encodeThriftString(t, "someMethod") +
				encodeEnvelopeSeqID(t, 1) +
				encodeThriftString(t, _irrelevant),
			wantError: "oneway outbound error",
		},
	}

	type streamProtocol = stream.Protocol
	type fakeProtocol struct {
		protocol.Protocol
		streamProtocol
	}

	var _ stream.Protocol = &fakeProtocol{}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			sp := streamtest.NewMockProtocol(mockCtrl)
			if tt.wantRequestBody != "" {
				sp.EXPECT().Writer(gomock.Any()).
					DoAndReturn(func(w io.Writer) stream.Writer {
						return tbinary.Default.Writer(w)
					})
			}

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()

			oneway := transporttest.NewMockOnewayOutbound(mockCtrl)
			if tt.expectCall {
				if tt.wantError != "" {
					oneway.EXPECT().CallOneway(gomock.Any(),
						transporttest.NewRequestMatcher(t, &transport.Request{
							Caller:    "caller",
							Service:   "service",
							Encoding:  Encoding,
							Procedure: "MyService::someMethod",
							Body:      strings.NewReader(tt.wantRequestBody),
						}),
					).Return(nil, errors.New("oneway outbound error"))
				} else {
					oneway.EXPECT().CallOneway(gomock.Any(),
						transporttest.NewRequestMatcher(t, &transport.Request{
							Caller:    "caller",
							Service:   "service",
							Encoding:  Encoding,
							Procedure: "MyService::someMethod",
							Body:      strings.NewReader(tt.wantRequestBody),
						}),
					).Return(&successAck{}, nil)
				}
			}

			opts := tt.clientOptions
			opts = append(opts, Protocol(&fakeProtocol{streamProtocol: sp}))
			nwc := NewNoWire(Config{
				Service: "MyService",
				ClientConfig: clientconfig.MultiOutbound("caller", "service",
					transport.Outbounds{
						Oneway: oneway,
					}),
			}, opts...)

			ack, err := nwc.CallOneway(ctx, tt.giveRequestBody)
			if tt.wantError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "success", ack.String())
		})
	}
}

func TestNoNewWireBadProtocolConfig(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proto := thrifttest.NewMockProtocol(mockCtrl)
	assert.Panics(t,
		func() {
			NewNoWire(Config{}, Protocol(proto))
		})
}

func TestBuildTransportRequestWriteError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sp := streamtest.NewMockProtocol(mockCtrl)
	sw := streamtest.NewMockWriter(mockCtrl)
	sp.EXPECT().Writer(gomock.Any()).Return(sw).AnyTimes()

	nwc := noWireThriftClient{
		cc:         clientconfig.MultiOutbound("caller", "service", transport.Outbounds{}),
		p:          sp,
		Enveloping: true,
	}

	wantEnvHeader := stream.EnvelopeHeader{
		Name:  "someMethod",
		Type:  wire.Call,
		SeqID: 1,
	}

	t.Run("envelope begin", func(t *testing.T) {
		sw.EXPECT().Close().Return(nil)
		sw.EXPECT().WriteEnvelopeBegin(wantEnvHeader).Return(errors.New("writeenvelopebegin error"))

		_, _, err := nwc.buildTransportRequest(fakeEnveloper(wire.Call))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `failed to encode "thrift" request body for procedure "::someMethod" of service "service": writeenvelopebegin error`)
	})

	t.Run("encode", func(t *testing.T) {
		sw.EXPECT().Close().Return(nil)
		sw.EXPECT().WriteEnvelopeBegin(wantEnvHeader).Return(nil)

		_, _, err := nwc.buildTransportRequest(errorEnveloper{envelopeType: wire.Call, err: errors.New("encode error")})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `failed to encode "thrift" request body for procedure "::someMethod" of service "service": encode error`)
	})

	t.Run("encode", func(t *testing.T) {
		sw.EXPECT().Close().Return(nil)
		sw.EXPECT().WriteEnvelopeBegin(wantEnvHeader).Return(nil)
		sw.EXPECT().WriteString(_irrelevant).Return(nil)
		sw.EXPECT().WriteEnvelopeEnd().Return(errors.New("writeenvelopeend error"))

		_, _, err := nwc.buildTransportRequest(fakeEnveloper(wire.Call))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `failed to encode "thrift" request body for procedure "::someMethod" of service "service": writeenvelopeend error`)
	})
}

// encodeThriftString prefixes the passed in string with an int32 that contains
// the length of the string, compliant to the Thrift protocol.
func encodeThriftString(t *testing.T, s string) string {
	t.Helper()

	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(len(s)))
	return string(buf) + s
}

func encodeEnvelopeSeqID(t *testing.T, seqID int) string {
	t.Helper()

	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(seqID))
	return string(buf)
}

func encodeEnvelopeType(t *testing.T, et wire.EnvelopeType) string {
	t.Helper()

	buf := make([]byte, 4)
	version := uint32(0x80010000) | uint32(et)
	binary.BigEndian.PutUint32(buf, version)
	return string(buf)
}
