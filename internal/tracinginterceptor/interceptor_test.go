// Copyright (c) 2025 Uber Technologies, Inc.
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
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/interceptor/interceptortest"
	"go.uber.org/yarpc/transport/tchannel/tracing"
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
		expectedErrorCode *int
		expectedErrorType string
		expectedAppCode   *int
		expectedAppName   string
		injectError       bool
	}{
		{
			name:             "successful call with no errors",
			response:         &transport.Response{},
			callError:        nil,
			expectedErrorTag: false,
		},
		{
			name:             "call returns an error",
			response:         nil,
			callError:        yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "call error"),
			expectedErrorTag: true,
			expectedErrorCode: func() *int {
				code := int(yarpcerrors.CodeInvalidArgument)
				return &code
			}(),
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
			injectError:      true,
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

			outbound := interceptortest.NewMockUnaryOutboundChain(ctrl)
			outbound.EXPECT().Next(gomock.Any(), req).
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
				if tt.expectedErrorCode != nil {
					assert.Equal(t, *tt.expectedErrorCode, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set correctly")
				}
				if tt.response != nil && tt.response.ApplicationError && tt.response.ApplicationErrorMeta != nil {
					if tt.expectedAppCode != nil {
						assert.Equal(t, *tt.expectedAppCode, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set")
					}
					if tt.expectedAppName != "" {
						assert.Equal(t, tt.expectedAppName, spanTags["error.name"], "Expected error.name to be set")
					}
				}
			} else {
				assert.Nil(t, spanTags[rpcStatusCodeTag], "Expected no rpc.yarpc.status_code tag to be set")
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
		expectedErrorCode  interface{} // Allows flexibility for int and string cases
		expectedAppCode    *int
		expectedAppName    string
	}{
		{
			name:               "known YARPC error",
			err:                yarpcerrors.Newf(yarpcerrors.CodeInternal, "known error"),
			isApplicationError: false,
			appErrorMeta:       nil,
			expectedErrorCode: func() int {
				return int(yarpcerrors.CodeInternal)
			}(),
		},
		{
			name:               "random unknown error",
			err:                fmt.Errorf("random unknown error"),
			isApplicationError: false,
			appErrorMeta:       nil,
			expectedErrorCode: func() int {
				return int(yarpcerrors.CodeUnknown)
			}(),
		},
		{
			name:               "application error with metadata",
			err:                nil,
			isApplicationError: true,
			appErrorMeta: &transport.ApplicationErrorMeta{
				Code: (*yarpcerrors.Code)(func() *int { code := 500; return &code }()),
				Name: "InternalError",
			},
			expectedErrorCode: func() int { return 500 }(),
			expectedAppCode: func() *int {
				code := 500
				return &code
			}(),
			expectedAppName: "InternalError",
		},
		{
			name:               "application error without metadata",
			err:                nil,
			isApplicationError: true,
			appErrorMeta:       nil,
			expectedErrorCode:  "application_error", // As set in the function without appErrorMeta
		},
		{
			name:               "nil error and no application error",
			err:                nil,
			isApplicationError: false,
			appErrorMeta:       nil,
			expectedErrorCode:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := mocktracer.New()
			span := tracer.StartSpan("test")

			err := updateSpanWithErrorDetails(span, tt.isApplicationError, tt.appErrorMeta, tt.err)
			span.Finish()

			finishedSpans := tracer.FinishedSpans()
			require.Len(t, finishedSpans, 1)

			spanTags := finishedSpans[0].Tags()

			// Check if error is returned and error code tag is set correctly
			if tt.expectedErrorCode != nil {
				assert.Equal(t, tt.err, err, "Expected error to be returned")

				// Use type assertion to handle different types of expectedErrorCode
				switch v := tt.expectedErrorCode.(type) {
				case int:
					assert.Equal(t, v, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set correctly")
				case string:
					assert.Equal(t, v, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set correctly")
				}

				if tt.isApplicationError && tt.appErrorMeta != nil {
					// Check application error code and name tags
					if tt.expectedAppCode != nil {
						assert.Equal(t, *tt.expectedAppCode, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set")
					}
					if tt.expectedAppName != "" {
						assert.Equal(t, tt.expectedAppName, spanTags["error.name"], "Expected error.name to be set")
					}
				}
			} else {
				// No error code tag should be set
				assert.Nil(t, spanTags[rpcStatusCodeTag], "Expected no rpc.yarpc.status_code tag to be set")
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
		expectedErrorCode *int
	}{
		{
			name:             "successful handle oneway with no errors",
			handlerError:     nil,
			expectedErrorTag: false,
		},
		{
			name:             "handle oneway returns an error",
			handlerError:     yarpcerrors.Newf(yarpcerrors.CodeInternal, "handler error"),
			expectedErrorTag: true,
			expectedErrorCode: func() *int {
				code := int(yarpcerrors.CodeInternal)
				return &code
			}(),
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
				if tt.expectedErrorCode != nil {
					assert.Equal(t, *tt.expectedErrorCode, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set correctly")
				}
			} else {
				assert.Nil(t, spanTags[rpcStatusCodeTag], "Expected no rpc.yarpc.status_code tag to be set")
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
		expectedErrorCode *int
	}{
		{
			name:             "successful call oneway with no errors",
			callError:        nil,
			expectedErrorTag: false,
		},
		{
			name:             "call oneway returns an error",
			callError:        yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "call error"),
			expectedErrorTag: true,
			expectedErrorCode: func() *int {
				code := int(yarpcerrors.CodeInvalidArgument)
				return &code
			}(),
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

			outbound := interceptortest.NewMockOnewayOutboundChain(ctrl)
			outbound.EXPECT().
				Next(gomock.Any(), req).
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
				if tt.expectedErrorCode != nil {
					assert.Equal(t, *tt.expectedErrorCode, spanTags[rpcStatusCodeTag], "Expected rpc.yarpc.status_code to be set correctly")
				}
			} else {
				assert.Nil(t, spanTags[rpcStatusCodeTag], "Expected no rpc.yarpc.status_code tag to be set")
			}
		})
	}
}

func TestInterceptorHandleStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tracer := mocktracer.New()
	interceptor := New(Params{
		Tracer:    tracer,
		Transport: "http",
	})

	// Mock the server stream and stream handler
	mockStream := transporttest.NewMockStream(ctrl)
	mockStream.EXPECT().Context().Return(context.Background()).AnyTimes()
	mockStream.EXPECT().Request().Return(&transport.StreamRequest{Meta: &transport.RequestMeta{Procedure: "test-procedure"}}).AnyTimes()

	serverStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err)

	handler := transporttest.NewMockStreamHandler(ctrl)
	handler.EXPECT().HandleStream(gomock.Any()).Return(nil)

	// Call HandleStream and validate behavior
	err = interceptor.HandleStream(serverStream, handler)
	require.NoError(t, err)

	// Check that exactly one span has finished and its tags
	finishedSpans := tracer.FinishedSpans()
	require.Len(t, finishedSpans, 1, "Expected one span to be finished.")
	spanTags := finishedSpans[0].Tags()

	// Verify error.tag is absent for success cases
	assert.Nil(t, spanTags["error.type"], "Expected no error.type tag to be set")
}

func TestInterceptorHandleStream_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tracer := mocktracer.New()
	interceptor := New(Params{
		Tracer:    tracer,
		Transport: "http",
	})

	// Mock the server stream and stream handler
	mockStream := transporttest.NewMockStream(ctrl)
	mockStream.EXPECT().Context().Return(context.Background()).AnyTimes()
	mockStream.EXPECT().Request().Return(&transport.StreamRequest{Meta: &transport.RequestMeta{Procedure: "test-procedure"}}).AnyTimes()

	serverStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err)

	handler := transporttest.NewMockStreamHandler(ctrl)
	handler.EXPECT().HandleStream(gomock.Any()).Return(yarpcerrors.Newf(yarpcerrors.CodeInternal, "handler error"))

	// Call HandleStream and capture any error
	err = interceptor.HandleStream(serverStream, handler)
	require.Error(t, err)

	// Check that one span has finished and contains error details
	finishedSpans := tracer.FinishedSpans()
	require.Len(t, finishedSpans, 1, "Expected one span to be finished.")
	spanTags := finishedSpans[0].Tags()

	// Verify the rpcStatusCodeTag is set correctly for the internal error
	assert.Equal(t, int(yarpcerrors.CodeInternal), spanTags[rpcStatusCodeTag], "Expected rpcStatusCodeTag to be set to 'internal'")
}

func TestInterceptorCallStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tracer := mocktracer.New()
	interceptor := New(Params{
		Tracer:    tracer,
		Transport: "http",
	})

	// Mock the client stream and outbound
	mockStream := transporttest.NewMockStreamCloser(ctrl)
	mockStream.EXPECT().Context().Return(context.Background()).AnyTimes()
	mockStream.EXPECT().Request().Return(&transport.StreamRequest{Meta: &transport.RequestMeta{Procedure: "test-procedure"}}).AnyTimes()

	clientStream, err := transport.NewClientStream(mockStream)
	require.NoError(t, err)

	outbound := interceptortest.NewMockStreamOutboundChain(ctrl)
	outbound.EXPECT().Next(gomock.Any(), gomock.Any()).Return(clientStream, nil)

	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{Procedure: "test-procedure"},
	}

	// Call CallStream and validate behavior
	stream, err := interceptor.CallStream(context.Background(), req, outbound)
	require.NoError(t, err)
	require.NotNil(t, stream)
}

