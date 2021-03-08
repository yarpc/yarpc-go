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

package encoding

import (
	"context"
	"sort"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/inboundcall"
	"go.uber.org/yarpc/yarpcerrors"
)

type keyValuePair struct{ k, v string }

// Call provides information about the current request inside handlers.
type Call struct{ md inboundcall.Metadata }

// CallFromContext retrieves information about the current incoming request
// from the given context. Returns nil if the context is not a valid request
// context.
//
// The object is valid only as long as the request is ongoing.
func CallFromContext(ctx context.Context) *Call {
	if md, ok := inboundcall.GetMetadata(ctx); ok {
		return &Call{md}
	}
	return nil
}

// WriteResponseHeader writes headers to the response of this call.
func (c *Call) WriteResponseHeader(k, v string) error {
	if c == nil {
		return yarpcerrors.InvalidArgumentErrorf(
			"failed to write response header: " +
				"Call was nil, make sure CallFromContext was called with a request context")
	}
	return c.md.WriteResponseHeader(k, v)
}

// Caller returns the name of the service making this request.
func (c *Call) Caller() string {
	if c == nil {
		return ""
	}
	return c.md.Caller()
}

// Service returns the name of the service being called.
func (c *Call) Service() string {
	if c == nil {
		return ""
	}
	return c.md.Service()
}

// Transport returns the name of the transport being called.
func (c *Call) Transport() string {
	if c == nil {
		return ""
	}
	return c.md.Transport()
}

// Procedure returns the name of the procedure being called.
func (c *Call) Procedure() string {
	if c == nil {
		return ""
	}
	return c.md.Procedure()
}

// Encoding returns the encoding for this request.
func (c *Call) Encoding() transport.Encoding {
	if c == nil {
		return ""
	}
	return c.md.Encoding()
}

// Header returns the value of the given request header provided with the
// request.
func (c *Call) Header(k string) string {
	if c == nil {
		return ""
	}

	if v, ok := c.md.Headers().Get(k); ok {
		return v
	}

	return ""
}

// HeaderNames returns a sorted list of the names of user defined headers
// provided with this request.
func (c *Call) HeaderNames() []string {
	if c == nil {
		return nil
	}

	items := c.md.Headers().Items()
	names := make([]string, 0, len(items))
	for k := range items {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ShardKey returns the shard key for this request.
func (c *Call) ShardKey() string {
	if c == nil {
		return ""
	}
	return c.md.ShardKey()
}

// RoutingKey returns the routing key for this request.
func (c *Call) RoutingKey() string {
	if c == nil {
		return ""
	}
	return c.md.RoutingKey()
}

// RoutingDelegate returns the routing delegate for this request.
func (c *Call) RoutingDelegate() string {
	if c == nil {
		return ""
	}
	return c.md.RoutingDelegate()
}

// CallerProcedure returns the name of the procedure from the service making this request.
func (c *Call) CallerProcedure() string {
	if c == nil {
		return ""
	}
	return c.md.CallerProcedure()
}
