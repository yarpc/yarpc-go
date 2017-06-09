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
	"strconv"

	"go.uber.org/yarpc/api/errors"
	"go.uber.org/yarpc/api/transport"
)

// RequestBodyEncodeError builds an error that represents a failure to encode
// the request body.
func RequestBodyEncodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, err, true, false)
}

// ResponseBodyDecodeError builds an error that represents a failure to decode
// the response body.
func ResponseBodyDecodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, err, false, false)
}

// RequestHeadersEncodeError builds an error that represents a failure to
// encode the request headers.
func RequestHeadersEncodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, err, true, true)
}

// ResponseHeadersDecodeError builds an error that represents a failure to
// decode the response headers.
func ResponseHeadersDecodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, err, false, true)
}

func newClientEncodingError(req *transport.Request, err error, isRequest bool, isHeaders bool) error {
	return errors.InvalidArgument(
		"encoding", string(req.Encoding),
		"service", req.Service,
		"procedure", req.Procedure,
		"is_request", strconv.FormatBool(isRequest),
		"is_headers", strconv.FormatBool(isHeaders),
		"error", err.Error(),
	)
}
