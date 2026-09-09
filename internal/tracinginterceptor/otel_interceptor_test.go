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

package tracinginterceptor

import (
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/interceptor/interceptortest"
	"go.uber.org/yarpc/yarpcerrors"
)

// newTestProvider returns a TracerProvider backed by an in-memory span recorder.
func newTestProvider(t *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return tp, sr
}

// testPropagator returns a propagator stand-in for tests. yarpc-go has no
// dependency on Uber's internal jaeger client, so we use an SDK propagator
// purely to exercise the inject/extract plumbing. Production callers pass
// Uber's composite JaegerTrace+JaegerBaggage propagator via OTelParams.
func testPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newOTelInterceptor(t *testing.T, transportName string) (*OTelInterceptor, *tracetest.SpanRecorder) {
	t.Helper()
	tp, sr := newTestProvider(t)
	i := NewOTel(OTelParams{
		TracerProvider: tp,
		Propagator:     testPropagator(),
		Transport:      transportName,
	})
	return i, sr
}

// hasAttr reports whether a span carries an attribute with the given key and value.
func hasAttr(attrs []attribute.KeyValue, key, value string) bool {
	for _, a := range attrs {
		if string(a.Key) == key && a.Value.AsString() == value {
			return true
		}
	}
	return false
}

func TestOTelInterceptorHandle(t *testing.T) {
	tests := []struct {
		name               string
		handlerError       error
		isApplicationError bool
		appErrorMeta       *transport.ApplicationErrorMeta
		useNonExtendedRW   bool
		expectedStatus     codes.Code
	}{
		{
			name:           "successful handle with no errors",
			expectedStatus: codes.Unset,
		},
		{
			name:           "handler returns a yarpc error",
			handlerError:   yarpcerrors.Newf(yarpcerrors.CodeInternal, "handler error"),
			expectedStatus: codes.Error,
		},
		{
			name:               "application error",
			isApplicationError: true,
			expectedStatus:     codes.Error,
		},
		{
			name:               "application error with metadata",
			isApplicationError: true,
			appErrorMeta: &transport.ApplicationErrorMeta{
				Code: (*yarpcerrors.Code)(func() *int { c := 500; return &c }()),
				Name: "InternalError",
			},
			expectedStatus: codes.Error,
		},
		{
			name:             "non-ExtendedResponseWriter falls back gracefully",
			useNonExtendedRW: true,
			expectedStatus:   codes.Unset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			i, sr := newOTelInterceptor(t, "grpc")

			var rw transport.ResponseWriter
			if tt.useNonExtendedRW {
				rw = &transporttest.FakeResponseWriter{}
			} else {
				trw := &testResponseWriter{FakeResponseWriter: &transporttest.FakeResponseWriter{}}
				if tt.isApplicationError {
					trw.SetApplicationError()
					trw.appErrorMeta = tt.appErrorMeta
				}
				rw = trw
			}

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			handler := transporttest.NewMockUnaryHandler(ctrl)
			handler.EXPECT().Handle(gomock.Any(), req, rw).Return(tt.handlerError)

			err := i.Handle(context.Background(), req, rw, handler)
			if tt.handlerError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			spans := sr.Ended()
			require.Len(t, spans, 1)
			assert.Equal(t, tt.expectedStatus, spans[0].Status().Code)
			assert.Equal(t, trace.SpanKindServer, spans[0].SpanKind())
		})
	}
}

func TestOTelInterceptorCall(t *testing.T) {
	tests := []struct {
		name           string
		response       *transport.Response
		callError      error
		expectedStatus codes.Code
	}{
		{
			name:           "successful call with no errors",
			response:       &transport.Response{},
			expectedStatus: codes.Unset,
		},
		{
			name:           "call returns a yarpc error",
			callError:      yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "call error"),
			expectedStatus: codes.Error,
		},
		{
			name:           "application error in response",
			response:       &transport.Response{ApplicationError: true},
			expectedStatus: codes.Error,
		},
		{
			name:           "generic non-yarpc error",
			callError:      errors.New("generic error"),
			expectedStatus: codes.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			i, sr := newOTelInterceptor(t, "grpc")

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			outbound := interceptortest.NewMockUnaryOutboundChain(ctrl)
			outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(tt.response, tt.callError)

			res, err := i.Call(context.Background(), req, outbound)
			if tt.callError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response, res)
			}

			spans := sr.Ended()
			require.Len(t, spans, 1)
			assert.Equal(t, tt.expectedStatus, spans[0].Status().Code)
			assert.Equal(t, trace.SpanKindClient, spans[0].SpanKind())
		})
	}
}

