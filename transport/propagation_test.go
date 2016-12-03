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

import (
	"context"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/uber/jaeger-client-go"
)

func TestContextMarshalling(t *testing.T) {
	tracer, closer := initTracer()
	defer closer()

	req := &Request{}
	baggage := map[string]string{
		"hello": "world",
		"foo":   "bar",
		":)":    ":(",
	}

	_, span := CreateOpentracingSpan(
		context.Background(),
		req,
		tracer,
		"fake-transport",
		time.Now())
	addBaggage(span, baggage)
	span.Finish()

	spanCtxBytes, err := MarshalSpanContext(tracer, span.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, spanCtxBytes)

	spanContext, err := UnmarshalSpanContext(tracer, spanCtxBytes)
	assert.NoError(t, err)
	assert.NotNil(t, span)

	_, span = ExtractOpenTracingSpan(
		context.Background(),
		spanContext,
		req,
		tracer,
		"fake-transport",
		time.Now())
	defer span.Finish()

	assert.Equal(t, baggage, getBaggage(span))
}

func initTracer() (opentracing.Tracer, func() error) {
	tracer, closer := jaeger.NewTracer(
		"internal-propagation",
		jaeger.NewConstSampler(true),
		jaeger.NewNullReporter())
	opentracing.InitGlobalTracer(tracer)
	return tracer, closer.Close
}

func addBaggage(span opentracing.Span, baggage map[string]string) {
	for k, v := range baggage {
		span.SetBaggageItem(k, v)
	}
}
func getBaggage(span opentracing.Span) map[string]string {
	baggage := make(map[string]string)
	span.Context().ForeachBaggageItem(func(k, v string) bool {
		baggage[k] = v
		return true
	})

	return baggage
}
