// Copyright (c) 2021 Uber Technologies, Inc.
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
	"log"
	"net/http"
	"strings"

	"go.uber.org/yarpc/api/transport"
)

// headerConverter converts HTTP headers to and from transport headers.
type headerMapper struct{ Prefix string }

var (
	applicationHeaders = headerMapper{ApplicationHeaderPrefix}
)

// toHTTPHeaders converts application headers into transport headers.
//
// Headers are read from 'from' and written to 'to'. The final header collection
// is returned.
//
// If 'to' is nil, a new map will be assigned.
func (hm headerMapper) ToHTTPHeaders(from transport.Headers, to http.Header) http.Header {
	if to == nil {
		to = make(http.Header, from.Len())
	}
	for k, v := range from.Items() {
		to.Add(hm.Prefix+k, v)
	}
	return to
}

// fromHTTPHeaders converts HTTP headers to application headers.
//
// Headers are read from 'from' and written to 'to'. The final header collection
// is returned.
//
// If 'to' is nil, a new map will be assigned.
func (hm headerMapper) FromHTTPHeaders(from http.Header, to transport.Headers) transport.Headers {
	prefixLen := len(hm.Prefix)
	for k := range from {
		if strings.HasPrefix(k, hm.Prefix) {
			key := k[prefixLen:]
			to = to.With(key, from.Get(k))
		}
		// Note: undefined behavior for multiple occurrences of the same header
	}
	return to
}

func (hm headerMapper) deleteHTTP2PseudoHeadersIfNeeded(from transport.Headers) transport.Headers {
	// deleting all http2 pseudo-header fields
	// RFC https://tools.ietf.org/html/rfc7540#section-8.1.2.3 does not mention
	// what to do with those headers though.
	// :method -> this can be removed, YARPC uses POST for all HTTP requests.
	// :path -> this can be removed, this is handled by YARPC with RPC-procedure.
	// :scheme -> this can be removed, scheme is defined in the URI (http or https).
	// :authority -> even if the RFC advises to copy :authority into host header, it is safe to remove it
	// here. Host of the request is controlled through the YARPC outbound configuration.
	for _, k := range _http2PseudoHeaders {
		if _, ok := from.Get(k); ok {
			from.Del(k)
			log.Printf("WARN: HTTP2 pseudo-header %s was deleted", k)
		}
	}
	return from
}
