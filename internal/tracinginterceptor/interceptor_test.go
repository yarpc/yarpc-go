// Copyright (c) 2024 Uber Technologies, Inc.
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
	"fmt"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/transport/tchannel/tracing"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/yarpcerrors"
)

type testResponseWriter struct {
	*transporttest.FakeResponseWriter
	isAppError   bool
	appErrorMeta *transport.ApplicationErrorMeta
	responseSize int
}

// Ensure testResponseWriter implements ExtendedResponseWriter
var _ transport.ExtendedResponseWriter = (*testResponseWriter)(nil)

// Override SetApplicationError to track the application error state
func (rw *testResponseWriter) SetApplicationError() {
	rw.isAppError = true
}

// Implement IsApplicationError for the interceptor
func (rw *testResponseWriter) IsApplicationError() bool {
	return rw.isAppError
}

// Implement SetApplicationErrorMeta for the interceptor
func (rw *testResponseWriter) SetApplicationErrorMeta(meta *transport.ApplicationErrorMeta) {
	rw.appErrorMeta = meta
}

// Implement GetApplicationErrorMeta for the interceptor
func (rw *testResponseWriter) ApplicationErrorMeta() *transport.ApplicationErrorMeta {
	return rw.appErrorMeta
}

// Implement GetApplicationError to satisfy the ExtendedResponseWriter interface
func (rw *testResponseWriter) GetApplicationError() bool {
	return rw.isAppError
}

// Implement Write to capture response size
func (rw *testResponseWriter) Write(p []byte) (int, error) {
	rw.responseSize += len(p)
	return len(p), nil
}

// Implement ResponseSize to retrieve the response size
func (rw *testResponseWriter) ResponseSize() int {
	return rw.responseSize
}

// Table-driven test for Unary Inbound Interceptor's Handle method
func TestInterceptorHandle(t *testing.T) {
	tests := []struct {
		name               string
		handlerError       error
		isApplicationError bool
		expectedErrorTag   bool
		expectedErrorType  string
		useNonExtendedRW   bool // Add flag to test non-ExtendedResponseWriter case
	}{
		{
			name:               "successful handle with no errors",
			handlerError:       nil,
			isApplicationError: false,
			expectedErrorTag:   false,
		},
		{
			name:               "handler returns an error",
			handlerError:       yarpcerrors.Newf(yarpcerrors.CodeInternal, "handler error"),
			isApplicationError: false,
			expectedErrorTag:   true,
			expectedErrorType:  "internal",
		},
		{
			name:               "application error",
			handlerError:       nil,
			isApplicationError: true,
			expectedErrorTag:   true,
			expectedErrorType:  "application_error",
		},
		{
			name:               "non-ExtendedResponseWriter",
			handlerError:       nil,
			isApplicationError: false,
			expectedErrorTag:   false,
			useNonExtendedRW:   true, // This case uses a non-ExtendedResponseWriter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tracer := mocktracer.New()
			interceptor := New(Params{
				Tracer:    tracer,
				Transport: "http",
			})

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			var responseWriter transport.ResponseWriter
			if tt.useNonExtendedRW {
				// Use FakeResponseWriter without ExtendedResponseWriter implementation
				responseWriter = &transporttest.FakeResponseWriter{}
			} else {
				// Use testResponseWriter to simulate ExtendedResponseWriter
				rw := &testResponseWriter{FakeResponseWriter: &transporttest.FakeResponseWriter{}}
				if tt.isApplicationError {
					rw.SetApplicationError()
				}
				responseWriter = rw
			}

			handler := transporttest.NewMockUnaryHandler(ctrl)
			handler.EXPECT().
				Handle(gomock.Any(), req, responseWriter).
				Return(tt.handlerError)

			err := interceptor.Handle(context.Background(), req, responseWriter, handler)

			if tt.handlerError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			finishedSpans := tracer.FinishedSpans()
			require.Len(t, finishedSpans, 1, "Expected one span to be finished.")
		})
	}
}

