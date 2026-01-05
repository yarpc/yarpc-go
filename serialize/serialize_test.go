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

package serialize

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jaeger "github.com/uber/jaeger-client-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
)

func TestSerialization(t *testing.T) {
	tracer := opentracing.NoopTracer{}
	spanContext := tracer.StartSpan("test-span").Context()

	headers := map[string]string{
		"hello": "world",
		"foo":   "bar",
	}
	body := []byte(
		"My mind is tellin' me no... but my body... my body's tellin' me yes",
	)

	haveReq := &transport.Request{
		Caller:          "Caller",
		Service:         "ServiceName",
		Encoding:        "Encoding",
		Procedure:       "Procedure",
		Headers:         transport.HeadersFromMap(headers),
		ShardKey:        "ShardKey",
		RoutingKey:      "RoutingKey",
		RoutingDelegate: "RoutingDelegate",
		Body:            bytes.NewReader(body),
	}

	matcher := transporttest.NewRequestMatcher(t, haveReq)

	marshalledReq, err := ToBytes(tracer, spanContext, haveReq)
	require.NoError(t, err, "could not marshal RPC to bytes")
	assert.NotEmpty(t, marshalledReq)
	assert.Equal(t, byte(0), marshalledReq[0], "serialization byte invalid")

	_, gotReq, err := FromBytes(tracer, marshalledReq)
	require.NoError(t, err, "could not unmarshal RPC from bytes")

	assert.True(t, matcher.Matches(gotReq))
}

func TestDeserializationFailure(t *testing.T) {
	tracer := opentracing.NoopTracer{}
	spanContext := tracer.StartSpan("test-span").Context()

	req := &transport.Request{
		Caller:    "Caller",
		Service:   "ServiceName",
		Procedure: "Procedure",
		Body:      strings.NewReader("someBODY"),
	}

	marshalledReq, err := ToBytes(tracer, spanContext, req)
	require.NoError(t, err, "could not marshal RPC to bytes")
	require.NotEmpty(t, marshalledReq)

	// modify serialization byte to something unsupported
	marshalledReq[0] = 1

	_, _, err = FromBytes(tracer, marshalledReq)
	assert.Error(t, err, "able to deserialize RPC from bytes")
}

func TestContextSerialization(t *testing.T) {
	tracer, closer := initTracer()
	defer closer()

	req := &transport.Request{}
	baggage := map[string]string{
		"hello": "world",
		"foo":   "bar",
		":)":    ":(",
	}

	createOpenTracingSpan := transport.CreateOpenTracingSpan{
		Tracer:        tracer,
		TransportName: "fake-transport",
		StartTime:     time.Now(),
	}

	_, span := createOpenTracingSpan.Do(context.Background(), req)
	addBaggage(span, baggage)
	span.Finish()

	spanCtxBytes, err := spanContextToBytes(tracer, span.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, spanCtxBytes)

	spanContext, err := spanContextFromBytes(tracer, spanCtxBytes)
	assert.NoError(t, err)
	assert.NotNil(t, span)

	extractOpenTracingSpan := transport.ExtractOpenTracingSpan{
		ParentSpanContext: spanContext,
		Tracer:            tracer,
		TransportName:     "fake-transport",
		StartTime:         time.Now(),
	}

	_, span = extractOpenTracingSpan.Do(context.Background(), req)
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
