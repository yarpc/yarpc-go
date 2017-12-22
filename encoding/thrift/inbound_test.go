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

package thrift

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/testtime"
)

func TestThriftHandler(t *testing.T) {
	requestBody := wire.NewValueStruct(wire.Struct{})
	responseBody := wire.NewValueStruct(wire.Struct{})

	tests := []struct {
		giveEnvelope         *wire.Envelope    // envelope read off the wire
		responseEnvelopeType wire.EnvelopeType // envelope type returned by handler
		responseIsAppError   bool              // whether the handler encountered an application error

		wantEnvelope *wire.Envelope // envelope expected written to the wire
		expectHandle bool           // whether an actual call to the handler is expected
		wantError    string         // if non empty, an error is expected
	}{
		{
			giveEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 42,
				Type:  wire.Call,
				Value: requestBody,
			},
			wantEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 42, // response seqID must match
				Type:  wire.Reply,
				Value: responseBody,
			},
			expectHandle:         true,
			responseEnvelopeType: wire.Reply,
		},
		{
			giveEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 42,
				Type:  wire.Call,
				Value: requestBody,
			},
			wantEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 42, // response seqID must match
				Type:  wire.Reply,
				Value: responseBody,
			},
			expectHandle:         true,
			responseIsAppError:   true,
			responseEnvelopeType: wire.Reply,
		},
		{
			giveEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 42,
				Type:  wire.Exception,
				Value: requestBody,
			},
			wantError: `failed to decode "thrift" request body for procedure ` +
				`"MyService::someMethod" of service "service" from caller "caller": ` +
				"unexpected envelope type: Exception",
		},
		{
			giveEnvelope: &wire.Envelope{
				Name:  "someMethod",
				SeqID: 42,
				Type:  wire.Call,
				Value: requestBody,
			},
			expectHandle:         true,
			responseEnvelopeType: wire.OneWay,
			wantError: `failed to encode "thrift" response body for procedure ` +
				`"MyService::someMethod" of service "service" from caller "caller": ` +
				"unexpected envelope type: OneWay",
		},
	}

	for _, tt := range tests {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		proto := NewMockProtocol(mockCtrl)
		if tt.giveEnvelope != nil {
			proto.EXPECT().DecodeEnveloped(gomock.Any()).Return(*tt.giveEnvelope, nil)
		}
		if tt.wantEnvelope != nil {
			proto.EXPECT().EncodeEnveloped(*tt.wantEnvelope, gomock.Any()).
				Do(func(_ wire.Envelope, w io.Writer) {
					_, err := w.Write([]byte("hello"))
					require.NoError(t, err, "Write() failed")
				}).Return(nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()

		handler := func(ctx context.Context, w wire.Value) (Response,
			error) {

			if tt.expectHandle {
				assert.Equal(t, requestBody, w)
			}
			return Response{
				Body:               fakeEnveloper(tt.responseEnvelopeType),
				IsApplicationError: tt.responseIsAppError,
			}, nil
		}
		h := thriftUnaryHandler{Protocol: proto, UnaryHandler: handler, Enveloping: true}

		rw := new(transporttest.FakeResponseWriter)
		err := h.Handle(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  Encoding,
			Procedure: "MyService::someMethod",
			Body:      bytes.NewReader([]byte("irrelevant")),
		}, rw)

		if tt.wantError != "" {
			if assert.Error(t, err, "expected an error") {
				assert.Contains(t, err.Error(), tt.wantError)
			}
		} else {
			assert.NoError(t, err, "expected no error")
			assert.Equal(t, tt.responseIsAppError, rw.IsApplicationError,
				"isApplicationError did not match")
			assert.Equal(t, rw.Body.String(), "hello", "body did not match")
		}
	}
}
