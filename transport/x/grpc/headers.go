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

	"go.uber.org/yarpc/api/transport"

	"go.uber.org/multierr"
	"google.golang.org/grpc/metadata"
)

const (
	globalHeaderPrefix      = "yarpc-grpc-"
	reservedHeaderPrefix    = globalHeaderPrefix + "reserved-"
	applicationHeaderPrefix = globalHeaderPrefix + "app-"
	callerHeader            = reservedHeaderPrefix + "caller"
	encodingHeader          = reservedHeaderPrefix + "encoding"
	serviceHeader           = reservedHeaderPrefix + "service"
)

// transportRequestToMetadata will populate all reserved and application headers
// from the Request into a new MD.
func transportRequestToMetadata(request *transport.Request) (metadata.MD, error) {
	md := metadata.New(nil)
	err := addToMetadata(md, callerHeader, request.Caller)
	err = multierr.Append(err, addToMetadata(md, encodingHeader, string(request.Encoding)))
	err = multierr.Append(err, addToMetadata(md, serviceHeader, request.Service))
	err = multierr.Append(err, addApplicationHeaders(md, request.Headers))
	return md, err
}

// populateTransportRequest will populate the Request with all reserved and application
// headers from the MD.
func populateTransportRequest(md metadata.MD, request *transport.Request) error {
	caller, callerErr := getFromMetadata(md, callerHeader)
	request.Caller = caller
	encoding, encodingErr := getFromMetadata(md, encodingHeader)
	request.Encoding = transport.Encoding(encoding)
	service, serviceErr := getFromMetadata(md, serviceHeader)
	request.Service = service
	headers, headersErr := getApplicationHeaders(md)
	request.Headers = headers
	return multierr.Combine(callerErr, encodingErr, serviceErr, headersErr)
}

// add headers into md as application headers
// return error if md already has a key defined that is defined in headers
func addApplicationHeaders(md metadata.MD, headers transport.Headers) error {
	for key, value := range headers.Items() {
		if err := addToMetadata(md, applicationHeaderPrefix+key, value); err != nil {
			return err
		}
	}
	return nil
}

// get application headers from md
// return error if any application error has more than one value
func getApplicationHeaders(md metadata.MD) (transport.Headers, error) {
	headers := transport.NewHeadersWithCapacity(md.Len())
	for mdKey := range md {
		key := strings.TrimPrefix(mdKey, applicationHeaderPrefix)
		// not an application header
		if key == mdKey {
			continue
		}
		value, err := getFromMetadata(md, mdKey)
		if err != nil {
			return headers, err
		}
		headers.With(key, value)
	}
	return headers, nil
}

// add to md
// return error if key already in md
func addToMetadata(md metadata.MD, key string, value string) error {
	key = transport.CanonicalizeHeaderKey(key)
	if _, ok := md[key]; ok {
		return fmt.Errorf("duplicate key: %s", key)
	}
	md[key] = []string{value}
	return nil
}

// get from md
// return "" if not present
// return error if more than one value
func getFromMetadata(md metadata.MD, key string) (string, error) {
	values, ok := md[key]
	if !ok {
		return "", nil
	}
	switch len(values) {
	case 0:
		return "", nil
	case 1:
		return values[0], nil
	default:
		return "", fmt.Errorf("key has more than one value: %s", key)
	}
}
