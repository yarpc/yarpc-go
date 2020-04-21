// Copyright (c) 2020 Uber Technologies, Inc.
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

package inboundcall

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// Metadata holds metadata for an incoming request. This includes metadata
// about the inbound request as well as response metadata.
//
// This drives the behavior of yarpc.Call and encoding.Call.
type Metadata interface {
	WriteResponseHeader(k, v string) error
	Caller() string
	Service() string
	Transport() string
	Procedure() string
	Encoding() transport.Encoding
	Headers() transport.Headers
	ShardKey() string
	RoutingKey() string
	RoutingDelegate() string
}

type metadataKey struct{} // context key for Metadata

// WithMetadata places the provided metadata on the context.
func WithMetadata(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, md)
}

// GetMetadata retrieves inbound call metadata from a context.
func GetMetadata(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(metadataKey{}).(Metadata)
	return md, ok
}
