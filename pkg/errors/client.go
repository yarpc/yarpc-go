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

package errors

import (
	"fmt"
	"strings"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

// RequestBodyEncodeError builds a YARPC error with code
// yarpcerrors.CodeInternal that represents a failure to encode
// the request body.
func RequestBodyEncodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, false /*isResponse*/, false /*isHeader*/, err)
}

// ResponseBodyDecodeError builds a YARPC error with code
// yarpcerrors.CodeInternal that represents a failure to decode
// the response body.
func ResponseBodyDecodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, true /*isResponse*/, false /*isHeader*/, err)
}

// RequestHeadersEncodeError builds a YARPC error with code
// yarpcerrors.CodeInternal that represents a failure to
// encode the request headers.
func RequestHeadersEncodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, false /*isResponse*/, true /*isHeader*/, err)
}

// ResponseHeadersDecodeError builds a YARPC error with code
// yarpcerrors.CodeInternal that represents a failure to
// decode the response headers.
func ResponseHeadersDecodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, true /*isResponse*/, true /*isHeader*/, err)
}

func newClientEncodingError(req *transport.Request, isResponse bool, isHeader bool, err error) error {
	parts := []string{"failed to"}
	if isResponse {
		parts = append(parts, fmt.Sprintf("decode %q response", string(req.Encoding)))
	} else {
		parts = append(parts, fmt.Sprintf("encode %q request", string(req.Encoding)))
	}
	if isHeader {
		parts = append(parts, "headers")
	} else {
		parts = append(parts, "body")
	}
	parts = append(parts,
		fmt.Sprintf("for procedure %q of service %q: %v",
			req.Procedure, req.Service, err))
	return yarpcerrors.Newf(yarpcerrors.CodeInternal, strings.Join(parts, " "))
}
