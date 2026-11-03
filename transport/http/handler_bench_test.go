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
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"go.uber.org/yarpc/api/transport"
)

func BenchmarkHandlerCreateSpan(b *testing.B) {
	for _, tt := range []struct {
		name   string
		tracer opentracing.Tracer
	}{
		{name: "noop", tracer: opentracing.NoopTracer{}},
		{name: "noop-pointer", tracer: &opentracing.NoopTracer{}},
		{name: "recording", tracer: mocktracer.New()},
	} {
		b.Run(tt.name, func(b *testing.B) {
			inbound := startTracingTestInbound(b, tt.tracer, newTestRouter(nil), false)
			h := inbound.server.Handler.(handler)
			ctx := context.Background()
			req := &http.Request{Header: make(http.Header)}
			treq := &transport.Request{Caller: "caller", Service: "service", Procedure: "hello", Encoding: "raw"}
			start := time.Now()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Measure setup only; finishing recording spans would retain every sample.
				h.createSpan(ctx, req, treq, start)
			}
		})
	}
}
