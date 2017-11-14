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

package grpc

import (
	"bytes"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"google.golang.org/grpc/metadata"
)

type responseWriter struct {
	buffer    *bytes.Buffer
	md        metadata.MD
	headerErr error
}

func newResponseWriter() *responseWriter {
	return &responseWriter{}
}

func (r *responseWriter) Write(p []byte) (int, error) {
	if r.buffer == nil {
		r.buffer = bufferpool.Get()
	}
	return r.buffer.Write(p)
}

func (r *responseWriter) AddHeaders(headers transport.Headers) {
	if r.md == nil {
		r.md = metadata.New(nil)
	}
	r.headerErr = multierr.Combine(r.headerErr, addApplicationHeaders(r.md, headers))
}

func (r *responseWriter) SetApplicationError() {
	r.AddSystemHeader(ApplicationErrorHeader, ApplicationErrorHeaderValue)
}

func (r *responseWriter) AddSystemHeader(key string, value string) {
	if r.md == nil {
		r.md = metadata.New(nil)
	}
	r.headerErr = multierr.Combine(r.headerErr, addToMetadata(r.md, key, value))
}

func (r *responseWriter) Bytes() []byte {
	if r.buffer == nil {
		return nil
	}
	return r.buffer.Bytes()
}

func (r *responseWriter) Close() {
	if r.buffer != nil {
		bufferpool.Put(r.buffer)
	}
	r.buffer = nil
}