func TestInterceptorCallStream_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tracer := mocktracer.New()
	interceptor := New(Params{
		Tracer:    tracer,
		Transport: "http",
	})

	// Mock the client stream and outbound stream
	mockStreamCloser := transporttest.NewMockStreamCloser(ctrl)
	clientStream, err := transport.NewClientStream(mockStreamCloser)
	require.NoError(t, err)

	outbound := interceptortest.NewMockStreamOutboundChain(ctrl)
	outbound.EXPECT().
		Next(gomock.Any(), gomock.Any()).
		Return(clientStream, yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "call error"))

	// Set up the request
	ctx := context.Background()
	req := &transport.StreamRequest{Meta: &transport.RequestMeta{Procedure: "test-procedure"}}

	// Call CallStream and expect an error
	_, err = interceptor.CallStream(ctx, req, outbound)
	require.Error(t, err)
}

// TestGetPropagationCarrier verifies the getPropagationCarrier returns the correct
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

func TestInterceptor_HandleStream_Errors(t *testing.T) {
	tracer := mocktracer.New()
	interceptor := New(Params{
		Tracer:    tracer,
		Transport: "test",
	})

	// Test error handling stream with a valid mock stream
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := transporttest.NewMockStream(ctrl)
	mockStream.EXPECT().Context().Return(context.Background()).AnyTimes()
	mockStream.EXPECT().Request().Return(&transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Procedure: "test-procedure",
			Headers:   transport.NewHeaders(),
		},
	}).AnyTimes()
	serverStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err)

	handler := &mockStreamHandler{
		handleStream: func(stream *transport.ServerStream) error {
			return errors.New("handler error")
		},
	}
	err = interceptor.HandleStream(serverStream, handler)
	assert.Error(t, err)
	assert.Equal(t, "handler error", err.Error())
}

