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

package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestOTelHeadersCarrierSetAddsPrefix(t *testing.T) {
	headers := make(map[string]string)
	carrier := OTelHeadersCarrier(headers)

	carrier.Set("uber-trace-id", "trace-value")

	assert.Equal(t, map[string]string{"$tracing$uber-trace-id": "trace-value"}, headers)
}

func TestOTelHeadersCarrierGetStripsPrefix(t *testing.T) {
	carrier := OTelHeadersCarrier(map[string]string{
		"$tracing$uber-trace-id": "trace-value",
		"not-a-tracing-header":   "ignored",
	})

	assert.Equal(t, "trace-value", carrier.Get("uber-trace-id"))
	assert.Empty(t, carrier.Get("missing-key"), "absent keys return the empty string")
	assert.Empty(t, carrier.Get("not-a-tracing-header"), "unprefixed headers are not tracing headers")
}

func TestOTelHeadersCarrierKeys(t *testing.T) {
	carrier := OTelHeadersCarrier(map[string]string{
		"$tracing$uber-trace-id": "trace-value",
		"$tracing$uberctx-key":   "baggage-value",
		"application-header":     "ignored",
	})

	assert.ElementsMatch(t, []string{"uber-trace-id", "uberctx-key"}, carrier.Keys())
}

// Keys are matched exactly. Both ends of a TChannel call write tracing headers
// through a propagator that emits lowercase names, and TChannel does not
// canonicalize header casing the way HTTP does, so no normalization is applied.
func TestOTelHeadersCarrierGetIsCaseSensitive(t *testing.T) {
	carrier := OTelHeadersCarrier(map[string]string{
		"$tracing$Uber-Trace-Id": "trace-value",
	})

	assert.Empty(t, carrier.Get("uber-trace-id"))
	assert.Equal(t, "trace-value", carrier.Get("Uber-Trace-Id"))
}

// The carrier must round-trip a real propagator: what Inject writes on one side
// of a TChannel hop is what Extract reads on the other.
func TestOTelHeadersCarrierRoundTripsThroughPropagator(t *testing.T) {
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})

	headers := make(map[string]string)
	propagator.Inject(
		trace.ContextWithSpanContext(context.Background(), spanCtx),
		OTelHeadersCarrier(headers),
	)
	require.NotEmpty(t, headers, "propagator must write at least one header")
	for k := range headers {
		assert.Contains(t, k, tchannelTracingKeyPrefix, "injected header %q must be prefixed", k)
	}

	extracted := trace.SpanContextFromContext(
		propagator.Extract(context.Background(), OTelHeadersCarrier(headers)),
	)
	assert.Equal(t, spanCtx.TraceID(), extracted.TraceID())
	assert.Equal(t, spanCtx.SpanID(), extracted.SpanID())
	assert.True(t, extracted.IsSampled())
}
