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

package raw

import (
	"context"

	"go.uber.org/yarpc/api/transport"
)

// Register calls the RouteTable's Register method.
//
// This function exists for backwards compatibility only. It will be removed
// in a future version.
//
// Deprecated: Use the RouteTable's Register method directly.
func Register(r transport.RouteTable, rs []transport.Procedure) {
	r.Register(rs)
}

// UnaryHandler implements a single, unary procedure.
type UnaryHandler func(context.Context, []byte) ([]byte, error)

// Procedure builds a Procedure from the given raw handler.
func Procedure(name string, handler UnaryHandler) []transport.Procedure {
	return []transport.Procedure{
		{
			Name:        name,
			HandlerSpec: transport.NewUnaryHandlerSpec(rawUnaryHandler{handler}),
		},
	}
}

// OnewayHandler implements a single, onweway procedure
type OnewayHandler func(context.Context, []byte) error

// OnewayProcedure builds a Procedure from the given raw handler
func OnewayProcedure(name string, handler OnewayHandler) []transport.Procedure {
	return []transport.Procedure{
		{
			Name:        name,
			HandlerSpec: transport.NewOnewayHandlerSpec(rawOnewayHandler{handler}),
		},
	}
}
