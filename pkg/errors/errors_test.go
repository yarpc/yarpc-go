// Copyright (c) 2019 Uber Technologies, Inc.
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

package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestWrapHandlerError(t *testing.T) {
	assert.Nil(t, WrapHandlerError(nil, "foo", "bar"))
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcerrors.FromError(WrapHandlerError(yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, ""), "foo", "bar")).Code())
	assert.Equal(t, yarpcerrors.CodeUnknown, yarpcerrors.FromError(WrapHandlerError(errors.New(""), "foo", "bar")).Code())
}

func TestExpectEncodings(t *testing.T) {
	assert.Error(t, ExpectEncodings(&transport.Request{}, "foo"))
	assert.NoError(t, ExpectEncodings(&transport.Request{Encoding: "foo"}, "foo"))
	assert.NoError(t, ExpectEncodings(&transport.Request{Encoding: "foo"}, "foo", "bar"))
	assert.Error(t, ExpectEncodings(&transport.Request{Encoding: "foo"}, "bar"))
	assert.Error(t, ExpectEncodings(&transport.Request{Encoding: "foo"}, "bar", "baz"))
}

func TestEncodeErrors(t *testing.T) {
	tests := []struct {
		errorFunc     func(*transport.Request, error) error
		expectedCode  yarpcerrors.Code
		expectedWords []string
	}{
		{
			errorFunc:     RequestBodyEncodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"request", "body", "encode"},
		},
		{
			errorFunc:     RequestHeadersEncodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"request", "headers", "encode"},
		},
		{
			errorFunc:     RequestBodyDecodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"request", "body", "decode"},
		},
		{
			errorFunc:     RequestHeadersDecodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"request", "headers", "decode"},
		},
		{
			errorFunc:     ResponseBodyEncodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"response", "body", "encode"},
		},
		{
			errorFunc:     ResponseHeadersEncodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"response", "headers", "encode"},
		},
		{
			errorFunc:     ResponseBodyDecodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"response", "body", "decode"},
		},
		{
			errorFunc:     ResponseHeadersDecodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"response", "headers", "decode"},
		},
	}
	request := &transport.Request{}
	for _, tt := range tests {
		assertError(t, tt.errorFunc(request, errors.New("")), tt.expectedCode, tt.expectedWords...)
	}
}

func assertError(t *testing.T, err error, expectedCode yarpcerrors.Code, expectedWords ...string) {
	assert.Error(t, err)
	assert.Equal(t, expectedCode, yarpcerrors.FromError(err).Code())
	for _, expectedWord := range expectedWords {
		assert.Contains(t, err.Error(), expectedWord)
	}
}
