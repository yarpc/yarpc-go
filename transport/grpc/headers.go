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

package grpc

import (
	"strings"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc/metadata"
)

const (
	// CallerHeader is the header key for the name of the service sending the
	// request. This corresponds to the Request.Caller attribute.
	// This header is required.
	CallerHeader = "rpc-caller"
	// ServiceHeader is the header key for the name of the service to which
	// the request is being sent. This corresponds to the Request.Service attribute.
	// This header is also used in responses to ensure requests are processed by the
	// correct service.
	// This header is required.
	ServiceHeader = "rpc-service"
	// ShardKeyHeader is the header key for the shard key used by the destined service
	// to shard the request. This corresponds to the Request.ShardKey attribute.
	// This header is optional.
	ShardKeyHeader = "rpc-shard-key"
	// RoutingKeyHeader is the header key for the traffic group responsible for
	// handling the request. This corresponds to the Request.RoutingKey attribute.
	// This header is optional.
	RoutingKeyHeader = "rpc-routing-key"
	// RoutingDelegateHeader is the header key for a service that can proxy the
	// destined service. This corresponds to the Request.RoutingDelegate attribute.
	// This header is optional.
	RoutingDelegateHeader = "rpc-routing-delegate"
	// EncodingHeader is the header key for the encoding used for the request body.
	// This corresponds to the Request.Encoding attribute.
	// If this is not set, content-type will attempt to be read for the encoding per
	// the gRPC wire format http://www.grpc.io/docs/guides/wire.html
	// For example, a content-type of "application/grpc+proto" will be intepreted
	// as the proto encoding.
	// This header is required unless content-type is set properly.
	EncodingHeader = "rpc-encoding"
	// ErrorNameHeader is the header key for the error name.
	ErrorNameHeader = "rpc-error-name"
	// ApplicationErrorHeader is the header key that will contain a non-empty value
	// if there was an application error.
	ApplicationErrorHeader = "rpc-application-error"
	// Load header is used to send load reports back to the caller.
	LoadHeader = "rpc-load"

	// _applicationErrorNameHeader is the header for the name of the application
	// error.
	_applicationErrorNameHeader = "rpc-application-error-name"
	// _applicationErrorDetailsHeader is the header for the the application error
	// meta details string.
	_applicationErrorDetailsHeader = "rpc-application-error-details"

	// ApplicationErrorHeaderValue is the value that will be set for
	// ApplicationErrorHeader is there was an application error.
	//
	// The definition says any non-empty value is valid, however this is
	// the specific value that will be used for now.
	ApplicationErrorHeaderValue = "error"

	baseContentType   = "application/grpc"
	contentTypeHeader = "content-type"
)

// TODO: there are way too many repeat calls to strings.ToLower
// Note that these calls are done indirectly, primarily through
// transport.CanonicalizeHeaderKey

func isReserved(header string) bool {
	// Allow setting the load header by the application.
	// This is to be used by opt-in middlewares.
	if header == LoadHeader {
		return false
	}
	return strings.HasPrefix(strings.ToLower(header), "rpc-")
}

// transportRequestToMetadata will populate all reserved and application headers
// from the Request into a new MD.
func transportRequestToMetadata(request *transport.Request) (metadata.MD, error) {
	md := metadata.New(nil)
	if err := multierr.Combine(
		addToMetadata(md, CallerHeader, request.Caller),
		addToMetadata(md, ServiceHeader, request.Service),
		addToMetadata(md, ShardKeyHeader, request.ShardKey),
		addToMetadata(md, RoutingKeyHeader, request.RoutingKey),
		addToMetadata(md, RoutingDelegateHeader, request.RoutingDelegate),
		addToMetadata(md, EncodingHeader, string(request.Encoding)),
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
			return nil, yarpcerrors.InvalidArgumentErrorf("header has more than one value: %s", header)
		}
		header = transport.CanonicalizeHeaderKey(header)
		switch header {
		case CallerHeader:
			request.Caller = value
		case ServiceHeader:
			request.Service = value
		case ShardKeyHeader:
			request.ShardKey = value
		case RoutingKeyHeader:
			request.RoutingKey = value
		case RoutingDelegateHeader:
			request.RoutingDelegate = value
		case EncodingHeader:
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

func metadataToApplicationErrorMeta(responseMD metadata.MD) *transport.ApplicationErrorMeta {
	if responseMD == nil {
		return nil
	}

	var details, name string
	if header := responseMD[_applicationErrorDetailsHeader]; len(header) == 1 {
		details = header[0]
	}
	if header := responseMD[_applicationErrorNameHeader]; len(header) == 1 {
		name = header[0]
	}

	return &transport.ApplicationErrorMeta{
		Details: details,
		Name:    name,
		// ignore Code, this should be derived from the error since codes are
		// natively supported in gRPC and YARPC
		Code: nil,
	}
}

// addApplicationHeaders adds the headers to md.
func addApplicationHeaders(md metadata.MD, headers transport.Headers) error {
	for header, value := range headers.Items() {
		header = transport.CanonicalizeHeaderKey(header)
		if isReserved(header) {
			return yarpcerrors.InvalidArgumentErrorf("cannot use reserved header in application headers: %s", header)
		}
		if err := addToMetadata(md, header, value); err != nil {
			return err
		}
	}
	return nil
}

// getApplicationHeaders returns the headers from md without any reserved headers.
func getApplicationHeaders(md metadata.MD) (transport.Headers, error) {
	if len(md) == 0 {
		return transport.Headers{}, nil
	}
	headers := transport.NewHeadersWithCapacity(md.Len())
	for header, values := range md {
		header = transport.CanonicalizeHeaderKey(header)
		if isReserved(header) {
			continue
		}
		var value string
		switch len(values) {
		case 0:
			continue
		case 1:
			value = values[0]
		default:
			return headers, yarpcerrors.InvalidArgumentErrorf("header has more than one value: %s", header)
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
		return yarpcerrors.InvalidArgumentErrorf("duplicate key: %s", key)
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

type mdReadWriter metadata.MD

// ForeachKey implements opentracing.TextMapReader.
func (md mdReadWriter) ForeachKey(handler func(string, string) error) error {
	for key, values := range md {
		for _, value := range values {
			if err := handler(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set implements opentracing.TextMapWriter.
func (md mdReadWriter) Set(key string, value string) {
	key = strings.ToLower(key)
	md[key] = []string{value}
}
