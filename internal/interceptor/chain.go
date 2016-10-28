// Copyright (c) 2016 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/transport"

	"context"
)

// Chain combines a series of Interceptors into a single Interceptor.
func Chain(interceptors ...transport.Interceptor) transport.Interceptor {
	switch len(interceptors) {
	case 0:
		return transport.NopInterceptor
	case 1:
		return interceptors[0]
	default:
		return chain(interceptors)
	}
}

// interceptorChain combines a series of interceptors into a single Interceptor.
type chain []transport.Interceptor

func (c chain) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.Handler) error {
	return chainExec{
		Chain: []transport.Interceptor(c),
		Final: h,
	}.Handle(ctx, req, resw)
}

// chainExec adapts a series of interceptors into a Handler. It is scoped to a
// single request to the Handler and is not thread-safe.
type chainExec struct {
	Chain []transport.Interceptor
	Final transport.Handler
}

func (x chainExec) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	if len(x.Chain) == 0 {
		return x.Final.Handle(ctx, req, resw)
	}
	next := x.Chain[0]
	x.Chain = x.Chain[1:]
	return next.Handle(ctx, req, resw, x)
}
