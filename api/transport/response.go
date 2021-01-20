// Copyright (c) 2021 Uber Technologies, Inc.
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

package transport

import (
	"io"

	"go.uber.org/yarpc/yarpcerrors"
)

// Response is the low level response representation.
type Response struct {
	Headers Headers
	Body    io.ReadCloser
	// Response payload size before any compression applied by the protocol
	// When using the HTTP transport, this value is set from the HTTP header
	// content-length. It should be noted that this value is set manually and
	// will not be updated automatically if the body is being modified
	BodySize int

	ApplicationError bool
	// ApplicationErrorMeta adds information about the application error.
	// This field will only be set if `ApplicationError` is true.
	ApplicationErrorMeta *ApplicationErrorMeta
}

// ResponseWriter allows Handlers to write responses in a streaming fashion.
//
// Functions on ResponseWriter are not thread-safe.
type ResponseWriter interface {
	io.Writer

	// AddHeaders adds the given headers to the response. If called, this MUST
	// be called before any invocation of Write().
	//
	// This MUST NOT panic if Headers is nil.
	AddHeaders(Headers)
	// TODO(abg): Ability to set individual headers instead?

	// SetApplicationError specifies that this response contains an
	// application error. If called, this MUST be called before any invocation
	// of Write().
	SetApplicationError()
}

// ApplicationErrorMeta contains additional information to describe the
// application error, using an error name, code and details string.
//
// Fields are optional for backwards-compatibility and may not be present in all
// responses.
type ApplicationErrorMeta struct {
	Code    *yarpcerrors.Code
	Name    string
	Details string
}

// ApplicationErrorMetaSetter enables setting the name of an
// application error, surfacing it in metrics.
//
// Conditionally upcast a ResponseWriter to access the
// functionality.
type ApplicationErrorMetaSetter interface {
	// SetApplicationErrorMeta specifies the name of the application error, if any.
	// If called, this MUST be called before any invocation of Write().
	SetApplicationErrorMeta(*ApplicationErrorMeta)
}
