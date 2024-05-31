// Copyright (c) 2024 Uber Technologies, Inc.
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

package yarpctest

import (
	"context"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/inboundcall"
)

// Call specifies metadata for ContextWithCall.
type Call struct {
	Caller          string
	Service         string
	Transport       string
	Procedure       string
	Encoding        transport.Encoding
	Headers         map[string]string
	ShardKey        string
	RoutingKey      string
	RoutingDelegate string
	CallerProcedure string

	// If set, this map will be filled with response headers written to
	// yarpc.Call.
	ResponseHeaders map[string]string
}

// ContextWithCall builds a Context which will yield the provided request
// metadata when used with yarpc.CallFromContext.
//
//	ctx := yarpctest.ContextWithCall(ctx, &Call{..})
//	handler.GetValue(ctx, &Request{...})
func ContextWithCall(ctx context.Context, call *Call) context.Context {
	if call == nil {
		return ctx // no-op
	}

	return inboundcall.WithMetadata(ctx, callMetadata{call})
}

type callMetadata struct{ c *Call }

func (c callMetadata) WriteResponseHeader(k string, v string) error {
	if c.c.ResponseHeaders != nil {
		c.c.ResponseHeaders[k] = v
	}
	return nil
}

func (c callMetadata) Caller() string               { return c.c.Caller }
func (c callMetadata) Service() string              { return c.c.Service }
func (c callMetadata) Transport() string            { return c.c.Transport }
func (c callMetadata) Procedure() string            { return c.c.Procedure }
func (c callMetadata) Encoding() transport.Encoding { return c.c.Encoding }
func (c callMetadata) CallerProcedure() string      { return c.c.CallerProcedure }

func (c callMetadata) Headers() transport.Headers {
	return transport.HeadersFromMap(c.c.Headers)
}

func (c callMetadata) ShardKey() string        { return c.c.ShardKey }
func (c callMetadata) RoutingKey() string      { return c.c.RoutingKey }
func (c callMetadata) RoutingDelegate() string { return c.c.RoutingDelegate }
