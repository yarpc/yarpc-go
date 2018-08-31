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

package grpc

import (
	"bytes"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type responseWriter struct {
	buffer    *bytes.Buffer
	md        metadata.MD
	resMeta   *transport.ResponseMeta
	headerErr error
}

func newResponseWriter(treq *transport.Request) *responseWriter {
	return &responseWriter{
		md: metadata.New(nil),
		resMeta: &transport.ResponseMeta{
			ID:      treq.ID,
			Service: treq.Service,
		},
	}
}

func (r *responseWriter) Write(p []byte) (int, error) {
	if r.buffer == nil {
		r.buffer = bytes.NewBuffer(make([]byte, 0, len(p)))
	}
	return r.buffer.Write(p)
}

func (r *responseWriter) AddHeaders(headers transport.Headers) {
	r.resMeta.AddHeaders(headers)
}

func (r *responseWriter) SetApplicationError() {
	r.resMeta.ApplicationError = true
}

func (r *responseWriter) AddSystemHeader(key string, value string) {
	r.headerErr = multierr.Combine(r.headerErr, addToMetadata(r.md, key, value))
}

func (r *responseWriter) Bytes() []byte {
	if r.buffer == nil {
		return nil
	}
	return r.buffer.Bytes()
}

func (r *responseWriter) ResponseMeta() *transport.ResponseMeta {
	return r.resMeta
}

func (r *responseWriter) setResponseMeta() {
	if r.resMeta.ID != "" {
		r.AddSystemHeader(IDHeader, r.resMeta.ID)
	}
	if r.resMeta.Host != "" {
		r.AddSystemHeader(HostHeader, r.resMeta.Host)
	}
	if r.resMeta.Environment != "" {
		r.AddSystemHeader(EnvironmentHeader, r.resMeta.Environment)
	}
	if r.resMeta.Service != "" {
		r.AddSystemHeader(ServiceHeader, r.resMeta.Service)
	}

	r.headerErr = multierr.Combine(r.headerErr, addApplicationHeaders(r.md, r.resMeta.Headers))
	if r.resMeta.ApplicationError {
		r.AddSystemHeader(ApplicationErrorHeader, ApplicationErrorHeaderValue)
	}
}

// Close should only be called after a successful send on the server stream
func (r *responseWriter) Close(serverStream grpc.ServerStream) {
	r.setResponseMeta()
	serverStream.SetTrailer(r.md)
	r.buffer = nil
}
