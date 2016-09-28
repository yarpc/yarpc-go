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

package baggage

import (
	"github.com/yarpc/yarpc-go/transport"

	"context"
)

type baggageKey struct{}

// NewContext returns a copy of the context with the given key-value pair
// attached to it.
//
// The parent context's baggage is left intact.
func NewContext(ctx context.Context, key, value string) context.Context {
	hs := FromContext(ctx)
	baggage := transport.NewHeadersWithCapacity(hs.Len() + 1)

	// Copy baggage already attached to the context.
	if hs.Len() > 0 {
		for k, v := range hs.Items() {
			baggage = baggage.With(k, v)
		}
	}

	baggage = baggage.With(key, value)
	return context.WithValue(ctx, baggageKey{}, baggage)
}

// Get returns the baggage value attached to the context with the given key.
func Get(ctx context.Context, key string) (value string, ok bool) {
	return FromContext(ctx).Get(key)
}

// NewContextWithHeaders is similar to NewContext except it attaches all
// elements in the given headers map to the context.
func NewContextWithHeaders(ctx context.Context, headers map[string]string) context.Context {
	hs := FromContext(ctx)

	// This API is for use by Inbound implementations only. It's significantly
	// cheaper than calling NewContext in a loop.
	baggage := transport.NewHeadersWithCapacity(hs.Len() + len(headers))

	// Copy baggage already attached to the context.
	if hs.Len() > 0 {
		for k, v := range hs.Items() {
			baggage = baggage.With(k, v)
		}
	}

	for k, v := range headers {
		baggage = baggage.With(k, v)
	}

	return context.WithValue(ctx, baggageKey{}, baggage)
}

// FromContext returns all baggage attached to the given context as a header
// map, or nil if no baggage is attached to it.
func FromContext(ctx context.Context) transport.Headers {
	hs, ok := ctx.Value(baggageKey{}).(transport.Headers)
	if !ok {
		return transport.Headers{}
	}
	return hs
}
