// Copyright (c) 2026 Uber Technologies, Inc.
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

// RequestBodyDecodeError builds a YARPC error with code
// yarpcerrors.CodeInvalidArgument that represents a failure to decode
// the request body.
func RequestBodyDecodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, nil, false /*isResponse*/, false /*isHeader*/, err)
}

// ResponseBodyEncodeError builds a YARPC error with code
// yarpcerrors.CodeInvalidArgument that represents a failure to encode
// the response body.
func ResponseBodyEncodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, nil, true /*isResponse*/, false /*isHeader*/, err)
}

// RequestHeadersDecodeError builds a YARPC error with code
// yarpcerrors.CodeInvalidArgument that represents a failure to
// decode the request headers.
func RequestHeadersDecodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, nil, false /*isResponse*/, true /*isHeader*/, err)
}

// ResponseHeadersEncodeError builds a YARPC error with code
// yarpcerrors.CodeInvalidArgument that represents a failure to
// encode the response headers.
func ResponseHeadersEncodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, nil, true /*isResponse*/, true /*isHeader*/, err)
}

// ExpectEncodings verifies that the given request has one of the given
// encodings, otherwise it returns a YARPC error with code
// yarpcerrors.CodeInvalidArgument.
func ExpectEncodings(req *transport.Request, want ...transport.Encoding) error {
	got := req.Encoding
	for _, w := range want {
		if w == got {
			return nil
		}
	}

	return newServerEncodingError(req, want, false /*isResponse*/, false /*isHeader*/, newEncodingMismatchError(want, got))
}

func newServerEncodingError(req *transport.Request, encodings []transport.Encoding, isResponse bool, isHeader bool, err error) error {
	if len(encodings) == 0 {
		encodings = []transport.Encoding{req.Encoding}
	}
	parts := []string{"failed to"}
	if isResponse {
		switch len(encodings) {
		case 1:
			parts = append(parts, fmt.Sprintf("encode %q response", string(encodings[0])))
		default:
			parts = append(parts, fmt.Sprintf("encode %v response", encodings))
		}
	} else {
		switch len(encodings) {
		case 1:
			parts = append(parts, fmt.Sprintf("decode %q request", string(encodings[0])))
		default:
			parts = append(parts, fmt.Sprintf("decode %v request", encodings))
		}
	}
	if isHeader {
		parts = append(parts, "headers")
	} else {
		parts = append(parts, "body")
	}
	parts = append(parts,
		fmt.Sprintf("for procedure %q of service %q from caller %q: %v",
			req.Procedure, req.Service, req.Caller, err))
	return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, strings.Join(parts, " "))
}

// newEncodingMismatchError returns an error indicating that the request
// encoding did not match any of the expected encodings.
func newEncodingMismatchError(want []transport.Encoding, got transport.Encoding) error {
	switch len(want) {
	case 1:
		return fmt.Errorf("expected encoding %q but got %q", want[0], got)
	default:
		return fmt.Errorf("expected one of encodings %v but got %q", want, got)
	}
}
