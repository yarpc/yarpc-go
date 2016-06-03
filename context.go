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

package yarpc

import (
	"github.com/yarpc/yarpc-go/internal/reqcontext"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// WithHeaders returns a copy of the given context with the given context headers.
//
// If the context already has headers on it, the given headers will be appended
// to the context, overwriting any existing headers with conflicting names.
//
// Note that context headers are propagated across multiple hops; all downstream
// requests will be able to see these. Use the Headers field in ReqMeta for
// single-hop headers.
func WithHeaders(ctx context.Context, headers transport.Headers) context.Context {
	return reqcontext.AddHeaders(ctx, headers)
}

// HeadersFromContext returns a copy of the headers stored on the given context.
//
// An empty headers map is returned if the context does not have any headers
// associated with it.
//
// Note: The returned headers map is a copy of the headers stored on the
// context. Modifications to it won't be propagated.
func HeadersFromContext(ctx context.Context) transport.Headers {
	hs := make(transport.Headers)
	if headers := reqcontext.GetHeaders(ctx); headers != nil {
		for k, v := range headers {
			hs.Set(k, v)
		}
	}
	return hs
}
