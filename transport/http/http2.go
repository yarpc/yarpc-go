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
	// CONNECT requests
	errMalformedHTTP2ConnectRequestExtraScheme      = malformedHTTP2ConnectRequestError(http2SchemePseudoHeader, false)
	errMalformedHTTP2ConnectRequestExtraPath        = malformedHTTP2ConnectRequestError(http2PathPseudoHeader, false)
	errMalformedHTTP2ConnectRequestMissingAuthority = malformedHTTP2ConnectRequestError(http2AuthorityPseudoHeader, true)

	// non-CONNECT request
	errMalformedHTTP2NonConnectRequestMissingMethod = malformedHTTP2NonConnectRequestError(http2MethodPseudoHeader)
	errMalformedHTTP2NonConnectRequestMissingScheme = malformedHTTP2NonConnectRequestError(http2SchemePseudoHeader)
	errMalformedHTTP2NonConnectRequestMissingPath   = malformedHTTP2NonConnectRequestError(http2PathPseudoHeader)
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

func malformedHTTP2NonConnectRequestError(h string) error {
	return fmt.Errorf("HTTP2 non-CONNECT request must contain pseudo header %q", h)
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
	return nil, errMalformedHTTP2ConnectRequestMissingAuthority
}

// take a HTTP/2 request NOT with CONNECT method, mostly with grpc implementation request
// and convert to a HTTP/1.X equivalent request.
// All comments below are quotes from RFC7540:
// https://tools.ietf.org/html/rfc7540#section-8.1.2.3
func fromHTTP2NonConnectRequest(treq *transport.Request) (*http.Request, error) {
	// All HTTP/2 requests MUST include exactly one valid value for the
	// ":method", ":scheme", and ":path" pseudo-header fields,unless it is
	// a CONNECT request. An HTTP request that omits
	// mandatory pseudo-header fields is malformed
	method, ok := treq.Headers.Get(http2MethodPseudoHeader)
	if !ok {
		return nil, errMalformedHTTP2NonConnectRequestMissingMethod
	}
	scheme, ok := treq.Headers.Get(http2SchemePseudoHeader)
	if !ok {
		return nil, errMalformedHTTP2NonConnectRequestMissingScheme
	}
	path, ok := treq.Headers.Get(http2PathPseudoHeader)
	if !ok {
		return nil, errMalformedHTTP2NonConnectRequestMissingPath
	}

	url := &url.URL{Scheme: scheme, Path: path}
	hreq, err := http.NewRequest(method, url.String(), treq.Body)
	if err != nil {
		return nil, err
	}
	// An intermediary that converts an HTTP/2 request to HTTP/1.1 MUST
	// create a Host header field if one is not present in a request by
	// copying the value of the ":authority" pseudo-header field.
	if a, ok := treq.Headers.Get(http2AuthorityPseudoHeader); ok && hreq.Host == "" {
		hreq.Host = a
	}
	return hreq, err
}
