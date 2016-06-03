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

package reqcontext

import (
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// AddHeaders adds headers to the given context. Existing headers will be merged
// with the new set, overwriting existing headers with conflicting names.
func AddHeaders(ctx context.Context, headers transport.Headers) context.Context {
	ctxHeaders := GetHeaders(ctx)
	if ctxHeaders == nil {
		ctxHeaders = make(transport.Headers, len(headers))
		ctx = context.WithValue(ctx, contextHeadersKey, ctxHeaders)
	}

	for k, v := range headers {
		ctxHeaders.Set(k, v)
	}
	return ctx
}

// GetHeaders returns the headers stored on the given context, or nil if the
// context does not have headers associated with it.
//
// Changes to the returned transport.Headers will be retained on the context.
func GetHeaders(ctx context.Context) transport.Headers {
	hs, ok := ctx.Value(contextHeadersKey).(transport.Headers)
	if ok {
		return hs
	}
	return nil
}
