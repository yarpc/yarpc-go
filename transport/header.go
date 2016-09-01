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

package transport

import "github.com/yarpc/yarpc-go/internal"

// CanonicalizeHeaderKey canonicalizes the given header key for storage into
// the Headers map.
func CanonicalizeHeaderKey(k string) string {
	return internal.CanonicalizeHeaderKey(k)
}

// Headers is the transport-level representation of application headers.
//
// Keys in the map MUST be canonicalized with CanonicalizeHeaderKey.
//
// You probably want to look at yarpc.Headers instead.
type Headers internal.Headers

// NewHeaders builds a new Headers object.
func NewHeaders() Headers {
	return Headers{}
}

// NewHeadersWithCapacity builds a new Headers object with the given capacity.
func NewHeadersWithCapacity(capacity int) Headers {
	return Headers(internal.NewHeadersWithCapacity(capacity))
}

// With returns a Headers object with the given key-value pair added to it.
// The returned object MAY not point to the same Headers underlying data store
// as the original Headers so the returned Headers MUST always be used instead
// of the original object.
//
// 	headers = headers.With("foo", "bar").With("baz", "qux")
func (h Headers) With(k, v string) Headers {
	return Headers(internal.Headers(h).With(k, v))
}

// Del deletes the header with the given name from the Headers map.
//
// This is a no-op if the key does not exist.
func (h Headers) Del(k string) {
	internal.Headers(h).Del(k)
}

// Get retrieves the value associated with the given header name.
func (h Headers) Get(k string) (string, bool) {
	return internal.Headers(h).Get(k)
}

// Len returns the number of headers defined on this object.
func (h Headers) Len() int {
	return internal.Headers(h).Len()
}

// Items returns the underlying map for this Headers map.
//
// Keys in the map are normalized using CanonicalizeHeaderKey.
//
// The returned map MUST NOT be mutated.
func (h Headers) Items() map[string]string {
	return internal.Headers(h).Items()
}

// HeadersFromMap builds a new Headers object from the given map of header
// key-value pairs.
func HeadersFromMap(m map[string]string) Headers {
	if len(m) == 0 {
		return Headers{}
	}
	headers := NewHeadersWithCapacity(len(m))
	for k, v := range m {
		headers = headers.With(k, v)
	}
	return headers
}
