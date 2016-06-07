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
	"testing"
	"time"

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
			&ReqMeta{Context: ctx},
			requestBody,
		).Return(Response{
			Body:               wire.NewValueStruct(wire.Struct{}),
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
