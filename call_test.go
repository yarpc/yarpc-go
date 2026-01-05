// Copyright (c) 2026 Uber Technologies, Inc.
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

package yarpc_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	pkgencoding "go.uber.org/yarpc/pkg/encoding"
)

func TestCallOptionsWriteToRequest(t *testing.T) {
	outboundCall := encoding.NewOutboundCall(
		pkgencoding.FromOptions(
			[]yarpc.CallOption{
				yarpc.WithShardKey("foo"),
				yarpc.WithRoutingKey("bar"),
				yarpc.WithRoutingDelegate("baz"),
			},
		)...,
	)
	request := &transport.Request{}
	_, err := outboundCall.WriteToRequest(context.Background(), request)
	assert.NoError(t, err)
	assert.Equal(t, "foo", request.ShardKey)
	assert.Equal(t, "bar", request.RoutingKey)
	assert.Equal(t, "baz", request.RoutingDelegate)
}

func TestCallFromContext(t *testing.T) {
	ctx, inboundCall := encoding.NewInboundCall(context.Background())
	err := inboundCall.ReadFromRequest(
		&transport.Request{
			Caller:    "foo",
			Service:   "bar",
			Transport: "trans",
			Encoding:  transport.Encoding("baz"),
			Procedure: "hello",
			// later header's key/value takes precedence
			Headers:         transport.NewHeaders().With("Foo", "Bar").With("foo", "bar"),
			ShardKey:        "one",
			RoutingKey:      "two",
			RoutingDelegate: "three",
			CallerProcedure: "four",
		},
	)
	assert.NoError(t, err)
	call := yarpc.CallFromContext(ctx)
	assert.Equal(t, "foo", call.Caller())
	assert.Equal(t, "bar", call.Service())
	assert.Equal(t, "trans", call.Transport())
	assert.Equal(t, transport.Encoding("baz"), call.Encoding())
	assert.Equal(t, "hello", call.Procedure())
	assert.Equal(t, "bar", call.Header("foo"))
	assert.Equal(t, "bar", call.OriginalHeader("foo"))
	assert.Equal(t, "Bar", call.OriginalHeader("Foo"))
	assert.Equal(t, map[string]string{"Foo": "Bar", "foo": "bar"}, call.OriginalHeaders())
	assert.Equal(t, map[string]string{"foo": "bar"}, call.Headers()) // Headers are case insensitive
	assert.Equal(t, []string{"foo"}, call.HeaderNames())
	assert.Equal(t, []string{"foo"}, slices.Collect(call.HeaderNamesAll()))
	assert.Equal(t, map[string]string{"foo": "bar"}, maps.Collect(call.HeadersAll()))
	assert.Equal(t, map[string]string{"Foo": "Bar", "foo": "bar"}, maps.Collect(call.OriginalHeadersAll()))
	assert.Equal(t, 1, call.HeadersLen())
	assert.Equal(t, 2, call.OriginalHeadersLen())
	assert.Equal(t, "one", call.ShardKey())
	assert.Equal(t, "two", call.RoutingKey())
	assert.Equal(t, "three", call.RoutingDelegate())
	assert.Equal(t, "four", call.CallerProcedure())
}