func TestOTelInterceptorCall_PropagatesHeaders(t *testing.T) {
	tests := []struct {
		name          string
		transportName string
		wantPrefix    bool
	}{
		{
			name:          "grpc injects headers without $tracing$ prefix",
			transportName: "grpc",
			wantPrefix:    false,
		},
		{
			name:          "tchannel wraps injected headers with $tracing$ prefix",
			transportName: "tchannel",
			wantPrefix:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			i, _ := newOTelInterceptor(t, tt.transportName)

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.NewHeaders(),
			}

			outbound := interceptortest.NewMockUnaryOutboundChain(ctrl)
			outbound.EXPECT().Next(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, r *transport.Request) (*transport.Response, error) {
					items := r.Headers.Items()
					assert.NotEmpty(t, items, "propagator must inject at least one tracing header")
					for k := range items {
						if tt.wantPrefix {
							assert.True(t, strings.HasPrefix(k, "$tracing$"),
								"tchannel header %q must have $tracing$ prefix", k)
						} else {
							assert.False(t, strings.HasPrefix(k, "$tracing$"),
								"non-tchannel header %q must not have $tracing$ prefix", k)
						}
					}
					return &transport.Response{}, nil
				},
			)

			_, err := i.Call(context.Background(), req, outbound)
			require.NoError(t, err)
		})
	}
}

func TestOTelInterceptorHandleOneway(t *testing.T) {
	tests := []struct {
		name           string
		handlerError   error
		expectedStatus codes.Code
	}{
		{
			name:           "successful handle oneway",
			expectedStatus: codes.Unset,
		},
		{
			name:           "handle oneway returns a yarpc error",
			handlerError:   yarpcerrors.Newf(yarpcerrors.CodeInternal, "handler error"),
			expectedStatus: codes.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			i, sr := newOTelInterceptor(t, "grpc")

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			handler := transporttest.NewMockOnewayHandler(ctrl)
			handler.EXPECT().HandleOneway(gomock.Any(), req).Return(tt.handlerError)

			err := i.HandleOneway(context.Background(), req, handler)
			if tt.handlerError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			spans := sr.Ended()
			require.Len(t, spans, 1)
			assert.Equal(t, tt.expectedStatus, spans[0].Status().Code)
			assert.Equal(t, trace.SpanKindServer, spans[0].SpanKind())
		})
	}
}

func TestOTelInterceptorCallOneway(t *testing.T) {
	tests := []struct {
		name           string
		callError      error
		expectedStatus codes.Code
	}{
		{
			name:           "successful call oneway",
			expectedStatus: codes.Unset,
		},
		{
			name:           "call oneway returns a yarpc error",
			callError:      yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "call error"),
			expectedStatus: codes.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			i, sr := newOTelInterceptor(t, "grpc")

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			outbound := interceptortest.NewMockOnewayOutboundChain(ctrl)
			outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(nil, tt.callError)

			_, err := i.CallOneway(context.Background(), req, outbound)
			if tt.callError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			spans := sr.Ended()
			require.Len(t, spans, 1)
			assert.Equal(t, tt.expectedStatus, spans[0].Status().Code)
			assert.Equal(t, trace.SpanKindClient, spans[0].SpanKind())
		})
	}
}