func TestInterceptor_CallStream_Errors(t *testing.T) {
	tracer := mocktracer.New()
	interceptor := New(Params{
		Tracer:    tracer,
		Transport: "test",
	})
	ctx := context.Background()

	// Test error calling stream
	chain := &testStreamOutboundChain{
		next: func(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
			return nil, errors.New("chain error")
		},
	}
	_, err := interceptor.CallStream(ctx, &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Procedure: "test-procedure",
			Headers:   transport.NewHeaders(),
		},
	}, chain)
	assert.Error(t, err)
	assert.Equal(t, "chain error", err.Error())
}

type mockStream struct {
	ctx         context.Context
	req         *transport.StreamRequest
	sendMsg     func(context.Context, *transport.StreamMessage) error
	receiveMsg  func(context.Context) (*transport.StreamMessage, error)
	sendHeaders func(transport.Headers) error
}

func (m *mockStream) Context() context.Context          { return m.ctx }
func (m *mockStream) Request() *transport.StreamRequest { return m.req }
func (m *mockStream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	return m.sendMsg(ctx, msg)
}
func (m *mockStream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	return m.receiveMsg(ctx)
}
func (m *mockStream) SendHeaders(headers transport.Headers) error {
	return m.sendHeaders(headers)
}

type mockStreamCloser struct {
	*mockStream
	closeFunc func(context.Context) error
}

