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

package yarpcencoding_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcencoding"
	"go.uber.org/yarpc/v2/yarpcerror"
)

func TestExpectEncodings(t *testing.T) {
	assert.Error(t, yarpcencoding.ExpectEncodings(&yarpc.Request{}, "foo"))
	assert.NoError(t, yarpcencoding.ExpectEncodings(&yarpc.Request{Encoding: "foo"}, "foo"))
	assert.NoError(t, yarpcencoding.ExpectEncodings(&yarpc.Request{Encoding: "foo"}, "foo", "bar"))
	assert.Error(t, yarpcencoding.ExpectEncodings(&yarpc.Request{Encoding: "foo"}, "bar"))
	assert.Error(t, yarpcencoding.ExpectEncodings(&yarpc.Request{Encoding: "foo"}, "bar", "baz"))
}

func TestEncodeErrors(t *testing.T) {
	tests := []struct {
		errorFunc     func(*yarpc.Request, error) error
		expectedCode  yarpcerror.Code
		expectedWords []string
	}{
		{
			errorFunc:     yarpcencoding.RequestBodyEncodeError,
			expectedCode:  yarpcerror.CodeInvalidArgument,
			expectedWords: []string{"request", "body", "encode"},
		},
		{
			errorFunc:     yarpcencoding.RequestHeadersEncodeError,
			expectedCode:  yarpcerror.CodeInvalidArgument,
			expectedWords: []string{"request", "headers", "encode"},
		},
		{
			errorFunc:     yarpcencoding.RequestBodyDecodeError,
			expectedCode:  yarpcerror.CodeInvalidArgument,
			expectedWords: []string{"request", "body", "decode"},
		},
		{
			errorFunc:     yarpcencoding.RequestHeadersDecodeError,
			expectedCode:  yarpcerror.CodeInvalidArgument,
			expectedWords: []string{"request", "headers", "decode"},
		},
		{
			errorFunc:     yarpcencoding.ResponseBodyEncodeError,
			expectedCode:  yarpcerror.CodeInvalidArgument,
			expectedWords: []string{"response", "body", "encode"},
		},
		{
			errorFunc:     yarpcencoding.ResponseHeadersEncodeError,
			expectedCode:  yarpcerror.CodeInvalidArgument,
			expectedWords: []string{"response", "headers", "encode"},
		},
		{
			errorFunc:     yarpcencoding.ResponseBodyDecodeError,
			expectedCode:  yarpcerror.CodeInvalidArgument,
			expectedWords: []string{"response", "body", "decode"},
		},
		{
			errorFunc:     yarpcencoding.ResponseHeadersDecodeError,
			expectedCode:  yarpcerror.CodeInvalidArgument,
			expectedWords: []string{"response", "headers", "decode"},
		},
	}
	request := &yarpc.Request{}
	for _, tt := range tests {
		assertError(t, tt.errorFunc(request, errors.New("")), tt.expectedCode, tt.expectedWords...)
	}
}

func assertError(t *testing.T, err error, expectedCode yarpcerror.Code, expectedWords ...string) {
	assert.Error(t, err)
	assert.Equal(t, expectedCode, yarpcerror.GetInfo(err).Code)
	for _, expectedWord := range expectedWords {
		assert.Contains(t, err.Error(), expectedWord)
	}
}
