// Copyright (c) 2025 Uber Technologies, Inc.
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

// Package grpcctx contains helper functionality for testing with grpc-go.
package grpcctx

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// ContextWrapper wraps a context for grpc-go with the required headers for yarpc.
//
// This is a convenience object for use when using grpc-go clients. You must set
// certain yarpc-specific headers when using native grpc-go clients calling into yarpc
// servers, and this object makes that simpler.
type ContextWrapper struct {
	md metadata.MD
}

// NewContextWrapper returns a new ContextWrapper.
//
// The only fields that a grpc-go client needs to set are caller and service.
// Encoding is also required if content-type is not set properly.
// See the documention on EncodingHeader.
func NewContextWrapper() *ContextWrapper {
	return &ContextWrapper{metadata.New(nil)}
}

// Wrap wraps the given context with the headers.
func (c *ContextWrapper) Wrap(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, c.md)
}

// WithCaller returns a new ContextWrapper with the given caller.
func (c *ContextWrapper) WithCaller(caller string) *ContextWrapper {
	return c.copyAndAdd("rpc-caller", caller)
}

// WithService returns a new ContextWrapper with the given service.
func (c *ContextWrapper) WithService(service string) *ContextWrapper {
	return c.copyAndAdd("rpc-service", service)
}

// WithShardKey returns a new ContextWrapper with the given shard key.
func (c *ContextWrapper) WithShardKey(shardKey string) *ContextWrapper {
	return c.copyAndAdd("rpc-shard-key", shardKey)
}

// WithRoutingKey returns a new ContextWrapper with the given routing key.
func (c *ContextWrapper) WithRoutingKey(routingKey string) *ContextWrapper {
	return c.copyAndAdd("rpc-routing-key", routingKey)
}

// WithRoutingDelegate returns a new ContextWrapper with the given routing delegate.
func (c *ContextWrapper) WithRoutingDelegate(routingDelegate string) *ContextWrapper {
	return c.copyAndAdd("rpc-routing-delegate", routingDelegate)
}

// WithEncoding returns a new ContextWrapper with the given encoding.
func (c *ContextWrapper) WithEncoding(encoding string) *ContextWrapper {
	return c.copyAndAdd("rpc-encoding", encoding)
}

// WithHeader returns a new ContextWrapper with the given header.
func (c *ContextWrapper) WithHeader(key, value string) *ContextWrapper {
	return c.copyAndAdd(key, value)
}

func (c *ContextWrapper) copyAndAdd(key, value string) *ContextWrapper {
	md := c.md
	if md == nil {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	md[key] = []string{value}
	return &ContextWrapper{md}
}
