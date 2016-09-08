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

import "github.com/opentracing/opentracing-go"

// Deps is a struct shared by all inbounds and outbounds in the context of
// a dispatcher. The dispatcher starts every transport with these dependencies.
// A zero Deps struct is suitable for testing and provides noop implementations
// of all dependencies.
type Deps struct {
	tracer opentracing.Tracer
}

// NoDeps is a singleton zero Deps instance.
var NoDeps Deps

// WithTracer returns a variant of these dependencies with a given opentracing Tracer.
func (d Deps) WithTracer(t opentracing.Tracer) Deps {
	d.tracer = t
	return d
}

// Tracer provides the opentracing Tracer instance needed by transports.
func (d Deps) Tracer() opentracing.Tracer {
	if d.tracer != nil {
		return d.tracer
	}
	return opentracing.GlobalTracer()
}
