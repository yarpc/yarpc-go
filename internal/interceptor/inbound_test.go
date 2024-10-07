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

// TestNopUnaryInbound ensures NopUnaryInbound calls the underlying handler without modification.
func TestNopUnaryInbound(t *testing.T) {
	var called bool
	handler := UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		called = true
		return nil
	})

	err := NopUnaryInbound.Handle(context.Background(), &transport.Request{}, nil, handler)
	assert.NoError(t, err)
	assert.True(t, called)
}

// TestApplyUnaryInbound ensures that UnaryInbound middleware wraps correctly.
func TestApplyUnaryInbound(t *testing.T) {
	var called bool
	handler := UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		called = true
		return nil
	})

	middleware := UnaryInboundFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
		assert.False(t, called)
		return h.Handle(ctx, req, resw)
	})

	wrappedHandler := ApplyUnaryInbound(handler, middleware)
	err := wrappedHandler.Handle(context.Background(), &transport.Request{}, nil)
	assert.NoError(t, err)
	assert.True(t, called)
}