func TestOTelUpdateSpanWithErrorDetails(t *testing.T) {
	statusCodeKey := attribute.Key(rpcStatusCodeTag)
	errorNameKey := attribute.Key(errorNameTag)

	tests := []struct {
		name               string
		err                error
		isApplicationError bool
		appErrorMeta       *transport.ApplicationErrorMeta
		expectedStatus     codes.Code
		expectedAttrs      map[attribute.Key]string
	}{
		{
			name:           "nil error and no application error — no-op",
			expectedStatus: codes.Unset,
		},
		{
			name:           "known yarpc error sets Error status",
			err:            yarpcerrors.Newf(yarpcerrors.CodeInternal, "known error"),
			expectedStatus: codes.Error,
			// yarpcerrors.CodeInternal == 13
			expectedAttrs: map[attribute.Key]string{statusCodeKey: "13"},
		},
		{
			name:           "generic unknown error is reported as CodeUnknown",
			err:            errors.New("random unknown error"),
			expectedStatus: codes.Error,
			// yarpcerrors.CodeUnknown == 2
			expectedAttrs: map[attribute.Key]string{statusCodeKey: "2"},
		},
		{
			name:               "application error with metadata",
			isApplicationError: true,
			appErrorMeta: &transport.ApplicationErrorMeta{
				Code: (*yarpcerrors.Code)(func() *int { c := 500; return &c }()),
				Name: "InternalError",
			},
			expectedStatus: codes.Error,
			expectedAttrs: map[attribute.Key]string{
				statusCodeKey: "500",
				errorNameKey:  "InternalError",
			},
		},
		{
			name:               "application error without metadata",
			isApplicationError: true,
			appErrorMeta:       nil,
			expectedStatus:     codes.Error,
			expectedAttrs:      map[attribute.Key]string{statusCodeKey: applicationError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp, sr := newTestProvider(t)
			_, span := tp.Tracer("test").Start(context.Background(), "test-span")

			returnedErr := otelUpdateSpanWithErrorDetails(span, tt.isApplicationError, tt.appErrorMeta, tt.err)
			span.End()

			assert.Equal(t, tt.err, returnedErr)

			spans := sr.Ended()
			require.Len(t, spans, 1)
			assert.Equal(t, tt.expectedStatus, spans[0].Status().Code)

			attrs := spans[0].Attributes()
			for key, want := range tt.expectedAttrs {
				found := false
				for _, a := range attrs {
					if a.Key == key {
						assert.Equal(t, want, a.Value.Emit())
						found = true
					}
				}
				assert.True(t, found, "expected attribute %q not found in span", key)
			}
		})
	}
}

// The error message must never reach the span. It can carry request data, and
// the OpenTracing interceptor did not emit it either.
func TestOTelUpdateSpanOmitsErrorMessage(t *testing.T) {
	tp, sr := newTestProvider(t)
	_, span := tp.Tracer("test").Start(context.Background(), "test-span")

	otelUpdateSpanWithErrorDetails(span, false, nil, errors.New("sensitive-payload-detail"))
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)

	assert.Empty(t, spans[0].Events(), "no exception event should be recorded")
	assert.Empty(t, spans[0].Status().Description, "status description must not carry the error message")
	for _, a := range spans[0].Attributes() {
		assert.NotContains(t, a.Value.Emit(), "sensitive-payload-detail")
	}
}

func TestOTelInterceptorHandleStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i, sr := newOTelInterceptor(t, "grpc")

	mockStream := transporttest.NewMockStream(ctrl)
	mockStream.EXPECT().Context().Return(context.Background()).AnyTimes()
	mockStream.EXPECT().Request().Return(&transport.StreamRequest{
		Meta: &transport.RequestMeta{Procedure: "test-procedure"},
	}).AnyTimes()

	serverStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err)

	handler := transporttest.NewMockStreamHandler(ctrl)
	handler.EXPECT().HandleStream(gomock.Any()).Return(nil)

	require.NoError(t, i.HandleStream(serverStream, handler))

	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status().Code)
	assert.Equal(t, trace.SpanKindServer, spans[0].SpanKind())
}

