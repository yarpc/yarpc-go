// Copyright (c) 2017 Uber Technologies, Inc.
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

package grpc

import (
	"fmt"
	"strings"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/x/grpc/grpcheader"
	"google.golang.org/grpc/metadata"
)

const (
	baseContentType   = "application/grpc"
	contentTypeHeader = "content-type"
)

// TODO: there are way too many repeat calls to strings.ToLower
// Note that these calls are done indirectly, primarily through
// transport.CanonicalizeHeaderKey and grpcheader.IsReserved

// transportRequestToMetadata will populate all reserved and application headers
// from the Request into a new MD.
func transportRequestToMetadata(request *transport.Request) (metadata.MD, error) {
	md := metadata.New(nil)
	if err := multierr.Combine(
		addToMetadata(md, grpcheader.CallerHeader, request.Caller),
		addToMetadata(md, grpcheader.ServiceHeader, request.Service),
		addToMetadata(md, grpcheader.ShardKeyHeader, request.ShardKey),
		addToMetadata(md, grpcheader.RoutingKeyHeader, request.RoutingKey),
		addToMetadata(md, grpcheader.RoutingDelegateHeader, request.RoutingDelegate),
		addToMetadata(md, grpcheader.EncodingHeader, string(request.Encoding)),
	); err != nil {
		return md, err
	}
	return md, addApplicationHeaders(md, request.Headers)
}

// metadataToTransportRequest will populate the Request with all reserved and application
// headers into a new Request, only not setting the Body field.
func metadataToTransportRequest(md metadata.MD) (*transport.Request, error) {
	request := &transport.Request{
		Headers: transport.NewHeadersWithCapacity(md.Len()),
	}
	for header, values := range md {
		var value string
		switch len(values) {
		case 0:
			continue
		case 1:
			value = values[0]
		default:
			return nil, fmt.Errorf("header has more than one value: %s", header)
		}
		header = transport.CanonicalizeHeaderKey(header)
		switch header {
		case grpcheader.CallerHeader:
			request.Caller = value
		case grpcheader.ServiceHeader:
			request.Service = value
		case grpcheader.ShardKeyHeader:
			request.ShardKey = value
		case grpcheader.RoutingKeyHeader:
			request.RoutingKey = value
		case grpcheader.RoutingDelegateHeader:
			request.RoutingDelegate = value
		case grpcheader.EncodingHeader:
			request.Encoding = transport.Encoding(value)
		case contentTypeHeader:
			// if request.Encoding was set, do not parse content-type
			// this results in EncodingHeader overriding content-type
			if request.Encoding == "" {
				request.Encoding = transport.Encoding(getContentSubtype(value))
			}
		default:
			request.Headers = request.Headers.With(header, value)
		}
	}
	return request, nil
}

// addApplicationHeaders adds the headers to md.
func addApplicationHeaders(md metadata.MD, headers transport.Headers) error {
	for header, value := range headers.Items() {
		header = transport.CanonicalizeHeaderKey(header)
		if grpcheader.IsReserved(header) {
			return fmt.Errorf("cannot use reserved header in application headers: %s", header)
		}
		if err := addToMetadata(md, header, value); err != nil {
			return err
		}
	}
	return nil
}

// getApplicationHeaders returns the headers from md without any reserved headers.
func getApplicationHeaders(md metadata.MD) (transport.Headers, error) {
	headers := transport.NewHeadersWithCapacity(md.Len())
	for header, values := range md {
		header = transport.CanonicalizeHeaderKey(header)
		if grpcheader.IsReserved(header) {
			continue
		}
		var value string
		switch len(values) {
		case 0:
			continue
		case 1:
			value = values[0]
		default:
			return headers, fmt.Errorf("header has more than one value: %s", header)
		}
		headers = headers.With(header, value)
	}
	return headers, nil
}

// add to md
// return error if key already in md
func addToMetadata(md metadata.MD, key string, value string) error {
	if value == "" {
		return nil
	}
	if _, ok := md[key]; ok {
		return fmt.Errorf("duplicate key: %s", key)
	}
	md[key] = []string{value}
	return nil
}

// getContentSubtype attempts to get the content subtype.
// returns "" if no content subtype can be parsed.
func getContentSubtype(contentType string) string {
	if !strings.HasPrefix(contentType, baseContentType) || len(contentType) == len(baseContentType) {
		return ""
	}
	switch contentType[len(baseContentType)] {
	case '+', ';':
		return contentType[len(baseContentType)+1:]
	default:
		return ""
	}
}
