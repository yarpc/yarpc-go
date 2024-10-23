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
func (rw *testResponseWriter) GetApplicationErrorMeta() *transport.ApplicationErrorMeta {
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

			// Use testResponseWriter to simulate ExtendedResponseWriter
			responseWriter := &testResponseWriter{
				FakeResponseWriter: &transporttest.FakeResponseWriter{},
			}

			if tt.isApplicationError {
				responseWriter.SetApplicationError()
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
			expectedErrorType: "invalid-argument",
		},
		{
			name:              "application error in response",
			response:          &transport.Response{ApplicationError: true},
			callError:         nil,
			expectedErrorTag:  true,
			expectedErrorType: "application_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			outbound := transporttest.NewMockUnaryOutbound(gomock.NewController(t))
			outbound.EXPECT().
				Call(gomock.Any(), req).
				Return(tt.response, tt.callError)

			res, err := interceptor.Call(context.Background(), req, outbound)

			if tt.callError != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response, res, "Response mismatch")
			}

			finishedSpans := tracer.FinishedSpans()
			assert.Len(t, finishedSpans, 1, "Expected one span to be finished.")
		})
	}
}

func TestUpdateSpanWithError(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		expectedErrorType string
	}{
		{
			name:              "known YARPC error",
			err:               yarpcerrors.Newf(yarpcerrors.CodeInternal, "known error"),
			expectedErrorType: yarpcerrors.CodeInternal.String(),
		},
		{
			name:              "unknown internal YARPC error",
			err:               fmt.Errorf("random unknown error"),
			expectedErrorType: "unknown",
		},
		{
			name:              "nil error",
			err:               nil,
			expectedErrorType: "",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			tracer := mocktracer.New()
			span := tracer.StartSpan("test")

			err := updateSpanWithErrorDetails(span, nil, false, nil, tt.err)
			span.Finish()

			finishedSpans := tracer.FinishedSpans()
			require.Len(t, finishedSpans, 1)

			spanTags := finishedSpans[0].Tags()

			if tt.expectedErrorType != "" {
				assert.Equal(t, tt.err, err, "Expected error to be returned")
				assert.Equal(t, tt.expectedErrorType, spanTags["error.type"], "Expected error.type to be set correctly")
			} else {
				assert.Nil(t, spanTags["error.type"], "Expected no error.type tag to be set")
			}
		})
	}
}
