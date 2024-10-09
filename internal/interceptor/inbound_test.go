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

package interceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

// UnaryHandlerFunc allows a function to be treated as a UnaryHandler for testing purposes.
type UnaryHandlerFunc func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error

// Handle calls the underlying function in UnaryHandlerFunc.
func (f UnaryHandlerFunc) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return f(ctx, req, resw)
}

// TestUnaryInboundFunc ensures that UnaryInboundFunc works correctly.
func TestUnaryInboundFunc(t *testing.T) {
	var called bool
	handler := UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		called = true
		return nil
	})

	middleware := UnaryInboundFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
		assert.False(t, called) // Ensure that the middleware is called before the handler.
		return h.Handle(ctx, req, resw)
	})

	err := middleware.Handle(context.Background(), &transport.Request{}, nil, handler)
	assert.NoError(t, err)
	assert.True(t, called)
}

// TestUnaryHandlerWithMiddleware ensures that the unaryHandlerWithMiddleware applies the middleware correctly.
func TestUnaryHandlerWithMiddleware(t *testing.T) {
	var called bool
	handler := UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		called = true
		return nil
	})

	middleware := UnaryInboundFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
		assert.False(t, called) // Ensure middleware is called before the handler.
		return h.Handle(ctx, req, resw)
	})

	wrappedHandler := unaryHandlerWithMiddleware{h: handler, i: middleware}
	err := wrappedHandler.Handle(context.Background(), &transport.Request{}, nil)
	assert.NoError(t, err)
	assert.True(t, called)
}
