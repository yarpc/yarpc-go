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
	"sort"

	"github.com/yarpc/yarpc-go/internal"
)

// CanonicalizeHeaderKey canonicalizes the given header key for storage into
// the Headers map.
func CanonicalizeHeaderKey(k string) string {
	return internal.CanonicalizeHeaderKey(k)
}

// Headers defines application headers which will be sent over the wire to the
// recipient of an RPC call.
type Headers internal.Headers

// NewHeaders builds a new Headers object.
func NewHeaders() Headers {
	return Headers{}
}

// With returns a Headers object with the given key-value pair added to it.
// If a header with the same name already exists, it will be overwritten.
//
// This API is similar to Go's append function. The returned Headers object
// MAY not point to the same underlying data store, so the returned value MUST
// always be used in place of the original object.
//
// 	headers = headers.With("foo", "bar")
//
// This call may be chained to set multiple headers consecutively.
//
// 	headers = headers.With("foo", "bar").With("baz", "qux")
//
// Again, note that the returned Headers object MAY point to a new object. It
// MAY also mutate the original object instead.
//
// 	h1 = NewHeaders().With("foo", "bar")
// 	h2 = h1.With("baz", "qux")
// 	h1.Get("baz")  // this MAY return "qux"
//
func (h Headers) With(k, v string) Headers {
	return Headers(internal.Headers(h).With(k, v))
}

// Get retrieves the value associated with the given header key, and a
// boolean indicating whether the key actually existed in the header map.
func (h Headers) Get(k string) (string, bool) {
	return internal.Headers(h).Get(k)
}

// Del deletes the given header key from the map.
func (h Headers) Del(k string) {
	internal.Headers(h).Del(k)
}

// Len returns the number of headers defined on this object.
func (h Headers) Len() int {
	return internal.Headers(h).Len()
}

// Keys returns a list of header keys defined on this Headers object.
//
// All items in the list will be normalized using CanonicalizeHeaderKey.
func (h Headers) Keys() []string {
	ih := internal.Headers(h)
	keys := make([]string, 0, ih.Len())
	for k := range ih.Items() {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
