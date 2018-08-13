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
	// ID of the response as chosen by the client. This MAY be a trace ID or
	// UUID.
	//
	// If the corresponding transport.Request struct has this field set, this
	// field MUST have the same value.
	ID string

	// Host is the name of the server responding with this reponse.
	//
	// It MAY be set by a an environment-aware middleware.
	Host string

	// Environment is the name of the host environment that the request was
	// issued from. eg "staging", "production"
	//
	// It MAY be set by a an environment-aware middleware.
	Environment string

	// Service is the name of the responding service.
	Service string

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

// ResponseMetaWriter returns a ResponseMeta struct that handlers and inbound
// middleware may use to modify transport.Response fields.
//
// Middleware and handlers may attempt to upcast ResponseWriters to
// ResponseMetaWriters to access and write response metadata. Failure to cast
// MUST be handled.
//
//  if metaW, ok := resW.(transport.ResponseMetaWriter); ok {
//   if meta := metaW.ResponseMeta(); meta != nil{
//     meta.Host = "foo"
//     ...
//   }
//  }
//
// Transport implementations that support writing response metadata should have
// their ResponseWriters implement ResponseMetaWriter to facilitate this.
type ResponseMetaWriter interface {
	ResponseMeta() *ResponseMeta
}

// ResponseMeta is the low level response metadata representation. Transports
// that support writing response metadata with ResponseMetaWriter MUST inspect
// the final ResponseMeta before writing the response body.
//
// ResponseWriters should expose this struct using the ResponseMetaWriter
// interface.
//
// There should be one per request.
type ResponseMeta struct {
	// ID of the response as chosen by the client. This MAY be a trace ID or
	// UUID.
	//
	// If the corresponding transport.Request struct has this field set, this
	// field MUST have the same value.
	ID string

	// Host is the name of the server responding with this reponse.
	//
	// It MAY be set by a an environment-aware middleware.
	Host string

	// Environment is the name of the host environment that the request was
	// issued from. eg "staging", "production"
	//
	// It MAY be set by a an environment-aware middleware.
	Environment string

	// Service is the name of the responding service.
	Service string

	Headers          Headers
	ApplicationError bool
}

// AddHeaders is a convenience function for appending to existing Headers.
func (meta *ResponseMeta) AddHeaders(headers Headers) {
	for k, v := range headers.OriginalItems() {
		meta.Headers = meta.Headers.With(k, v)
	}
}
