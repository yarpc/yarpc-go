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

package http

import (
	"net/http"

	"github.com/yarpc/yarpc-go/transport"
)

// toHTTPHeader converts transport headers into HTTP headers.
//
// Headers are read from 'from' and written to 'to'. The final header
// collection is returned.
//
// If 'to' is nil, a new map will be assigned.
func toHTTPHeader(from transport.Headers, to http.Header) http.Header {
	if to == nil {
		to = make(http.Header)
	}
	for k, v := range from {
		to.Add(k, v)
	}
	return to
}

// fromHTTPHeader converts HTTP headers to transport headers.
//
// Headers are read from 'from' and written to 'to'. The final header
// collection is returned.
//
// If 'to' is nil, a new map will be assigned.
func fromHTTPHeader(from http.Header, to transport.Headers) transport.Headers {
	if to == nil {
		to = make(transport.Headers)
	}

	for k := range from {
		to.Set(k, from.Get(k))
		// undefined behavior for multiple occurrences of the same header
		// TODO figure out which headers we are actually allowing in here
		// TODO figure out header scheme for headers that are exposed like this
	}
	return to
}