func (m *mockStreamCloser) Close(ctx context.Context) error {
	return m.closeFunc(ctx)
}

type mockStreamHandler struct {
	handleStream func(*transport.ServerStream) error
}

func (m *mockStreamHandler) HandleStream(stream *transport.ServerStream) error {
	return m.handleStream(stream)
}

type testStreamOutboundChain struct {
	next func(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error)
}

func (c *testStreamOutboundChain) Next(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	return c.next(ctx, req)
}
func (c *testStreamOutboundChain) Outbound() transport.Outbound { return nil }

type mockOutbound struct {
	transport.Outbound
	startStream func(context.Context, *transport.StreamRequest) (*transport.ClientStream, error)
}

func (m *mockOutbound) StartStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	return m.startStream(ctx, request)
}

func TestInterceptor(t *testing.T) {
	tracer := mocktracer.New()
	interceptor := New(Params{
		Tracer:    tracer,
		Transport: "test",
	})

	t.Run("CallStream", func(t *testing.T) {
		ctx := context.Background()
		req := &transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Caller:    "test-caller",
				Service:   "test-service",
				Procedure: "test-procedure",
			},
		}

		// Test successful stream creation
		outbound := &mockOutbound{
			startStream: func(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
				stream := &mockStreamCloser{
					mockStream: &mockStream{
						ctx:         ctx,
						req:         request,
						sendMsg:     func(ctx context.Context, msg *transport.StreamMessage) error { return nil },
						receiveMsg:  func(ctx context.Context) (*transport.StreamMessage, error) { return &transport.StreamMessage{}, nil },
						sendHeaders: func(h transport.Headers) error { return nil },
					},
					closeFunc: func(ctx context.Context) error { return nil },
				}
				wrapper, err := transport.NewClientStream(stream)
				if err != nil {
					return nil, err
				}
				return wrapper, nil
			},
		}

		chain := &testStreamOutboundChain{
			next: func(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
				clientStream, err := outbound.StartStream(ctx, req)
				if err != nil {
					return nil, err
				}
				return clientStream, nil
			},
		}

		stream, err := interceptor.CallStream(ctx, req, chain)
		assert.NoError(t, err)
		assert.NotNil(t, stream)

		// Test error in stream creation
		outbound.startStream = func(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
			return nil, errors.New("stream error")
		}

		chainErr := &testStreamOutboundChain{
			next: func(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
				_, err := outbound.StartStream(ctx, req)
				return nil, err
			},
		}

		stream, err = interceptor.CallStream(ctx, req, chainErr)
		assert.Error(t, err)
		assert.Equal(t, "stream error", err.Error())
		assert.Nil(t, stream)
	})

	t.Run("HandleStream", func(t *testing.T) {
		ctx := context.Background()
		req := &transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Caller:    "test-caller",
				Service:   "test-service",
				Procedure: "test-procedure",
			},
		}

		// Test successful stream handling
		serverStream := &mockStream{
			ctx:         ctx,
			req:         req,
			sendMsg:     func(ctx context.Context, msg *transport.StreamMessage) error { return nil },
			receiveMsg:  func(ctx context.Context) (*transport.StreamMessage, error) { return &transport.StreamMessage{}, nil },
			sendHeaders: func(h transport.Headers) error { return nil },
		}

		wrapper, err := transport.NewServerStream(serverStream)
		assert.NoError(t, err)

		handler := &mockStreamHandler{
			handleStream: func(stream *transport.ServerStream) error { return nil },
		}

		err = interceptor.HandleStream(wrapper, handler)
		assert.NoError(t, err)

		// Test error in stream handling
		handler.handleStream = func(stream *transport.ServerStream) error { return errors.New("handle error") }

		err = interceptor.HandleStream(wrapper, handler)
		assert.Error(t, err)
		assert.Equal(t, "handle error", err.Error())
	})
}

func TestTracedStreamMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test tracedClientStream methods
	t.Run("tracedClientStream", func(t *testing.T) {
		// Create a mock client stream
		mockStream := transporttest.NewMockStreamCloser(ctrl)
		ctx := context.Background()
		req := &transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Procedure: "test-procedure",
				Headers:   transport.NewHeaders(),
			},
		}

		mockStream.EXPECT().Context().Return(ctx).AnyTimes()
		mockStream.EXPECT().Request().Return(req).AnyTimes()
		mockStream.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()

		clientStream, err := transport.NewClientStream(mockStream)
		require.NoError(t, err)

		// Create a traced client stream
		tracer := mocktracer.New()
		span := tracer.StartSpan("test-span")
		tracedStream := &tracedClientStream{
			clientStream: clientStream,
			span:         span,
		}

		// Test Context()
		assert.Equal(t, ctx, tracedStream.Context())

		// Test Request()
		assert.Equal(t, req, tracedStream.Request())
	})

	// Test tracedServerStream methods
	t.Run("tracedServerStream", func(t *testing.T) {
		// Create a mock server stream
		mockStream := transporttest.NewMockStream(ctrl)
		ctx := context.Background()
		req := &transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Procedure: "test-procedure",
				Headers:   transport.NewHeaders(),
			},
		}

		mockStream.EXPECT().Context().Return(ctx).AnyTimes()
		mockStream.EXPECT().Request().Return(req).AnyTimes()

		serverStream, err := transport.NewServerStream(mockStream)
		require.NoError(t, err)

		// Create a traced server stream without enriched context
		tracer := mocktracer.New()
		span := tracer.StartSpan("test-span")
		tracedStream := &tracedServerStream{
			serverStream: serverStream,
			span:         span,
		}

		// Test Context() - should return server stream's context when ctx is nil
		assert.Equal(t, ctx, tracedStream.Context())

		// Test Request()
		assert.Equal(t, req, tracedStream.Request())

		// Test Context() with enriched context
		type contextKey string
		enrichedCtx := context.WithValue(ctx, contextKey("test-key"), "test-value")
		tracedStreamWithCtx := &tracedServerStream{
			serverStream: serverStream,
			span:         span,
			ctx:          enrichedCtx,
		}

		// Should return the enriched context
		assert.Equal(t, enrichedCtx, tracedStreamWithCtx.Context())
	})
}

func TestInterceptorHandleStream_ContextPropagation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tracer := mocktracer.New()
	interceptor := New(Params{
		Tracer:    tracer,
		Transport: "http",
	})

	// Mock the server stream with minimal headers
	baseCtx := context.Background()
	mockStream := transporttest.NewMockStream(ctrl)
	mockStream.EXPECT().Context().Return(baseCtx).AnyTimes()
	mockStream.EXPECT().Request().Return(&transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Procedure: "test-procedure",
			Headers:   transport.NewHeaders(),
			Caller:    "test-caller",
			Service:   "test-service",
		},
	}).AnyTimes()

	serverStream, err := transport.NewServerStream(mockStream)
	require.NoError(t, err)

	// Create a handler that captures the context
	var capturedCtx context.Context
	handler := transporttest.NewMockStreamHandler(ctrl)
	handler.EXPECT().HandleStream(gomock.Any()).DoAndReturn(func(s *transport.ServerStream) error {
		capturedCtx = s.Context()
		return nil
	})

	// Call HandleStream
	err = interceptor.HandleStream(serverStream, handler)
	require.NoError(t, err)

	// Verify the context was enriched with tracing span
	require.NotNil(t, capturedCtx, "Expected context to be captured")

	// The key assertion: verify that the context contains the span
	span := opentracing.SpanFromContext(capturedCtx)
	require.NotNil(t, span, "Expected span to be present in the context - this is critical for baggage propagation")

	// Verify the context is different from the base context (it was enriched)
	assert.NotEqual(t, baseCtx, capturedCtx, "Expected context to be enriched with span, not the original base context")
}
