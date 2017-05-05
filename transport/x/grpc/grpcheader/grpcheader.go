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

// Package grpcheader provides the headers for gRPC.
package grpcheader

import "strings"

// these are the same as in transport/http but lowercase
// http2 does all lowercase headers and this should be explicit

const (
	// CallerHeader is the header key for the caller.
	CallerHeader = "rpc-caller"
	// ServiceHeader is the header key for the service.
	ServiceHeader = "rpc-service"
	// ShardKeyHeader is the header key for the shard key.
	ShardKeyHeader = "rpc-shard-key"
	// RoutingKeyHeader is the header key for the routing key.
	RoutingKeyHeader = "rpc-routing-key"
	// RoutingDelegateHeader is the header key for the routing delegate.
	RoutingDelegateHeader = "rpc-routing-delegate"
	// EncodingHeader is the header key for the encoding.
	// This will be removed when we get encoding propagated using content-type.
	EncodingHeader = "rpc-encoding"
)

var (
	reservedHeaders = map[string]bool{
		CallerHeader:          true,
		ServiceHeader:         true,
		ShardKeyHeader:        true,
		RoutingKeyHeader:      true,
		RoutingDelegateHeader: true,
		EncodingHeader:        true,
	}
)

// IsReserved returns true if the header is reserved.
func IsReserved(header string) bool {
	_, ok := reservedHeaders[strings.ToLower(header)]
	return ok
}
