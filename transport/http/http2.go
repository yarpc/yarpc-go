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
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"go.uber.org/yarpc/api/transport"
)

const (
	http2SchemePseudoHeader    = ":scheme"
	http2MethodPseudoHeader    = ":method"
	http2AuthorityPseudoHeader = ":authority"
	http2PathPseudoHeader      = ":path"
)

var (
	errMalformedHTTP2ConnectRequestExtraScheme    = malformedHTTP2ConnectRequestError(http2SchemePseudoHeader, false)
	errMalformedHTTP2ConnectRequestExtraPath      = malformedHTTP2ConnectRequestError(http2PathPseudoHeader, false)
	errMalformedHTTP2ConnectRequestExtraAuthority = malformedHTTP2ConnectRequestError(http2AuthorityPseudoHeader, true)
)

func malformedHTTP2ConnectRequestError(h string, shouldContain bool) error {
	base := "HTTP2 CONNECT request "
	if shouldContain {
		base += fmt.Sprintf("must contain pseudo header %q", h)
	} else {
		base += fmt.Sprintf("must not contain pseudo header %q", h)
	}
	return errors.New(base)
}

// take a HTTP/2 request with CONNECT method, mostly with grpc implementation request
// and convert to a HTTP/1.X equivalent request.
// All comments below are quotes from RFC7540:
// https://tools.ietf.org/html/rfc7540#section-8.3
func fromHTTP2ConnectRequest(treq *transport.Request) (*http.Request, error) {
	// The ":scheme" and ":path" pseudo-header fields MUST be omitted.
	if _, ok := treq.Headers.Get(http2SchemePseudoHeader); ok {
		return nil, errMalformedHTTP2ConnectRequestExtraScheme
	}
	if _, ok := treq.Headers.Get(http2PathPseudoHeader); ok {
		return nil, errMalformedHTTP2ConnectRequestExtraPath
	}

	// The ":authority" pseudo-header field contains the host and port to
	// connect to
	if a, ok := treq.Headers.Get(http2AuthorityPseudoHeader); ok {
		url := &url.URL{Host: a}
		return http.NewRequest(http.MethodConnect, url.String(), nil)
	}
	return nil, errMalformedHTTP2ConnectRequestExtraAuthority
}
