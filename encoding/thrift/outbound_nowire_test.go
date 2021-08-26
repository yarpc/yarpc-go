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
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/thrifttest/streamtest"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/testtime"
)

func TestNoWireClientCall(t *testing.T) {
	tests := []struct {
		msg              string
		giveRequestBody  stream.Enveloper
		giveResponseBody []byte
		clientOptions    []ClientOption

		expectCall       bool
		wantRequestBody  []byte
		wantResponseBody string
		wantError        string
	}{
		{
			msg:              "positive case, without enveloping",
			giveRequestBody:  fakeEnveloper(wire.Call),
			giveResponseBody: []byte("\x00\x00\x00\x08" + _response), // len(string) + string
			expectCall:       true,
			wantRequestBody:  []byte("\x00\x00\x00\x0A" + _irrelevant), // len(string) + string
			wantResponseBody: _response,
		},
		{
			msg:             "positive case, with enveloping",
			giveRequestBody: fakeEnveloper(wire.Call),
			giveResponseBody: []byte("\x80\x01\x00\x02" + // strict envelope version + wire.Reply
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x08" + _response), // len(string) + string
			clientOptions: []ClientOption{Enveloped},
			expectCall:    true,
			wantRequestBody: []byte("\x80\x01\x00\x01" + // strict envelope version + wire.Call
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x0A" + _irrelevant), // len(string) + string
			wantResponseBody: _response,
		},
		{
			msg:             "unexpected request envelope type",
			giveRequestBody: fakeEnveloper(wire.Exception),
			wantError:       `failed to encode "thrift" request body for procedure "MyService::someMethod" of service "service": unexpected envelope type: Exception`,
		},
		{
			msg:             "response envelope exception (TApplicationException) decoding error",
			giveRequestBody: fakeEnveloper(wire.Call),
			clientOptions:   []ClientOption{Enveloped},
			giveResponseBody: []byte("\x80\x01\x00\x03" + // strict envelope version + wire.Exception
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01"), // seqID
			expectCall: true,
			wantRequestBody: []byte("\x80\x01\x00\x01" + // strict envelope version + wire.Call
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x0A" + _irrelevant), // len(string) + string
			wantResponseBody: _response,
			wantError:        `failed to decode "thrift" response body for procedure "MyService::someMethod" of service "service": unexpected EOF`,
		},
		{
			msg:             "response envelope exception (TApplicationException) error",
			giveRequestBody: fakeEnveloper(wire.Call),
			clientOptions:   []ClientOption{Enveloped},
			giveResponseBody: []byte("\x80\x01\x00\x03" + // strict envelope version + wire.Exception
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x08" + _response), // len(string) + string
			expectCall: true,
			wantRequestBody: []byte("\x80\x01\x00\x01" + // strict envelope version + wire.Call
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x0A" + _irrelevant), // len(string) + string
			wantResponseBody: _response,
			wantError:        "encountered an internal failure: TApplicationException{}",
		},
		{
			msg:             "unexpected response envelope type",
			giveRequestBody: fakeEnveloper(wire.Call),
			giveResponseBody: []byte("\x80\x01\x00\x01" + // strict envelope version + wire.Reply
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x08" + _response), // len(string) + string
			clientOptions: []ClientOption{Enveloped},
			expectCall:    true,
			wantRequestBody: []byte("\x80\x01\x00\x01" + // strict envelope version + wire.Call
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x0A" + _irrelevant), // len(string) + string
			wantResponseBody: _response,
			wantError:        `failed to decode "thrift" response body for procedure "MyService::someMethod" of service "service": unexpected envelope type: Call`,
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
			if tt.wantRequestBody != nil {
				sp.EXPECT().Writer(gomock.Any()).
					DoAndReturn(func(w io.Writer) stream.Writer {
						return binary.Default.Writer(w)
					}).
					Times(1)
			}

			if tt.wantResponseBody != "" {
				sp.EXPECT().Reader(gomock.Any()).
					DoAndReturn(func(r io.Reader) stream.Reader {
						return binary.Default.Reader(r)
					}).
					Times(1)
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
						Body:      bytes.NewReader(tt.wantRequestBody),
					}),
				).Return(&transport.Response{
					Body: readCloser{bytes.NewReader(tt.giveResponseBody)},
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
		wantRequestBody []byte
		wantError       string
	}{
		{
			msg:             "positive case, without enveloping",
			giveRequestBody: fakeEnveloper(wire.OneWay),
			expectCall:      true,
			wantRequestBody: []byte("\x00\x00\x00\x0A" + _irrelevant), // len(string) + string
		},
		{
			msg:             "positive case, with enveloping",
			giveRequestBody: fakeEnveloper(wire.OneWay),
			clientOptions:   []ClientOption{Enveloped},
			expectCall:      true,
			wantRequestBody: []byte("\x80\x01\x00\x04" + // strict envelope version + wire.Oneway
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x0A" + _irrelevant), // len(string) + string
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
			wantRequestBody: []byte("\x80\x01\x00\x04" + // strict envelope version + wire.Oneway
				"\x00\x00\x00\x0A" + "someMethod" + // len(string) + string
				"\x00\x00\x00\x01" + // seqID
				"\x00\x00\x00\x0A" + _irrelevant), // len(string) + string
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
			if tt.wantRequestBody != nil {
				sp.EXPECT().Writer(gomock.Any()).
					DoAndReturn(func(w io.Writer) stream.Writer {
						return binary.Default.Writer(w)
					}).
					Times(1)
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
							Body:      bytes.NewReader(tt.wantRequestBody),
						}),
					).Return(nil, errors.New("oneway outbound error"))
				} else {
					oneway.EXPECT().CallOneway(gomock.Any(),
						transporttest.NewRequestMatcher(t, &transport.Request{
							Caller:    "caller",
							Service:   "service",
							Encoding:  Encoding,
							Procedure: "MyService::someMethod",
							Body:      bytes.NewReader(tt.wantRequestBody),
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
