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

package transport

import "io"

// Response is the low level response representation.
type Response struct {
	// ID is a unique identifier for a request/response pair. This MAY be a
	// trace ID or UUID.
	//
	// If the corresponding transport.Request struct has this field set, this
	// field MUST also be set.
	ID string

	// Service is the name of the responding service.
	Service string

	// Host is the name of the server responding with this reponse.
	//
	// It MAY be set by a an environment-aware middleware.
	Host string

	// Environment is the name of the host environment that the request was
	// issued from. eg "staging", "production"
	//
	// It MAY be set by a an environment-aware middleware.
	Environment string

	Headers          Headers
	Body             io.ReadCloser
	ApplicationError bool
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
