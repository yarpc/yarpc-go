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
//
// Interceptors MAY
//
// - change the context
// - change the request
// - call the ResponseWriter
// - modify the response body by wrapping the ResponseWriter
// - handle the returned error
// - call the given handler zero or more times
//
// Interceptors MUST be thread-safe.
//
// Interceptors are re-used across requests and MAY be called multiple times for
// the same request.
type Interceptor interface {
	Apply(ctx context.Context, req *Request, resw ResponseWriter, h Handler) error
}

// NopInterceptor is a interceptor that does not do anything special. It
// simply calls the underlying Handler.
var NopInterceptor Interceptor = nopInterceptor{}

// ApplyInterceptor applies the given Interceptor to the given Handler.
func ApplyInterceptor(h Handler, i Interceptor) Handler {
	return interceptedHandler{h: h, i: i}
}

// InterceptorFunc adapts a function into an Interceptor.
type InterceptorFunc func(context.Context, *Request, ResponseWriter, Handler) error

// Apply for InterceptorFunc
func (f InterceptorFunc) Apply(ctx context.Context, req *Request, resw ResponseWriter, h Handler) error {
	return f(ctx, req, resw, h)
}

// ChainInterceptors combines the given interceptors in-order into a single
// Interceptor.
func ChainInterceptors(interceptors ...Interceptor) Interceptor {
	switch len(interceptors) {
	case 0:
		return NopInterceptor
	case 1:
		return interceptors[0]
	default:
		return interceptorChain(interceptors)
	}
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
	return interceptorChainExec{
		Chain: []Interceptor(ic),
		Final: handler,
	}.Handle(ctx, req, resw)
}

// interceptorChainExec adapts a series of interceptors into a Handler. It is
// scoped to a single request to the Handler and is not thread-safe.
type interceptorChainExec struct {
	Chain []Interceptor
	Final Handler
}

func (ix interceptorChainExec) Handle(ctx context.Context, req *Request, resw ResponseWriter) error {
	if len(ix.Chain) == 0 {
		return ix.Final.Handle(ctx, req, resw)
	}
	return ix.Chain[0].Apply(ctx, req, resw, interceptorChainExec{
		Chain: ix.Chain[1:],
		Final: ix.Final,
	})
}
