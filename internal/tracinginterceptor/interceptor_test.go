//package tracinginterceptor
//
//import (
//	"context"
//	"github.com/golang/mock/gomock"
//	"github.com/opentracing/opentracing-go/mocktracer"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//	"go.uber.org/yarpc/api/transport"
//	"go.uber.org/yarpc/api/transport/transporttest"
//	"go.uber.org/yarpc/yarpcerrors"
//	"testing"
//)
//
//// Table-driven test for Unary Inbound Interceptor's Handle method
//func TestInterceptorHandle(t *testing.T) {
//	tests := []struct {
//		name               string
//		handlerError       error
//		isApplicationError bool
//		appErrorMeta       *transport.ApplicationErrorMeta
//		expectedErrorTag   bool
//		expectedErrorType  string
//		expectedErrorCode  *int
//		expectedDetails    string
//	}{
//		{
//			name:               "successful handle with no errors",
//			handlerError:       nil,
//			isApplicationError: false,
//			expectedErrorTag:   false,
//		},
//		{
//			name:               "handler returns an error",
//			handlerError:       yarpcerrors.Newf(yarpcerrors.CodeInternal, "handler error"),
//			isApplicationError: false,
//			expectedErrorTag:   true,
//			expectedErrorType:  "internal",
//		},
//		{
//			name:               "application error with metadata",
//			handlerError:       nil,
//			isApplicationError: true,
//			appErrorMeta:       &transport.ApplicationErrorMeta{Code: (*yarpcerrors.Code)(intPtr(123)), Details: "something went wrong"},
//			expectedErrorTag:   true,
//			expectedErrorType:  "application_error",
//			expectedErrorCode:  intPtr(123),
//			expectedDetails:    "something went wrong",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ctrl := gomock.NewController(t)
//			defer ctrl.Finish()
//
//			tracer := mocktracer.New()
//			interceptor := New(Params{
//				Tracer:    tracer,
//				Transport: "http",
//			})
//
//			req := &transport.Request{
//				Caller:    "caller",
//				Service:   "service",
//				Procedure: "procedure",
//				Headers:   transport.Headers{},
//			}
//
//			wrappedWriter := newWriter(&transporttest.FakeResponseWriter{})
//			defer wrappedWriter.free()
//
//			if tt.isApplicationError {
//				wrappedWriter.SetApplicationError()
//				wrappedWriter.SetApplicationErrorMeta(tt.appErrorMeta)
//			}
//
//			handler := transporttest.NewMockUnaryHandler(ctrl)
//			handler.EXPECT().
//				Handle(gomock.Any(), req, gomock.Any()).
//				Return(tt.handlerError)
//
//			err := interceptor.Handle(context.Background(), req, wrappedWriter, handler)
//
//			if tt.handlerError != nil {
//				require.Error(t, err)
//			} else {
//				require.NoError(t, err)
//			}
//
//			finishedSpans := tracer.FinishedSpans()
//			assert.Len(t, finishedSpans, 1, "Expected one span to be finished.")
//
//			span := finishedSpans[0]
//
//			if tt.expectedErrorTag {
//				tag, ok := span.Tag("error.type").(string)
//				assert.True(t, ok, "Expected error.type tag to be set.")
//				assert.Equal(t, tt.expectedErrorType, tag, "Mismatch in error.type tag")
//				if tt.expectedErrorCode != nil {
//					assert.Equal(t, *tt.expectedErrorCode, span.Tag("application_error_code"), "Mismatch in application_error_code tag")
//				}
//				if tt.expectedDetails != "" {
//					assert.Equal(t, tt.expectedDetails, span.Tag("application_error_details"), "Mismatch in application_error_details tag")
//				}
//			} else {
//				assert.Nil(t, span.Tag("error"), "Error tag should not be set.")
//			}
//		})
//	}
//}
//
//// // Table-driven test for Unary Outbound Interceptor's Call method
////
////	func TestInterceptorCall(t *testing.T) {
////		tests := []struct {
////			name              string
////			callError         error
////			response          *transport.Response
////			expectedErrorTag  bool
////			expectedErrorType string
////			expectedErrorCode *int
////			expectedDetails   string
////		}{
////			{
////				name:             "successful call with no errors",
////				callError:        nil,
////				response:         &transport.Response{},
////				expectedErrorTag: false,
////			},
////			{
////				name:              "call returns an error",
////				callError:         yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "call error"),
////				response:          nil,
////				expectedErrorTag:  true,
////				expectedErrorType: "invalid-argument",
////			},
////			{
////				name:              "application error in response",
////				callError:         nil,
////				response:          &transport.Response{ApplicationError: true, ApplicationErrorMeta: &transport.ApplicationErrorMeta{Code: (*yarpcerrors.Code)(intPtr(456)), Details: "application error details"}},
////				expectedErrorTag:  true,
////				expectedErrorType: "application_error",
////				expectedErrorCode: intPtr(456),
////				expectedDetails:   "application error details",
////			},
////		}
////
////		for _, tt := range tests {
////			t.Run(tt.name, func(t *testing.T) {
////				ctrl := gomock.NewController(t)
////				defer ctrl.Finish()
////
////				tracer := mocktracer.New()
////				interceptor := New(Params{
////					Tracer:    tracer,
////					Transport: "http",
////				})
////
////				req := &transport.Request{
////					Caller:    "caller",
////					Service:   "service",
////					Procedure: "procedure",
////					Headers:   transport.Headers{},
////				}
////
////				outbound := transporttest.NewMockUnaryOutbound(ctrl)
////				outbound.EXPECT().
////					Call(gomock.Any(), req).
////					Return(tt.response, tt.callError)
////
////				res, err := interceptor.Call(context.Background(), req, outbound)
////
////				if tt.callError != nil {
////					require.Error(t, err)
////				} else {
////					require.NoError(t, err)
////					assert.Equal(t, tt.response, res, "Response mismatch")
////				}
////
////				finishedSpans := tracer.FinishedSpans()
////				assert.Len(t, finishedSpans, 1, "Expected one span to be finished.")
////
////				span := finishedSpans[0]
////
////				if tt.expectedErrorTag {
////					tag, ok := span.Tag("error.type").(string)
////					assert.True(t, ok, "Expected error.type tag to be set.")
////					assert.Equal(t, tt.expectedErrorType, tag, "Mismatch in error.type tag")
////					if tt.expectedErrorCode != nil {
////						assert.Equal(t, *tt.expectedErrorCode, span.Tag("application_error_code"), "Mismatch in application_error_code tag")
////					}
////					if tt.expectedDetails != "" {
////						assert.Equal(t, tt.expectedDetails, span.Tag("application_error_details"), "Mismatch in application_error_details tag")
////					}
////				} else {
////					assert.Nil(t, span.Tag("error"), "Error tag should not be set.")
////				}
////			})
////		}
////	}
//func intPtr(i int) *int {
//	return &i
//}
