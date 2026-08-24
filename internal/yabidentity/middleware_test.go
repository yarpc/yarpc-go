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

package yabidentity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

func newTestMiddleware(t *testing.T) (*Middleware, *metrics.Root) {
	root := metrics.New()
	return New(root.Scope(), zap.NewNop()), root
}

type unaryHandlerFunc func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error

func (f unaryHandlerFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return f(ctx, req, resw)
}

type onewayHandlerFunc func(ctx context.Context, req *transport.Request) error

func (f onewayHandlerFunc) HandleOneway(ctx context.Context, req *transport.Request) error {
	return f(ctx, req)
}

func TestHandleStripsYabHeadersAndTags(t *testing.T) {
	m, root := newTestMiddleware(t)

	var seenHeaders map[string]string
	handler := unaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		seenHeaders = req.Headers.Items()
		return nil
	})

	req := &transport.Request{
		Caller:  "yab-person1",
		Service: "myservice",
		Headers: transport.HeadersFromMap(map[string]string{
			"x-yab-client":          "true",
			"x-yab-source":          "realservice",
			"x-yab-dest":            "myservice",
			"x-yab-env":             "production",
			"x-yab-source-mismatch": "true",
			"other-header":          "keepme",
		}),
	}

	err := m.Handle(context.Background(), req, nil, handler)
	require.NoError(t, err)

	// yab headers must not reach the handler.
	for _, h := range []string{"x-yab-client", "x-yab-source", "x-yab-dest", "x-yab-env", "x-yab-source-mismatch"} {
		_, ok := seenHeaders[h]
		assert.False(t, ok, "expected %q to be stripped", h)
	}
	assert.Equal(t, "keepme", seenHeaders["other-header"])

	snap := root.Snapshot()
	require.Len(t, snap.Counters, 1)
	assert.Equal(t, "yab_calls", snap.Counters[0].Name)
	assert.Equal(t, int64(1), snap.Counters[0].Value)
	assert.Equal(t, "realservice", snap.Counters[0].Tags["source"])
	assert.Equal(t, "production", snap.Counters[0].Tags["env"])
	assert.Equal(t, "true", snap.Counters[0].Tags["source_mismatch"])
}

func TestHandleNonYabRequestUntouched(t *testing.T) {
	m, root := newTestMiddleware(t)

	var seenHeaders map[string]string
	handler := unaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		seenHeaders = req.Headers.Items()
		return nil
	})

	req := &transport.Request{
		Caller:  "realservice",
		Service: "myservice",
		Headers: transport.HeadersFromMap(map[string]string{
			"other-header": "keepme",
		}),
	}

	err := m.Handle(context.Background(), req, nil, handler)
	require.NoError(t, err)

	assert.Equal(t, "keepme", seenHeaders["other-header"])

	snap := root.Snapshot()
	assert.Len(t, snap.Counters, 0)
}

func TestHandleOnewayStripsYabHeaders(t *testing.T) {
	m, root := newTestMiddleware(t)

	var seenHeaders map[string]string
	handler := onewayHandlerFunc(func(ctx context.Context, req *transport.Request) error {
		seenHeaders = req.Headers.Items()
		return nil
	})

	req := &transport.Request{
		Caller:  "yab-person1",
		Service: "myservice",
		Headers: transport.HeadersFromMap(map[string]string{
			"x-yab-client": "true",
		}),
	}

	err := m.HandleOneway(context.Background(), req, handler)
	require.NoError(t, err)

	_, ok := seenHeaders["x-yab-client"]
	assert.False(t, ok)

	snap := root.Snapshot()
	require.Len(t, snap.Counters, 1)
	assert.Equal(t, "__unknown__", snap.Counters[0].Tags["source"])
	assert.Equal(t, "__unknown__", snap.Counters[0].Tags["env"])
	assert.Equal(t, "false", snap.Counters[0].Tags["source_mismatch"])
}

func TestNewWithNilLogger(t *testing.T) {
	root := metrics.New()
	m := New(root.Scope(), nil)
	assert.NotNil(t, m.logger)
}
