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
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/envelope"
	"go.uber.org/thriftrw/thrifttest"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/internal/testtime"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctest"
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
		},
		{
			desc:             "happy case without enveloping",
			giveRequestBody:  fakeEnveloper(wire.Call),
			wantRequestBody:  valueptr(wire.NewValueStruct(wire.Struct{})),
			expectCall:       true,
			giveResponseBody: valueptr(wire.NewValueStruct(wire.Struct{})),
		},
		{
			desc:            "wrong envelope type for request",
			clientOptions:   []ClientOption{Enveloped},
			giveRequestBody: fakeEnveloper(wire.Reply),
			wantError: `failed to encode "thrift" request body for procedure ` +
				`"MyService::someMethod" of service "service": unexpected envelope type: Reply`,
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
		},
	}

	for _, tt := range tests {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		proto := thrifttest.NewMockProtocol(mockCtrl)
		payload := []byte("irrelevant")

		if tt.wantRequestEnvelope != nil {
			proto.EXPECT().EncodeEnveloped(*tt.wantRequestEnvelope, gomock.Any()).
				Do(func(_ wire.Envelope, w io.Writer) {
					_, err := w.Write(payload)
					require.NoError(t, err, "Write() failed")
				}).Return(nil)
		}

		if tt.wantRequestBody != nil {
			proto.EXPECT().Encode(*tt.wantRequestBody, gomock.Any()).
				Do(func(_ wire.Value, w io.Writer) {
					_, err := w.Write(payload)
					require.NoError(t, err, "Write() failed")
				}).Return(nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		trans := yarpctest.NewMockUnaryOutbound(mockCtrl)
		if tt.expectCall {
			trans.EXPECT().Call(ctx,
				&yarpc.Request{
					Caller:    "caller",
					Service:   "service",
					Encoding:  Encoding,
					Procedure: "MyService::someMethod",
				},
				gomock.Any(),
			).Return(&yarpc.Response{},
				yarpc.NewBufferBytes(payload),
				nil)
		}

		if tt.giveResponseEnvelope != nil {
			proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(*tt.giveResponseEnvelope, nil)
		}

		if tt.giveResponseBody != nil {
			proto.EXPECT().Decode(gomock.Any(), wire.TStruct).Return(*tt.giveResponseBody, nil)
		}

		opts := tt.clientOptions
		opts = append(opts, Protocol(proto))
		c := New(
			&yarpc.Client{
				Caller:  "caller",
				Service: "service",
				Unary:   trans,
			},
			"MyService",
			opts...)

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