// Table-driven test for Unary Outbound Interceptor's Call method
func TestInterceptorCall(t *testing.T) {
	tests := []struct {
		name              string
		response          *transport.Response
		callError         error
		expectedErrorTag  bool
		expectedErrorType string
		expectedAppCode   *int
		expectedAppName   string
		injectError       bool // Add flag to simulate Inject error
	}{
		{
			name:             "successful call with no errors",
			response:         &transport.Response{},
			callError:        nil,
			expectedErrorTag: false,
		},
		{
			name:              "call returns an error",
			response:          nil,
			callError:         yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "call error"),
			expectedErrorTag:  true,
			expectedErrorType: yarpcerrors.CodeInvalidArgument.String(),
		},
		{
			name:              "application error in response",
			response:          &transport.Response{ApplicationError: true},
			callError:         nil,
			expectedErrorTag:  true,
			expectedErrorType: "application_error",
		},
		{
			name:             "inject error",
			response:         &transport.Response{},
			callError:        nil,
			expectedErrorTag: false,
			injectError:      true, // This case simulates an Inject error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tracer := mocktracer.New()
			interceptor := New(Params{
				Tracer:    tracer,
				Transport: "http",
			})

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			outbound := transporttest.NewMockUnaryOutbound(ctrl)
			outbound.EXPECT().
				Call(gomock.Any(), req).
				Return(tt.response, tt.callError)

			// Mocking Inject to return an error
			if tt.injectError {
				tracer := mocktracer.New()
				span := tracer.StartSpan("test-span")
				err := tracer.Inject(span.Context(), opentracing.TextMap, transport.Headers{})
				require.Error(t, err, "Inject error expected")
			}

			res, err := interceptor.Call(context.Background(), req, outbound)

			// Assert errors
			if tt.callError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response, res, "Response mismatch")
			}

			// Check finished spans
			finishedSpans := tracer.FinishedSpans()
			assert.Len(t, finishedSpans, 1, "Expected one span to be finished.")
			spanTags := finishedSpans[0].Tags()

			// Check error tags and application error meta
			if tt.expectedErrorTag {
				assert.Equal(t, tt.expectedErrorType, spanTags[errorCodeTag], "Expected error.code to be set correctly")

				if tt.response != nil && tt.response.ApplicationError && tt.response.ApplicationErrorMeta != nil {
					if tt.expectedAppCode != nil {
						assert.Equal(t, *tt.expectedAppCode, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set")
					}
					if tt.expectedAppName != "" {
						assert.Equal(t, tt.expectedAppName, spanTags["error.name"], "Expected error.name to be set")
					}
				}
			} else {
				assert.Nil(t, spanTags[errorCodeTag], "Expected no error.code tag to be set")
			}
		})
	}
}

func TestUpdateSpanWithErrorDetails(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		isApplicationError bool
		appErrorMeta       *transport.ApplicationErrorMeta
		expectedErrorType  string
		expectedAppCode    *int
		expectedAppName    string
	}{
		{
			name:               "known YARPC error",
			err:                yarpcerrors.Newf(yarpcerrors.CodeInternal, "known error"),
			isApplicationError: false,
			appErrorMeta:       nil,
			expectedErrorType:  yarpcerrors.CodeInternal.String(),
		},
		{
			name:               "random unknown error",
			err:                fmt.Errorf("random unknown error"),
			isApplicationError: false,
			appErrorMeta:       nil,
			expectedErrorType:  "unknown",
		},
		{
			name:               "application error with metadata",
			err:                nil,
			isApplicationError: true,
			appErrorMeta: &transport.ApplicationErrorMeta{
				Code: (*yarpcerrors.Code)(intPtr(500)),
				Name: "InternalError",
			},
			expectedErrorType: "application_error",
			expectedAppCode:   intPtr(500),
			expectedAppName:   "InternalError",
		},
		{
			name:               "application error without metadata",
			err:                nil,
			isApplicationError: true,
			appErrorMeta:       nil,
			expectedErrorType:  "application_error",
		},
		{
			name:               "nil error and no application error",
			err:                nil,
			isApplicationError: false,
			appErrorMeta:       nil,
			expectedErrorType:  "",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			tracer := mocktracer.New()
			span := tracer.StartSpan("test")

			err := updateSpanWithErrorDetails(span, tt.isApplicationError, tt.appErrorMeta, tt.err)
			span.Finish()

			finishedSpans := tracer.FinishedSpans()
			require.Len(t, finishedSpans, 1)

			spanTags := finishedSpans[0].Tags()

			// Check if error is returned and error.type tag is set correctly
			if tt.expectedErrorType != "" {
				assert.Equal(t, tt.err, err, "Expected error to be returned")
				assert.Equal(t, tt.expectedErrorType, spanTags[errorCodeTag], "Expected error.code to be set correctly")

				if tt.expectedErrorType == "application_error" && tt.appErrorMeta != nil {
					// Check application error code and name tags
					if tt.expectedAppCode != nil {
						assert.Equal(t, *tt.expectedAppCode, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set")
					}
					if tt.expectedAppName != "" {
						assert.Equal(t, tt.expectedAppName, spanTags["error.name"], "Expected error.name to be set")
					}
				}
			} else {
				// No error.type tag should be set
				assert.Nil(t, spanTags[errorCodeTag], "Expected no error.code tag to be set")
			}
		})
	}
}

