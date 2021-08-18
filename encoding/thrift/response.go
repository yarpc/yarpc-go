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

package thrift

import (
	"go.uber.org/thriftrw/envelope"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/yarpc/yarpcerrors"
)

// Response contains the raw response from a generated Thrift handler.
type Response struct {
	Body envelope.Enveloper

	IsApplicationError bool

	ApplicationErrorDetails string
	ApplicationErrorName    string
	ApplicationErrorCode    *yarpcerrors.Code
}

// NoWireResponse is the response from a generated Thrift handler that can
// process requests that use the "nowire" implementation.
type NoWireResponse struct {
	// Body contains the response body. It knows how to encode itself into
	// the "nowire" representation.
	Body stream.Enveloper

	// ResponseWriter encodes the body into an output stream.
	ResponseWriter stream.ResponseWriter

	// IsApplicationError reports whether the response indicates an
	// appliation-level error. If true, a Thrift exception was thrown by
	// the handler.
	IsApplicationError bool

	ApplicationErrorDetails string
	ApplicationErrorName    string
	ApplicationErrorCode    *yarpcerrors.Code
}
