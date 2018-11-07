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

package yarpcgrpc

import (
	"go.uber.org/multierr"
	"go.uber.org/yarpc/v2"
	"google.golang.org/grpc/metadata"
)

type metadataWriter struct {
	md        metadata.MD
	headerErr error
}

func newMetadataWriter() *metadataWriter {
	return &metadataWriter{
		md: metadata.New(nil),
	}
}

func (r *metadataWriter) SetResponse(res *yarpc.Response) {
	if res == nil {
		return
	}

	r.headerErr = multierr.Combine(r.headerErr, addApplicationHeaders(r.md, res.Headers))

	if res.ApplicationError != nil {
		r.AddSystemHeader(ApplicationErrorHeader, ApplicationErrorHeaderValue)
	}
}

func (r *metadataWriter) AddSystemHeader(key string, value string) {
	r.headerErr = multierr.Combine(r.headerErr, addToMetadata(r.md, key, value))
}

func (r *metadataWriter) MD() metadata.MD {
	return r.md
}
