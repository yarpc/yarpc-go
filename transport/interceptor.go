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

package transport

import "golang.org/x/net/context"

// Interceptor defines a transport-level middleware for Inbounds.
type Interceptor interface {
	Apply(ctx context.Context, req *Request, resw ResponseWriter, handler Handler) error
}

// NopInterceptor is a interceptor that does not do anything special. It
// simply calls the underlying Handler.
var NopInterceptor Interceptor = nopInterceptor{}

// ApplyInterceptor wraps the given Handler with the given Interceptors
// in-order.
func ApplyInterceptor(h Handler, i Interceptor) Handler {
	return interceptedHandler{h: h, i: i}
}

// ChainInterceptors combines the given interceptors in-order into a single
// Interceptor.
func ChainInterceptors(interceptors ...Interceptor) Interceptor {
	if len(interceptors) == 0 {
		return NopInterceptor
	}
	if len(interceptors) == 1 {
		return interceptors[0]
	}
	return interceptorChain(interceptors)
}

type interceptedHandler struct {
	h Handler
	i Interceptor
}

func (h interceptedHandler) Handle(ctx context.Context, req *Request, resw ResponseWriter) error {
	return h.i.Apply(ctx, req, resw, h.h)
}

type nopInterceptor struct{}

func (nopInterceptor) Apply(ctx context.Context, req *Request, resw ResponseWriter, handler Handler) error {
	return handler.Handle(ctx, req, resw)
}

type interceptorChain []Interceptor

func (ic interceptorChain) Apply(ctx context.Context, req *Request, resw ResponseWriter, handler Handler) error {
	ix := interceptorChainExec{
		Chain: []Interceptor(ic),
		Index: 0,
		Final: handler,
	}
	return ix.Handle(ctx, req, resw)
}

// interceptorChainExec adapts a series of interceptors into a Handler. It is
// scoped to a single request to the Handler and is not thread-safe.
type interceptorChainExec struct {
	Chain []Interceptor
	Index int
	Final Handler
}

func (ix *interceptorChainExec) Handle(ctx context.Context, req *Request, resw ResponseWriter) error {
	i := ix.Index
	if i == len(ix.Chain) {
		return ix.Final.Handle(ctx, req, resw)
	}

	ix.Index++
	return ix.Chain[i].Apply(ctx, req, resw, ix)
}
