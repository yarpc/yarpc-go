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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/wire"
)

func TestEncodingHandler(t *testing.T) {
	tests := []struct {
		reqBody       interface{}
		retResponse   Response
		retError      error
		expectedError string
	}{
		{
			reqBody:       "blah",
			expectedError: "tried to handle a non-wire.Value in thrift handler",
		},
		{
			reqBody:       wire.Value{},
			retError:      errors.New("thrift handler error"),
			expectedError: "thrift handler error",
		},
		{
			reqBody:       wire.Value{},
			retResponse:   Response{Body: fakeEnveloper(wire.OneWay)},
			expectedError: "unexpected envelope type: OneWay",
		},
		{
			reqBody: wire.Value{},
			retResponse: Response{Body: errorEnveloper{
				envelopeType: wire.Reply,
				err:          errors.New("could not convert to wire value"),
			}},
			expectedError: "could not convert to wire value",
		},
		{
			reqBody:       wire.Value{},
			retResponse:   Response{Body: fakeEnveloper(wire.Reply), Exception: errors.New("application error")},
			expectedError: "application error",
		},
		{
			reqBody:     wire.Value{},
			retResponse: Response{Body: fakeEnveloper(wire.Reply)},
		},
	}

	for _, tt := range tests {
		h := EncodingHandler(func(context.Context, wire.Value) (Response, error) {
			return tt.retResponse, tt.retError
		})

		resBody, err := h.Handle(context.Background(), tt.reqBody)
		if tt.expectedError != "" {
			require.Error(t, err, "expected error")
			assert.Contains(t, err.Error(), tt.expectedError)
		} else {
			assert.NoError(t, err, "unexpected error")
			assert.NotNil(t, resBody)
		}
	}
}