func TestOTelInterceptorHandleStream_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i, sr := newOTelInterceptor(t, "grpc")

	mockStream := transporttest.NewMockStream(ctrl)
	mockStream.EXPECT().Context().Return(context.Background()).AnyTimes()
	mockStream.EXPECT().Request().Return(&transport.StreamRequest{
		Meta: &transport.RequestMeta{Procedure: "test-procedure"},
	}).AnyTimes()

	serverStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err)

	handler := transporttest.NewMockStreamHandler(ctrl)
	handler.EXPECT().HandleStream(gomock.Any()).Return(yarpcerrors.Newf(yarpcerrors.CodeInternal, "handler error"))

	require.Error(t, i.HandleStream(serverStream, handler))

	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status().Code)
}

func TestOTelInterceptorHandleStream_ContextPropagation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i, _ := newOTelInterceptor(t, "grpc")

	baseCtx := context.Background()
	mockStream := transporttest.NewMockStream(ctrl)
	mockStream.EXPECT().Context().Return(baseCtx).AnyTimes()
	mockStream.EXPECT().Request().Return(&transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Procedure: "test-procedure",
			Headers:   transport.NewHeaders(),
		},
	}).AnyTimes()

	serverStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err)

	var capturedCtx context.Context
	handler := transporttest.NewMockStreamHandler(ctrl)
	handler.EXPECT().HandleStream(gomock.Any()).DoAndReturn(func(s *transport.ServerStream) error {
		capturedCtx = s.Context()
		return nil
	})

	require.NoError(t, i.HandleStream(serverStream, handler))
	require.NotNil(t, capturedCtx)

	span := trace.SpanFromContext(capturedCtx)
	assert.True(t, span.SpanContext().IsValid(), "expected a valid span in the handler's context")
	assert.NotEqual(t, baseCtx, capturedCtx, "context must be enriched with the span")
}

func TestOTelInterceptorCallStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i, _ := newOTelInterceptor(t, "grpc")

	mockCloser := transporttest.NewMockStreamCloser(ctrl)
	mockCloser.EXPECT().Context().Return(context.Background()).AnyTimes()
	mockCloser.EXPECT().Request().Return(&transport.StreamRequest{
		Meta: &transport.RequestMeta{Procedure: "test-procedure"},
	}).AnyTimes()

	clientStream, err := transport.NewClientStream(mockCloser)
	require.NoError(t, err)

	outbound := interceptortest.NewMockStreamOutboundChain(ctrl)
	outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(clientStream, nil)

	stream, err := i.CallStream(context.Background(), &transport.StreamRequest{
		Meta: &transport.RequestMeta{Procedure: "test-procedure"},
	}, outbound)
	require.NoError(t, err)
	require.NotNil(t, stream)
}

func TestOTelInterceptorCallStream_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i, sr := newOTelInterceptor(t, "grpc")

	outbound := interceptortest.NewMockStreamOutboundChain(ctrl)
	outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(
		nil, yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "stream error"),
	)

	stream, err := i.CallStream(context.Background(), &transport.StreamRequest{
		Meta: &transport.RequestMeta{Procedure: "test-procedure"},
	}, outbound)
	require.Error(t, err)
	assert.Nil(t, stream)

	// Span must be ended and marked as error even though the stream never opened.
	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status().Code)
}

