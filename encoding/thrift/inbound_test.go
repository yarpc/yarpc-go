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
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thriftrw/thriftrw-go/wire"
	"golang.org/x/net/context"
)

//go:generate mockgen -destination=mock_handler_test.go -package=thrift github.com/yarpc/yarpc-go/encoding/thrift Handler
//go:generate mockgen -destination=mock_protocol_test.go -package=thrift github.com/thriftrw/thriftrw-go/protocol Protocol

func TestThriftHandler(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	requestBody := wire.NewValueStruct(wire.Struct{})
	responseBody := wire.NewValueStruct(wire.Struct{})

	proto := NewMockProtocol(mockCtrl)

	proto.EXPECT().Decode(gomock.Any(), wire.TStruct).
		Return(requestBody, nil).AnyTimes()

	proto.EXPECT().Encode(responseBody, gomock.Any()).
		Do(func(_ wire.Value, w io.Writer) {
			_, err := w.Write([]byte("hello"))
			require.NoError(t, err, "Write() failed")
		}).Return(nil).AnyTimes()

	tests := []bool{true, false}
	for _, isApplicationError := range tests {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)

		handler := NewMockHandler(mockCtrl)
		handler.EXPECT().Handle(
			fakeReqMeta{
				context:   ctx,
				caller:    "caller",
				service:   "service",
				encoding:  Encoding,
				procedure: "MyService::someMethod",
			},
			requestBody,
		).Return(Response{
			Body:               emptyEnveloper{},
			IsApplicationError: isApplicationError,
		}, nil)

		h := thriftHandler{Handler: handler, Protocol: proto}

		rw := new(transporttest.FakeResponseWriter)
		err := h.Handle(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  Encoding,
			Procedure: "MyService::someMethod",
			Body:      bytes.NewReader([]byte("irrelevant")),
		}, rw)
		require.NoError(t, err, "request failed")

		assert.Equal(t, isApplicationError, rw.IsApplicationError,
			"isApplicationError did not match")
		assert.Equal(t, rw.Body.String(), "hello", "body did not match")
	}
}

type emptyEnveloper struct{}

func (emptyEnveloper) MethodName() string {
	return "someMethod"
}

func (emptyEnveloper) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}

func (emptyEnveloper) ToWire() (wire.Value, error) {
	return wire.NewValueStruct(wire.Struct{}), nil
}

type fakeReqMeta struct {
	context   context.Context
	caller    string
	service   string
	procedure string
	encoding  transport.Encoding
	headers   yarpc.Headers
}

func (f fakeReqMeta) Matches(x interface{}) bool {
	reqMeta, ok := x.(yarpc.ReqMeta)
	if !ok {
		return false
	}

	// TODO: log to testing.T on mismatch if test becomes more complex
	if f.context != reqMeta.Context() {
		return false
	}

	if f.caller != reqMeta.Caller() {
		return false
	}
	if f.service != reqMeta.Service() {
		return false
	}
	if f.procedure != reqMeta.Procedure() {
		return false
	}
	if f.encoding != reqMeta.Encoding() {
		return false
	}
	if !reflect.DeepEqual(f.headers, reqMeta.Headers()) {
		return false
	}
	return true
}

func (f fakeReqMeta) String() string {
	return fmt.Sprintf("%#v", f)
}
