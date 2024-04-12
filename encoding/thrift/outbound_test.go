// Copyright (c) 2024 Uber Technologies, Inc.
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
	"go.uber.org/thriftrw/envelope"
	"go.uber.org/thriftrw/thrifttest"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/pkg/procedure"
)

func valueptr(v wire.Value) *wire.Value { return &v }

func TestClient(t *testing.T) {
	tests := []struct {
		desc                 string
		giveRequestBody      envelope.Enveloper // outgoing request body
		giveResponseEnvelope *wire.Envelope     // returned on DecodeEnveloped()
		giveResponseBody     *wire.Value        // return on Decode()
		clientOptions        []ClientOption

		expectCall          bool           // whether outbound.Call is expected
		wantRequestEnvelope *wire.Envelope // expect EncodeEnveloped(x)
		wantRequestBody     *wire.Value    // expect Encode(x)
		wantError           string         // whether an error is expected

		responseBody io.ReadCloser
	}{
		{
			desc:            "happy case",
			clientOptions:   []ClientOption{Enveloped},
			giveRequestBody: fakeEnveloper(wire.Call),
			wantRequestEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 1,
				Type:  wire.Call,
				Value: wire.NewValueStruct(wire.Struct{}),
			},
			expectCall: true,
			giveResponseEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 1,
				Type:  wire.Reply,
				Value: wire.NewValueStruct(wire.Struct{}),
			},
			responseBody: readCloser{bytes.NewReader([]byte("irrelevant"))},
		},
		{
			desc:             "happy case without enveloping",
			giveRequestBody:  fakeEnveloper(wire.Call),
			wantRequestBody:  valueptr(wire.NewValueStruct(wire.Struct{})),
			expectCall:       true,
			giveResponseBody: valueptr(wire.NewValueStruct(wire.Struct{})),
			responseBody:     readCloser{bytes.NewReader([]byte("irrelevant"))},
		},
		{
			desc:            "wrong envelope type for request",
			clientOptions:   []ClientOption{Enveloped},
			giveRequestBody: fakeEnveloper(wire.Reply),
			wantError: `failed to encode "thrift" request body for procedure ` +
				`"MyService::someMethod" of service "service": unexpected envelope type: Reply`,
			responseBody: readCloser{bytes.NewReader([]byte("irrelevant"))},
		},
		{
			desc:            "TApplicationException",
			clientOptions:   []ClientOption{Enveloped},
			giveRequestBody: fakeEnveloper(wire.Call),
			wantRequestEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 1,
				Type:  wire.Call,
				Value: wire.NewValueStruct(wire.Struct{}),
			},
			expectCall: true,
			giveResponseEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 1,
				Type:  wire.Exception,
				Value: wire.NewValueStruct(wire.Struct{Fields: []wire.Field{
					{ID: 1, Value: wire.NewValueString("great sadness")},
					{ID: 2, Value: wire.NewValueI32(7)},
				}}),
			},
			wantError: `thrift request to procedure "MyService::someMethod" of ` +
				`service "service" encountered an internal failure: ` +
				"TApplicationException{Message: great sadness, Type: PROTOCOL_ERROR}",
			responseBody: readCloser{bytes.NewReader([]byte("irrelevant"))},
		},
		{
			desc:            "wrong envelope type for response",
			clientOptions:   []ClientOption{Enveloped},
			giveRequestBody: fakeEnveloper(wire.Call),
			wantRequestEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 1,
				Type:  wire.Call,
				Value: wire.NewValueStruct(wire.Struct{}),
			},
			expectCall: true,
			giveResponseEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 1,
				Type:  wire.Call,
				Value: wire.NewValueStruct(wire.Struct{}),
			},
			wantError: `failed to decode "thrift" response body for procedure ` +
				`"MyService::someMethod" of service "service": unexpected envelope type: Call`,
			responseBody: io.NopCloser(bytes.NewReader([]byte("irrelevant"))),
		},
	}

	for _, tt := range tests {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		proto := thrifttest.NewMockProtocol(mockCtrl)

		if tt.wantRequestEnvelope != nil {
			proto.EXPECT().EncodeEnveloped(*tt.wantRequestEnvelope, gomock.Any()).
				Do(func(_ wire.Envelope, w io.Writer) {
					_, err := w.Write([]byte("irrelevant"))
					require.NoError(t, err, "Write() failed")
				}).Return(nil)
		}

		if tt.wantRequestBody != nil {
			proto.EXPECT().Encode(*tt.wantRequestBody, gomock.Any()).
				Do(func(_ wire.Value, w io.Writer) {
					_, err := w.Write([]byte("irrelevant"))
					require.NoError(t, err, "Write() failed")
				}).Return(nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		trans := transporttest.NewMockUnaryOutbound(mockCtrl)
		if tt.expectCall {
			trans.EXPECT().Call(ctx,
				transporttest.NewRequestMatcher(t, &transport.Request{
					Caller:    "caller",
					Service:   "service",
					Encoding:  Encoding,
					Procedure: "MyService::someMethod",
					Body:      bytes.NewReader([]byte("irrelevant")),
				}),
			).Return(&transport.Response{
				Body: tt.responseBody,
			}, nil)
		}

		if tt.giveResponseEnvelope != nil {
			proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(*tt.giveResponseEnvelope, nil)
		}

		if tt.giveResponseBody != nil {
			proto.EXPECT().Decode(gomock.Any(), wire.TStruct).Return(*tt.giveResponseBody, nil)
		}

		opts := tt.clientOptions
		opts = append(opts, Protocol(proto))
		c := New(Config{
			Service: "MyService",
			ClientConfig: clientconfig.MultiOutbound("caller", "service",
				transport.Outbounds{
					Unary: trans,
				}),
		}, opts...)

		_, err := c.Call(ctx, tt.giveRequestBody)
		if tt.wantError != "" {
			if assert.Error(t, err, "%v: expected failure", tt.desc) {
				assert.Contains(t, err.Error(), tt.wantError, "%v: error mismatch", tt.desc)
			}
		} else {
			assert.NoError(t, err, "%v: expected success", tt.desc)
		}
	}
}

type successAck struct{}

func (a successAck) String() string {
	return "success"
}

func TestClientOneway(t *testing.T) {
	caller, service, procedureName := "caller", "MyService", "someMethod"

	tests := []struct {
		desc            string
		giveRequestBody envelope.Enveloper // outgoing request body
		clientOptions   []ClientOption

		expectCall          bool           // whether outbound.Call is expected
		wantRequestEnvelope *wire.Envelope // expect EncodeEnveloped(x)
		wantRequestBody     *wire.Value    // expect Encode(x)
		wantError           string         // whether an error is expected
	}{
		{
			desc:            "happy case",
			giveRequestBody: fakeEnveloper(wire.Call),
			clientOptions:   []ClientOption{Enveloped},

			expectCall: true,
			wantRequestEnvelope: &wire.Envelope{
				Name:  procedureName,
				SeqID: 1,
				Type:  wire.Call,
				Value: wire.NewValueStruct(wire.Struct{}),
			},
		},
		{
			desc:            "happy case without enveloping",
			giveRequestBody: fakeEnveloper(wire.Call),

			expectCall:      true,
			wantRequestBody: valueptr(wire.NewValueStruct(wire.Struct{})),
		},
		{
			desc:            "wrong envelope type for request",
			giveRequestBody: fakeEnveloper(wire.Reply),
			clientOptions:   []ClientOption{Enveloped},

			wantError: `failed to encode "thrift" request body for procedure ` +
				`"MyService::someMethod" of service "MyService": unexpected envelope type: Reply`,
		},
	}

	for _, tt := range tests {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		proto := thrifttest.NewMockProtocol(mockCtrl)
		bodyBytes := []byte("irrelevant")

		if tt.wantRequestEnvelope != nil {
			proto.EXPECT().EncodeEnveloped(*tt.wantRequestEnvelope, gomock.Any()).
				Do(func(_ wire.Envelope, w io.Writer) {
					_, err := w.Write(bodyBytes)
					require.NoError(t, err, "Write() failed")
				}).Return(nil)
		}

		if tt.wantRequestBody != nil {
			proto.EXPECT().Encode(*tt.wantRequestBody, gomock.Any()).
				Do(func(_ wire.Value, w io.Writer) {
					_, err := w.Write(bodyBytes)
					require.NoError(t, err, "Write() failed")
				}).Return(nil)
		}

		ctx := context.Background()

		onewayOutbound := transporttest.NewMockOnewayOutbound(mockCtrl)

		requestMatcher := transporttest.NewRequestMatcher(t, &transport.Request{
			Caller:    caller,
			Service:   service,
			Encoding:  Encoding,
			Procedure: procedure.ToName(service, procedureName),
			Body:      bytes.NewReader(bodyBytes),
		})

		if tt.expectCall {
			if tt.wantError != "" {
				onewayOutbound.
					EXPECT().
					CallOneway(ctx, requestMatcher).
					Return(nil, errors.New(tt.wantError))
			} else {
				onewayOutbound.
					EXPECT().
					CallOneway(ctx, requestMatcher).
					Return(&successAck{}, nil)
			}
		}
		opts := tt.clientOptions
		opts = append(opts, Protocol(proto))

		c := New(Config{
			Service: service,
			ClientConfig: clientconfig.MultiOutbound(caller, service,
				transport.Outbounds{
					Oneway: onewayOutbound,
				}),
		}, opts...)

		ack, err := c.CallOneway(ctx, tt.giveRequestBody)
		if tt.wantError != "" {
			if assert.Error(t, err, "%v: expected failure", tt.desc) {
				assert.Contains(t, err.Error(), tt.wantError, "%v: error mismatch", tt.desc)
			}
		} else {
			assert.NoError(t, err, "%v: expected success", tt.desc)
			assert.Equal(t, "success", ack.String())
		}
	}
}

type readCloser struct {
	*bytes.Reader
}

func (r readCloser) Close() error { return nil }
