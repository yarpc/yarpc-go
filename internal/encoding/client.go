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
	"strings"

	"go.uber.org/yarpc/api/transport"
)

type clientEncodingError struct {
	Encoding  transport.Encoding
	Service   string
	Procedure string
	Reason    error

	// These parameters control whether the error is for a request or a response,
	// and whether it's for a header or body.

	IsResponse bool
	IsHeader   bool
}

func (e clientEncodingError) Error() string {
	parts := []string{"failed to"}
	if e.IsResponse {
		parts = append(parts, fmt.Sprintf("decode %q response", string(e.Encoding)))
	} else {
		parts = append(parts, fmt.Sprintf("encode %q request", string(e.Encoding)))
	}
	if e.IsHeader {
		parts = append(parts, "headers")
	} else {
		parts = append(parts, "body")
	}
	parts = append(parts,
		fmt.Sprintf("for procedure %q of service %q: %v",
			e.Procedure, e.Service, e.Reason))
	return strings.Join(parts, " ")
}

func newClientEncodingError(req *transport.Request, err error) clientEncodingError {
	return clientEncodingError{
		Encoding:  req.Encoding,
		Service:   req.Service,
		Procedure: req.Procedure,
		Reason:    err,
	}
}

// RequestBodyEncodeError builds an error that represents a failure to encode
// the request body.
func RequestBodyEncodeError(req *transport.Request, err error) error {
	return newClientEncodingError(req, err)
}

// ResponseBodyDecodeError builds an error that represents a failure to decode
// the response body.
func ResponseBodyDecodeError(req *transport.Request, err error) error {
	e := newClientEncodingError(req, err)
	e.IsResponse = true
	return e
}

// RequestHeadersEncodeError builds an error that represents a failure to
// encode the request headers.
func RequestHeadersEncodeError(req *transport.Request, err error) error {
	e := newClientEncodingError(req, err)
	e.IsHeader = true
	return e
}

// ResponseHeadersDecodeError builds an error that represents a failure to
// decode the response headers.
func ResponseHeadersDecodeError(req *transport.Request, err error) error {
	e := newClientEncodingError(req, err)
	e.IsHeader = true
	e.IsResponse = true
	return e
}
