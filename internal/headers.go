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

package internal

import "strings"

var emptyMap = map[string]string{}

// Headers provides the implementation of both, yarpc.Headers and
// transport.Headers.
//
// Keys in the map are canonicalized using CanonicalizeHeaderKey.
type Headers struct {
	items map[string]string
}

// NewHeadersWithCapacity allocates a new Headers object. capacity specifies
// the initial capacity of the headers map. A capacity of zero or less is
// ignored.
func NewHeadersWithCapacity(capacity int) Headers {
	if capacity <= 0 {
		return Headers{}
	}
	return Headers{items: make(map[string]string, capacity)}
}

// With returns a Headers object with the given key-value pair added to it.
//
// Returns a Headers object possibly backed by a different data store.
func (h Headers) With(k, v string) Headers {
	if h.items == nil {
		h.items = make(map[string]string)
	}
	h.items[CanonicalizeHeaderKey(k)] = v
	return h
}

// Items returns the underlying map for this Headers object. The returned map
// MUST NOT be mutated. Doing so will result in undefined behavior.
//
// This ALWAYS returns a non-nil result.
func (h Headers) Items() map[string]string {
	if h.items == nil {
		return emptyMap
	}
	return h.items
}

// Get retrieves the value associated with the given key.
func (h Headers) Get(k string) (string, bool) {
	v, ok := h.items[CanonicalizeHeaderKey(k)]
	return v, ok
}

// Del deletes the header with the given name.
func (h Headers) Del(k string) {
	delete(h.items, CanonicalizeHeaderKey(k))
}

// Len returns the number of headers defined on this object.
func (h Headers) Len() int {
	return len(h.items)
}

// CanonicalizeHeaderKey canonicalizes the given header key for storage into
// the Headers map.
func CanonicalizeHeaderKey(k string) string {
	// TODO: Deal with unsupported header keys (anything that's not a valid HTTP
	// header key).
	return strings.ToLower(k)
}