// The traced wrappers must keep implementing the optional stream header
// interfaces. transport.SendStreamHeaders and ReadStreamHeaders type assert,
// so a missing method silently degrades into an Unimplemented error.
func TestOTelTracedStreamsSupportHeaders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i, _ := newOTelInterceptor(t, "grpc")
	headers := transport.NewHeaders().With("k", "v")

	t.Run("server stream sends headers", func(t *testing.T) {
		var sent transport.Headers
		serverStream, err := transport.NewServerStream(&mockServerStream{
			ctx: context.Background(),
			req: &transport.StreamRequest{
				Meta: &transport.RequestMeta{Procedure: "test-procedure"},
			},
			sendHeaders: func(h transport.Headers) error {
				sent = h
				return nil
			},
		})
		require.NoError(t, err)

		handler := transporttest.NewMockStreamHandler(ctrl)
		handler.EXPECT().HandleStream(gomock.Any()).DoAndReturn(func(s *transport.ServerStream) error {
			return s.SendHeaders(headers)
		})

		require.NoError(t, i.HandleStream(serverStream, handler))
		assert.Equal(t, headers.Items(), sent.Items())
	})

	t.Run("client stream reads headers", func(t *testing.T) {
		clientStream, err := transport.NewClientStream(&mockClientStream{
			ctx: context.Background(),
			req: &transport.StreamRequest{
				Meta: &transport.RequestMeta{Procedure: "test-procedure"},
			},
			headers: func() (transport.Headers, error) { return headers, nil },
		})
		require.NoError(t, err)

		outbound := interceptortest.NewMockStreamOutboundChain(ctrl)
		outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(clientStream, nil)

		stream, err := i.CallStream(context.Background(), &transport.StreamRequest{
			Meta: &transport.RequestMeta{Procedure: "test-procedure"},
		}, outbound)
		require.NoError(t, err)

		got, err := stream.Headers()
		require.NoError(t, err)
		assert.Equal(t, headers.Items(), got.Items())
	})
}

// A stream that terminates normally reports io.EOF, which is not a failure.
// The span must also be ended exactly once even though ReceiveMessage and
// Close both reach closeWithErr.
func TestOTelTracedClientStreamEOFIsNotAnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i, sr := newOTelInterceptor(t, "grpc")

	clientStream, err := transport.NewClientStream(&mockClientStream{
		ctx: context.Background(),
		req: &transport.StreamRequest{
			Meta: &transport.RequestMeta{Procedure: "test-procedure"},
		},
		receiveMsg: func(context.Context) (*transport.StreamMessage, error) { return nil, io.EOF },
		close:      func(context.Context) error { return nil },
	})
	require.NoError(t, err)

	outbound := interceptortest.NewMockStreamOutboundChain(ctrl)
	outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(clientStream, nil)

	stream, err := i.CallStream(context.Background(), &transport.StreamRequest{
		Meta: &transport.RequestMeta{Procedure: "test-procedure"},
	}, outbound)
	require.NoError(t, err)

	_, err = stream.ReceiveMessage(context.Background())
	assert.ErrorIs(t, err, io.EOF)
	require.NoError(t, stream.Close(context.Background()))

	spans := sr.Ended()
	require.Len(t, spans, 1, "span must be ended exactly once")
	assert.Equal(t, codes.Unset, spans[0].Status().Code, "EOF must not mark the span as failed")
}

func TestOTelTracedClientStreamRecordsRealErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i, sr := newOTelInterceptor(t, "grpc")

	clientStream, err := transport.NewClientStream(&mockClientStream{
		ctx: context.Background(),
		req: &transport.StreamRequest{
			Meta: &transport.RequestMeta{Procedure: "test-procedure"},
		},
		receiveMsg: func(context.Context) (*transport.StreamMessage, error) {
			return nil, yarpcerrors.Newf(yarpcerrors.CodeInternal, "stream failed")
		},
	})
	require.NoError(t, err)

	outbound := interceptortest.NewMockStreamOutboundChain(ctrl)
	outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(clientStream, nil)

	stream, err := i.CallStream(context.Background(), &transport.StreamRequest{
		Meta: &transport.RequestMeta{Procedure: "test-procedure"},
	}, outbound)
	require.NoError(t, err)

	_, err = stream.ReceiveMessage(context.Background())
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status().Code)
}

