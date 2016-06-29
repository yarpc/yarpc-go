// Copyright (c) 2016 Uber Technologies, Inc.
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
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thriftrw/thriftrw-go/envelope"
	"github.com/thriftrw/thriftrw-go/wire"
	"golang.org/x/net/context"
)

func TestClient(t *testing.T) {
	tests := []struct {
		giveRequestBody      envelope.Enveloper // outgoing request body
		giveResponseEnvelope *wire.Envelope     // returned response envelope

		expectCall          bool           // whether outbound.Call is expected
		wantRequestEnvelope *wire.Envelope // expected envelope to encode
		wantError           string         // whether an error is expected
	}{
		{
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
			giveRequestBody: fakeEnveloper(wire.Reply),
			wantError: `failed to encode "thrift" request body for procedure ` +
				`"MyService::someMethod" of service "service": unexpected envelope type: Reply`,
		},
		{
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
				"TApplicationException{Message: great sadness, Type: ProtocolError}",
		},
		{
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

		proto := NewMockProtocol(mockCtrl)
		if tt.wantRequestEnvelope != nil {
			proto.EXPECT().EncodeEnveloped(*tt.wantRequestEnvelope, gomock.Any()).
				Do(func(_ wire.Envelope, w io.Writer) {
					_, err := w.Write([]byte("irrelevant"))
					require.NoError(t, err, "Write() failed")
				}).Return(nil)
		}

		ctx, _ := context.WithTimeout(context.Background(), time.Second)

		trans := transporttest.NewMockOutbound(mockCtrl)
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
				Body: ioutil.NopCloser(bytes.NewReader([]byte("irrelevant"))),
			}, nil)
		}

		if tt.giveResponseEnvelope != nil {
			proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(*tt.giveResponseEnvelope, nil)
		}

		c := thriftClient{
			t:             trans,
			p:             proto,
			thriftService: "MyService",
			caller:        "caller",
			service:       "service",
		}

		_, _, err := c.Call(yarpc.NewReqMeta(ctx), tt.giveRequestBody)
		if tt.wantError != "" {
			if assert.Error(t, err, "expected failure") {
				assert.Contains(t, err.Error(), tt.wantError)
			}
		} else {
			assert.NoError(t, err, "expected success")
		}
	}
}