// Table-driven test for Oneway Inbound Interceptor's HandleOneway method
func TestInterceptorHandleOneway(t *testing.T) {
	tests := []struct {
		name              string
		handlerError      error
		expectedErrorTag  bool
		expectedErrorType string
	}{
		{
			name:             "successful handle oneway with no errors",
			handlerError:     nil,
			expectedErrorTag: false,
		},
		{
			name:              "handle oneway returns an error",
			handlerError:      yarpcerrors.Newf(yarpcerrors.CodeInternal, "handler error"),
			expectedErrorTag:  true,
			expectedErrorType: "internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tracer := mocktracer.New()
			interceptor := New(Params{
				Tracer:    tracer,
				Transport: "http",
			})

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			handler := transporttest.NewMockOnewayHandler(ctrl)
			handler.EXPECT().
				HandleOneway(gomock.Any(), req).
				Return(tt.handlerError)

			err := interceptor.HandleOneway(context.Background(), req, handler)

			if tt.handlerError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			finishedSpans := tracer.FinishedSpans()
			require.Len(t, finishedSpans, 1, "Expected one span to be finished.")
			spanTags := finishedSpans[0].Tags()

			// Check error tags
			if tt.expectedErrorTag {
				assert.Equal(t, tt.expectedErrorType, spanTags[errorCodeTag], "Expected error.code to be set correctly")
			} else {
				assert.Nil(t, spanTags[errorCodeTag], "Expected no error.code tag to be set")
			}
		})
	}
}

// Table-driven test for Oneway Outbound Interceptor's CallOneway method
func TestInterceptorCallOneway(t *testing.T) {
	tests := []struct {
		name              string
		callError         error
		expectedErrorTag  bool
		expectedErrorType string
	}{
		{
			name:             "successful call oneway with no errors",
			callError:        nil,
			expectedErrorTag: false,
		},
		{
			name:              "call oneway returns an error",
			callError:         yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "call error"),
			expectedErrorTag:  true,
			expectedErrorType: yarpcerrors.CodeInvalidArgument.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tracer := mocktracer.New()
			interceptor := New(Params{
				Tracer:    tracer,
				Transport: "http",
			})

			req := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "procedure",
				Headers:   transport.Headers{},
			}

			outbound := transporttest.NewMockOnewayOutbound(ctrl)
			outbound.EXPECT().
				CallOneway(gomock.Any(), req).
				Return(nil, tt.callError) // Return nil for Ack

			_, err := interceptor.CallOneway(context.Background(), req, outbound)

			// Assert errors
			if tt.callError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Check finished spans
			finishedSpans := tracer.FinishedSpans()
			assert.Len(t, finishedSpans, 1, "Expected one span to be finished.")
			spanTags := finishedSpans[0].Tags()

			// Check error tags
			if tt.expectedErrorTag {
				assert.Equal(t, tt.expectedErrorType, spanTags[errorCodeTag], "Expected error.code to be set correctly")
			} else {
				assert.Nil(t, spanTags[errorCodeTag], "Expected no error.code tag to be set")
			}
		})
	}
}

// TestGetPropagationCarrier verifies that getPropagationCarrier returns the correct
// carrier type based on the specified transport. For "tchannel" transport, it should
// return a tracing.HeadersCarrier, while for other transports (e.g., "http"), it
// should return an opentracing.TextMapCarrier.
func TestGetPropagationCarrier(t *testing.T) {
	headers := map[string]string{"key": "value"}

	// Test with "tchannel" transport
	carrier := getPropagationCarrier(headers, "tchannel")
	_, isHeadersCarrier := carrier.(tracing.HeadersCarrier)
	assert.True(t, isHeadersCarrier, "Expected HeadersCarrier for tchannel transport")

	// Test with "http" transport (default case)
	carrier = getPropagationCarrier(headers, "http")
	_, isTextMapCarrier := carrier.(opentracing.TextMapCarrier)
	assert.True(t, isTextMapCarrier, "Expected TextMapCarrier for non-tchannel transport")
}

// TestGetPropagationFormat verifies that getPropagationFormat returns the correct
// format based on the specified transport. For "tchannel" transport, it should
// return opentracing.TextMap, while for other transports (e.g., "http"), it should
// return opentracing.HTTPHeaders.
func TestGetPropagationFormat(t *testing.T) {
	// Test with "tchannel" transport
	format := getPropagationFormat("tchannel")
	assert.Equal(t, opentracing.TextMap, format, "Expected TextMap format for tchannel transport")

	// Test with "http" transport (default case)
	format = getPropagationFormat("http")
	assert.Equal(t, opentracing.HTTPHeaders, format, "Expected HTTPHeaders format for non-tchannel transport")
}

// Helper function to create pointers to int values
func intPtr(i int) *int {
	return &i
}
