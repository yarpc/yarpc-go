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

package encoding

import (
	"fmt"
	"strings"

	"github.com/yarpc/yarpc-go/transport"
)

type serverEncodingError struct {
	Encoding  transport.Encoding
	Caller    string
	Service   string
	Procedure string
	Reason    error

	// These parameters control whether the error is for a request or a response,
	// and whether it's for a header or body.

	IsResponse bool
	IsHeader   bool
}

func (e serverEncodingError) Error() string {
	parts := []string{"failed to"}
	if e.IsResponse {
		parts = append(parts, fmt.Sprintf("encode %q response", string(e.Encoding)))
	} else {
		parts = append(parts, fmt.Sprintf("decode %q request", string(e.Encoding)))
	}
	if e.IsHeader {
		parts = append(parts, "headers")
	} else {
		parts = append(parts, "body")
	}
	parts = append(parts,
		fmt.Sprintf("for procedure %q of service %q from caller %q: %v",
			e.Procedure, e.Service, e.Caller, e.Reason))
	return strings.Join(parts, " ")
}

// AsHandlerError converts this error into a handler-level error.
func (e serverEncodingError) AsHandlerError() transport.HandlerError {
	if e.IsResponse {
		return transport.UnexpectedError{Reason: e}
	}
	return transport.BadRequestError{Reason: e}
}

func newServerEncodingError(req *transport.Request, err error) serverEncodingError {
	return serverEncodingError{
		Encoding:  req.Encoding,
		Caller:    req.Caller,
		Service:   req.Service,
		Procedure: req.Procedure,
		Reason:    err,
	}
}

// RequestBodyDecodeError builds an error that represents a failure to decode
// the request body.
func RequestBodyDecodeError(req *transport.Request, err error) error {
	return newServerEncodingError(req, err)
}

// ResponseBodyEncodeError builds an error that represents a failure to encode
// the response body.
func ResponseBodyEncodeError(req *transport.Request, err error) error {
	e := newServerEncodingError(req, err)
	e.IsResponse = true
	return e
}

// RequestHeadersDecodeError builds an error that represents a failure to
// decode the request headers.
func RequestHeadersDecodeError(req *transport.Request, err error) error {
	e := newServerEncodingError(req, err)
	e.IsHeader = true
	return e
}

// ResponseHeadersEncodeError builds an error that represents a failure to
// encode the response headers.
func ResponseHeadersEncodeError(req *transport.Request, err error) error {
	e := newServerEncodingError(req, err)
	e.IsResponse = true
	e.IsHeader = true
	return e
}