func TestOTelInterceptorHandle_PropagatesRemoteParent(t *testing.T) {
	tp, sr := newTestProvider(t)
	prop := testPropagator()
	i := NewOTel(OTelParams{TracerProvider: tp, Propagator: prop, Transport: "grpc"})

	// Inject a parent span into request headers.
	_, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")
	headers := make(map[string]string)
	prop.Inject(trace.ContextWithSpan(context.Background(), parentSpan), propagation.MapCarrier(headers))
	parentSpan.End()

	req := &transport.Request{
		Caller:    "upstream",
		Service:   "service",
		Procedure: "procedure",
		Headers:   transport.NewHeaders(),
	}
	for k, v := range headers {
		req.Headers = req.Headers.With(k, v)
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	handler := transporttest.NewMockUnaryHandler(ctrl)
	handler.EXPECT().Handle(gomock.Any(), req, gomock.Any()).Return(nil)

	require.NoError(t, i.Handle(context.Background(), req, &testResponseWriter{FakeResponseWriter: &transporttest.FakeResponseWriter{}}, handler))

	spans := sr.Ended()
	require.GreaterOrEqual(t, len(spans), 1)
	serverSpan := spans[len(spans)-1]
	assert.True(t, serverSpan.Parent().IsValid(), "server span must have a remote parent")
}

// Nil TracerProvider and Propagator fall back to the OTel globals, mirroring
// how the OT interceptor falls back to opentracing.GlobalTracer().
func TestOTelInterceptorDefaultsToGlobals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	i := NewOTel(OTelParams{Transport: "grpc"})

	req := &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Procedure: "procedure",
		Headers:   transport.Headers{},
	}
	rw := &testResponseWriter{FakeResponseWriter: &transporttest.FakeResponseWriter{}}

	handler := transporttest.NewMockUnaryHandler(ctrl)
	handler.EXPECT().Handle(gomock.Any(), req, rw).Return(nil)

	require.NoError(t, i.Handle(context.Background(), req, rw, handler))
}

// Span attributes must match the tags the OpenTracing interceptor set, so that
// dashboards and queries keep working across the migration.
func TestOTelSpanAttributes(t *testing.T) {
	req := &transport.Request{
		Caller:    "mycaller",
		Service:   "myservice",
		Encoding:  "raw",
		Procedure: "MyProcedure",
		Headers:   transport.Headers{},
	}

	t.Run("inbound", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		i, sr := newOTelInterceptor(t, "grpc")
		inboundReq := *req
		inboundReq.Transport = "grpc"

		handler := transporttest.NewMockUnaryHandler(ctrl)
		handler.EXPECT().Handle(gomock.Any(), &inboundReq, gomock.Any()).Return(nil)

		require.NoError(t, i.Handle(context.Background(), &inboundReq,
			&testResponseWriter{FakeResponseWriter: &transporttest.FakeResponseWriter{}}, handler))

		spans := sr.Ended()
		require.Len(t, spans, 1)
		attrs := spans[0].Attributes()

		assert.Equal(t, "MyProcedure", spans[0].Name())
		assert.True(t, hasAttr(attrs, "rpc.caller", "mycaller"))
		assert.True(t, hasAttr(attrs, "rpc.service", "myservice"))
		assert.True(t, hasAttr(attrs, "rpc.encoding", "raw"))
		assert.True(t, hasAttr(attrs, "rpc.transport", "grpc"))
		assert.True(t, hasAttr(attrs, "component", tracingComponentName))
		assert.True(t, hasAttr(attrs, "go.version", runtime.Version()))
		// The peer of an inbound call is the caller.
		assert.True(t, hasAttr(attrs, string(semconv.PeerServiceKey), "mycaller"))
	})

	t.Run("outbound", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		i, sr := newOTelInterceptor(t, "tchannel")
		outboundReq := *req

		outbound := interceptortest.NewMockUnaryOutboundChain(ctrl)
		outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(&transport.Response{}, nil)

		_, err := i.Call(context.Background(), &outboundReq, outbound)
		require.NoError(t, err)

		spans := sr.Ended()
		require.Len(t, spans, 1)
		attrs := spans[0].Attributes()

		// Outbound spans are tagged with the interceptor's transport, not the
		// request's.
		assert.True(t, hasAttr(attrs, "rpc.transport", "tchannel"))
		// The peer of an outbound call is the callee.
		assert.True(t, hasAttr(attrs, string(semconv.PeerServiceKey), "myservice"))
	})
}
