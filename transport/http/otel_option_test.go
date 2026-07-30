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

package http

import (
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/yarpc/internal/tracinginterceptor"
)

func TestOTelTracerProviderOption(t *testing.T) {
	tests := []struct {
		name                string
		options             []TransportOption
		wantOTelInterceptor bool
		wantOTInterceptor   bool
	}{
		{
			name:    "no tracing options installs neither interceptor",
			options: nil,
		},
		{
			name:              "TracingInterceptorEnabled installs the OT interceptor",
			options:           []TransportOption{TracingInterceptorEnabled(true)},
			wantOTInterceptor: true,
		},
		{
			name:                "OTelTracerProvider installs the OTel interceptor",
			options:             []TransportOption{OTelTracerProvider(sdktrace.NewTracerProvider())},
			wantOTelInterceptor: true,
		},
		{
			name: "OTelTracerProvider takes precedence over TracingInterceptorEnabled",
			options: []TransportOption{
				TracingInterceptorEnabled(true),
				OTelTracerProvider(sdktrace.NewTracerProvider()),
			},
			wantOTelInterceptor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans := NewTransport(append(tt.options, Tracer(mocktracer.New()))...)

			require.Equal(t, tt.wantOTelInterceptor, hasOTelInterceptor(trans.unaryOutboundInterceptor))
			require.Equal(t, tt.wantOTInterceptor, hasOTInterceptor(trans.unaryOutboundInterceptor))

			// Whenever an interceptor owns tracing, the legacy tracer must be
			// suppressed so spans are not double-reported.
			if tt.wantOTelInterceptor || tt.wantOTInterceptor {
				assert.Equal(t, opentracing.NoopTracer{}, trans.tracer)
			} else {
				assert.NotEqual(t, opentracing.NoopTracer{}, trans.tracer)
			}
		})
	}
}

func hasOTelInterceptor[T any](interceptors []T) bool {
	for _, i := range interceptors {
		if _, ok := any(i).(*tracinginterceptor.OTelInterceptor); ok {
			return true
		}
	}
	return false
}

func hasOTInterceptor[T any](interceptors []T) bool {
	for _, i := range interceptors {
		if _, ok := any(i).(*tracinginterceptor.Interceptor); ok {
			return true
		}
	}
	return false
}
