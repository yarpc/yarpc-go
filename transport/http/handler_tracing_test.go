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
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/yarpcerrors"
)

// A tracer may embed NoopTracer while providing real tracing methods.
type customHTTPTracer struct {
	opentracing.NoopTracer
	tracer opentracing.Tracer
}

func (t customHTTPTracer) StartSpan(name string, opts ...opentracing.StartSpanOption) opentracing.Span {
	return t.tracer.StartSpan(name, opts...)
}

func (t customHTTPTracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return t.tracer.Extract(format, carrier)
}

func TestInboundTracing(t *testing.T) {
	legacyTracer := mocktracer.New()
	customTracer := mocktracer.New()
	interceptorTracer := mocktracer.New()
	tests := []struct {
		name        string
		tracer      opentracing.Tracer
		recorder    *mocktracer.MockTracer
		interceptor bool
	}{
		{name: "noop value", tracer: opentracing.NoopTracer{}},
		{name: "noop pointer", tracer: &opentracing.NoopTracer{}},
		{name: "legacy", tracer: legacyTracer, recorder: legacyTracer},
		{name: "custom", tracer: customHTTPTracer{tracer: customTracer}, recorder: customTracer},
		{name: "interceptor", tracer: interceptorTracer, recorder: interceptorTracer, interceptor: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			parentTracer := tt.recorder
			if parentTracer == nil {
				parentTracer = mocktracer.New()
			}
			parent := parentTracer.StartSpan("caller").(*mocktracer.MockSpan)
			defer parent.Finish()
			parent.SetBaggageItem("tenant", "delivery")

			type contextKey struct{}
			deadline := time.Now().Add(30 * time.Second)
			ctx, cancel := context.WithDeadline(context.WithValue(context.Background(), contextKey{}, "request"), deadline)
			defer cancel()
			ctx = opentracing.ContextWithSpan(ctx, parent)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request")).WithContext(ctx)
			req.Header.Set(CallerHeader, "caller")
			req.Header.Set(ServiceHeader, "service")
			req.Header.Set(ProcedureHeader, "hello")
			req.Header.Set(EncodingHeader, "raw")
			req.Header.Set(TTLMSHeader, "60000")
			headers := make(http.Header)
			require.NoError(t, parentTracer.Inject(parent.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(headers)))
			for key, values := range headers {
				req.Header[key] = values
				// Interceptor tracing reads application headers instead of the HTTP request.
				req.Header[ApplicationHeaderPrefix+key] = values
			}

			router := transporttest.NewMockRouter(ctrl)
			rpcHandler := transporttest.NewMockUnaryHandler(ctrl)
			router.EXPECT().Procedures()
			router.EXPECT().Choose(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, _ *transport.Request) (transport.HandlerSpec, error) {
					span := opentracing.SpanFromContext(ctx)
					require.NotNil(t, span)
					if tt.recorder == nil || tt.interceptor {
						assert.IsType(t, opentracing.NoopTracer{}, span.Tracer())
					} else {
						assert.Same(t, tt.recorder, span.Tracer())
					}
					return transport.NewUnaryHandlerSpec(rpcHandler), nil
				},
			)
			rpcHandler.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, _ *transport.Request, _ transport.ResponseWriter) error {
					assert.Equal(t, "request", ctx.Value(contextKey{}))
					gotDeadline, ok := ctx.Deadline()
					assert.True(t, ok)
					assert.Equal(t, deadline, gotDeadline)
					span := opentracing.SpanFromContext(ctx)
					require.NotNil(t, span)
					if tt.recorder == nil {
						assert.IsType(t, opentracing.NoopTracer{}, span.Tracer())
					} else {
						assert.Same(t, tt.recorder, span.Tracer())
						assert.Equal(t, "delivery", span.BaggageItem("tenant"))
					}
					return yarpcerrors.Newf(yarpcerrors.CodeUnavailable, "handler unavailable")
				},
			)
			inbound := startTracingTestInbound(t, tt.tracer, router, tt.interceptor)
			response := httptest.NewRecorder()
			inbound.server.Handler.ServeHTTP(response, req)
			assert.Equal(t, http.StatusServiceUnavailable, response.Code)

			spans := parentTracer.FinishedSpans()
			if tt.recorder == nil {
				assert.Empty(t, spans)
				return
			}
			require.Len(t, spans, 1)
			span := spans[0]
			assert.Equal(t, parent.SpanContext.TraceID, span.SpanContext.TraceID)
			assert.Equal(t, parent.SpanContext.SpanID, span.ParentID)
			assert.Equal(t, "hello", span.OperationName)
			assert.Equal(t, "caller", span.Tag("rpc.caller"))
			assert.Equal(t, "service", span.Tag("rpc.service"))
			assert.Equal(t, raw.Encoding, span.Tag("rpc.encoding"))
			assert.Equal(t, "http", span.Tag("rpc.transport"))
			assert.Equal(t, "caller", span.Tag("peer.service"))
			assert.Equal(t, ext.SpanKindRPCServerEnum, span.Tag("span.kind"))
			assert.Equal(t, true, span.Tag("error"))
			if !tt.interceptor {
				assert.NotEmpty(t, span.Logs())
				for key, value := range yarpc.OpentracingTags {
					assert.Equal(t, value, span.Tag(key))
				}
			}
		})
	}
}

func TestNoopSpanContextCancellation(t *testing.T) {
	inbound := startTracingTestInbound(t, opentracing.NoopTracer{}, newTestRouter(nil), false)
	h := inbound.server.Handler.(handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, span := h.createSpan(ctx, &http.Request{}, &transport.Request{}, time.Now())
	defer span.Finish()
	require.NotNil(t, opentracing.SpanFromContext(ctx))
	cancel()
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	select {
	case <-ctx.Done():
	default:
		t.Fatal("span context did not inherit cancellation")
	}
}

func startTracingTestInbound(t testing.TB, tracer opentracing.Tracer, router transport.Router, interceptor bool) *Inbound {
	t.Helper()
	inbound := NewTransport(Tracer(tracer), TracingInterceptorEnabled(interceptor)).NewInbound("127.0.0.1:0", DisableHTTP2(true))
	inbound.SetRouter(router)
	require.NoError(t, inbound.Start())
	t.Cleanup(func() { assert.NoError(t, inbound.Stop()) })
	return inbound
}
