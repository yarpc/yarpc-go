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

import "strings"

// Headers represents YARPC headers.
//
// Header keys are case insensitive and duplicates are disallowed.
type Headers map[string]string

// NewHeaders builds a new Headers map from the given key-value pair.
//
// Use this instead of Headers(m).
func NewHeaders(m map[string]string) Headers {
	if m == nil {
		return nil
	}
	headers := make(Headers, len(m))
	for k, v := range m {
		headers.Set(k, v)
	}
	return headers
}

// Set sets the given header key to the given value.
//
// Note that empty strings are not valid header values. If the value is empty,
// the header will be deleted.
func (h Headers) Set(k, v string) {
	if v == "" {
		h.Del(k)
		return
	}

	h[strings.ToLower(k)] = v
}

// Get the header with the given name. Returns an empty string if a match is
// not found.
func (h Headers) Get(k string) string {
	if v, ok := h[strings.ToLower(k)]; ok {
		return v
	}
	return ""
}

// Del deletes the header with the given name.
func (h Headers) Del(k string) {
	delete(h, strings.ToLower(k))
}
