// Copyright (c) 2017 Uber Technologies, Inc.
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

package encoding

import (
	"fmt"
	"strconv"

	"go.uber.org/yarpc/api/errors"
	"go.uber.org/yarpc/api/transport"
)

// RequestBodyDecodeError builds an error that represents a failure to decode
// the request body.
func RequestBodyDecodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, err, true, false)
}

// ResponseBodyEncodeError builds an error that represents a failure to encode
// the response body.
func ResponseBodyEncodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, err, false, false)
}

// RequestHeadersDecodeError builds an error that represents a failure to
// decode the request headers.
func RequestHeadersDecodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, err, true, true)
}

// ResponseHeadersEncodeError builds an error that represents a failure to
// encode the response headers.
func ResponseHeadersEncodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, err, false, true)
}

// Expect verifies that the given request has one of the given encodings
// or it returns an error.
func Expect(req *transport.Request, want ...transport.Encoding) error {
	got := req.Encoding
	for _, w := range want {
		if w == got {
			return nil
		}
	}
	return newServerEncodingError(req, fmt.Errorf("expected one of encodings %v but got %q", want, got), true, false)
}

func newServerEncodingError(req *transport.Request, err error, isRequest bool, isHeaders bool) error {
	return errors.InvalidArgument(
		"encoding", string(req.Encoding),
		"caller", req.Caller,
		"service", req.Service,
		"procedure", req.Procedure,
		"is_request", strconv.FormatBool(isRequest),
		"is_headers", strconv.FormatBool(isHeaders),
		"error", err.Error(),
	)
}
