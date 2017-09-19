// Copyright (c) 2017 Uber Technologies, Inc.
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

// Package nooptrace is a wrapper for handling opentracing.NoopTracers.
package nooptrace

import opentracing "github.com/opentracing/opentracing-go"

// GetTracer returns tracer if tracer is not nil and not a NoopTracer,
// otherwise opentracing.GlobalTracer() if not nil and not a NoopTracer,
// otherwise nil.
//
// This helps with optimizing tracing logic in transport implementations.
func GetTracer(tracer opentracing.Tracer) opentracing.Tracer {
	if isNotNoopTracer(tracer) {
		return tracer
	}
	tracer = opentracing.GlobalTracer()
	if isNotNoopTracer(tracer) {
		return tracer
	}
	return nil
}

func isNotNoopTracer(tracer opentracing.Tracer) bool {
	if tracer == nil {
		return false
	}
	_, ok := tracer.(opentracing.NoopTracer)
	return !ok
}
