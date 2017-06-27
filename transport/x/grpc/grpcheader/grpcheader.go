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

// Package grpcheader provides the headers functionality for gRPC.
//
// This is separated from the main grpc package so that it can be used
// with grpc-go clients without needing to import so many yarpc packages,
// as well as for future cross-package yarpc work.
package grpcheader

import "strings"

// these are the same as in transport/http but lowercase
// http2 does all lowercase headers and this should be explicit

const (
	// CallerHeader is the header key for the name of the service sending the
	// request. This corresponds to the Request.Caller attribute.
	// This header is required.
	CallerHeader = "rpc-caller"
	// ServiceHeader is the header key for the name of the service to which
	// the request is being sent. This corresponds to the Request.Service attribute.
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
)

var (
	reservedHeaders = map[string]bool{
		CallerHeader:          true,
		ServiceHeader:         true,
		ShardKeyHeader:        true,
		RoutingKeyHeader:      true,
		RoutingDelegateHeader: true,
		EncodingHeader:        true,
		ErrorNameHeader:       true,
	}
)

// IsReserved returns true if the header is reserved.
func IsReserved(header string) bool {
	_, ok := reservedHeaders[strings.ToLower(header)]
	return ok
}
