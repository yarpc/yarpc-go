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

package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestCodes(t *testing.T) {
	for code, statusCode := range _codeToStatusCode {
		t.Run(code.String(), func(t *testing.T) {
			getStatusCode, ok := _codeToStatusCode[code]
			require.True(t, ok)
			require.Equal(t, statusCode, getStatusCode)
			getCodes, ok := _statusCodeToCodes[statusCode]
			require.True(t, ok)
			require.Contains(t, getCodes, code)
			require.Contains(t, getCodes, statusCodeToBestCode(statusCode))
		})
	}
}

func TestUnspecifiedCodes(t *testing.T) {
	tests := []struct {
		name string
		give int
		want yarpcerrors.Code
	}{
		{
			name: "code not modified",
			give: 304,
			want: yarpcerrors.CodeOK,
		},
		{
			name: "code temporary redirection",
			give: 307, // test for an x in range: [300, 400)
			want: yarpcerrors.CodeInvalidArgument,
		},
		{
			name: "code unprocessable context",
			give: 422,
			want: yarpcerrors.CodeInvalidArgument,
		},
		{
			name: "code invalid argument",
			give: 450, // test for an x in range: [400, 500)
			want: yarpcerrors.CodeInvalidArgument,
		},
		{
			name: "code unkown",
			give: 1000,
			want: yarpcerrors.CodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errCode := statusCodeToBestCode(tt.give)
			assert.Equal(t, tt.want, errCode, "yarpc error code did not match")
		})
	}
}
