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
	"github.com/yarpc/yarpc-go/internal/baggage"

	"golang.org/x/net/context"
)

// WithBaggage returns a copy of the context with the given baggage attached to
// it.
//
// Baggage is a set of key-value pairs that are sent to all downstream requests
// made with the returned context. Use this sparingly: all downstream services
// that receive baggage in a request will propagate it to all outbound calls
// made as a result of receiving that request.
//
// If baggage with the same key is already attached to the context, it will be
// overwritten in the new context. The parent context will always be left
// unchanged.
func WithBaggage(ctx context.Context, key, value string) context.Context {
	return baggage.NewContext(ctx, key, value)
}

// BaggageFromContext returns the baggage attached to the context with the given
// key name. False is returned if baggage with the given name was not attached
// to the context.
func BaggageFromContext(ctx context.Context, key string) (value string, ok bool) {
	return baggage.Get(ctx, key)
}
