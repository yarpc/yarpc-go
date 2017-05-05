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

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

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

// ContextWrapper wraps a context for grpc-go with the required headers for yarpc.
//
// This is a convienence object when using grpc-go clients.
type ContextWrapper struct {
	md metadata.MD
}

// NewContextWrapper returns a new ContextWrapper.
//
// The only fields that a grpc-go client needs to set are caller and service.
func NewContextWrapper() *ContextWrapper {
	return &ContextWrapper{metadata.New(nil)}
}

// Wrap wraps the given context with the headers.
func (c *ContextWrapper) Wrap(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, c.md)
}

// WithCaller returns a new ContextWrapper with the given caller.
func (c *ContextWrapper) WithCaller(caller string) *ContextWrapper {
	return c.copyAndAdd(CallerHeader, caller)
}

// WithService returns a new ContextWrapper with the given service.
func (c *ContextWrapper) WithService(service string) *ContextWrapper {
	return c.copyAndAdd(ServiceHeader, service)
}

// WithShardKey returns a new ContextWrapper with the given shard key.
func (c *ContextWrapper) WithShardKey(shardKey string) *ContextWrapper {
	return c.copyAndAdd(ShardKeyHeader, shardKey)
}

// WithRoutingKey returns a new ContextWrapper with the given routing key.
func (c *ContextWrapper) WithRoutingKey(routingKey string) *ContextWrapper {
	return c.copyAndAdd(RoutingKeyHeader, routingKey)
}

// WithRoutingDelegate returns a new ContextWrapper with the given routing delegate.
func (c *ContextWrapper) WithRoutingDelegate(routingDelegate string) *ContextWrapper {
	return c.copyAndAdd(RoutingDelegateHeader, routingDelegate)
}

func (c *ContextWrapper) copyAndAdd(key string, value string) *ContextWrapper {
	md := c.md
	if md == nil {
		md = metadata.New(nil)
	}
	md[key] = []string{value}
	return &ContextWrapper{md}
}
