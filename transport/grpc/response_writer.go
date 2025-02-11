// Copyright (c) 2024 Uber Technologies, Inc.
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
	"google.golang.org/grpc/metadata"
)

var (
	_ transport.ExtendedResponseWriter     = (*responseWriter)(nil)
	_ transport.ApplicationErrorMetaSetter = (*responseWriter)(nil)
)

type responseWriter struct {
	buffer             *bytes.Buffer
	md                 metadata.MD
	headerErr          error
	isApplicationError bool
	appErrorMeta       *transport.ApplicationErrorMeta
	responseSize       int
}

func newResponseWriter() *responseWriter {
	return &responseWriter{}
}

func (r *responseWriter) Write(p []byte) (int, error) {
	if r.buffer == nil {
		// Response writer bytes must not be pooled since calls to SendMsg hold on
		// to the bytes after the the function returns.
		//
		// See https://github.com/yarpc/yarpc-go/pull/1738 for details.
		r.buffer = bytes.NewBuffer(make([]byte, 0, len(p)))
	}
	n, err := r.buffer.Write(p)
	r.responseSize += n
	return n, err
}

func (r *responseWriter) ResponseSize() int {
	return r.responseSize
}

func (r *responseWriter) AddHeaders(headers transport.Headers) {
	if r.md == nil {
		r.md = metadata.New(nil)
	}
	r.headerErr = multierr.Combine(r.headerErr, addApplicationHeaders(r.md, headers))
}

func (r *responseWriter) SetApplicationError() {
	r.isApplicationError = true
	r.AddSystemHeader(ApplicationErrorHeader, ApplicationErrorHeaderValue)
}

func (r *responseWriter) SetApplicationErrorMeta(meta *transport.ApplicationErrorMeta) {
	if meta == nil {
		return
	}

	r.appErrorMeta = meta
	if meta.Name != "" {
		r.AddSystemHeader(_applicationErrorNameHeader, meta.Name)
	}
	if meta.Details != "" {
		r.AddSystemHeader(_applicationErrorDetailsHeader, meta.Details)
	}
}

func (r *responseWriter) IsApplicationError() bool {
	return r.isApplicationError
}

func (r *responseWriter) ApplicationErrorMeta() *transport.ApplicationErrorMeta {
	return r.appErrorMeta
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
	r.buffer = nil
}
