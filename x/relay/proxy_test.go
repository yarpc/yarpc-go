package relay

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
)

func TestUnaryProxy(t *testing.T) {
	testCases := []struct {
		name                     string
		requestBody              []byte
		requestHeaders           transport.Headers
		responseBody             []byte
		responseHeaders          transport.Headers
		responseApplicationError bool
		responseErr              error
		expectedHandlerErr       error
	}{
		{
			name:                     "successful proxy",
			requestBody:              []byte("this is the request"),
			requestHeaders:           transport.NewHeaders().With("key", "val"),
			responseBody:             []byte("this is the response"),
			responseHeaders:          transport.NewHeaders().With("respKey", "respVal"),
			responseApplicationError: false,
		},
		{
			name:                     "empty response body",
			requestBody:              []byte("this is the request"),
			requestHeaders:           transport.NewHeaders().With("key", "val"),
			responseBody:             []byte(nil),
			responseHeaders:          transport.NewHeaders().With("respKey", "respVal"),
			responseApplicationError: false,
		},
		{
			name:                     "application error",
			requestBody:              []byte("this is the request"),
			requestHeaders:           transport.NewHeaders().With("key", "val"),
			responseBody:             []byte("this is the response"),
			responseHeaders:          transport.NewHeaders().With("respKey", "respVal"),
			responseApplicationError: true,
		},
		{
			name:                     "propagate client error",
			requestBody:              []byte("this is the request"),
			requestHeaders:           transport.NewHeaders().With("key", "val"),
			responseBody:             []byte("this is the response"),
			responseHeaders:          transport.NewHeaders().With("respKey", "respVal"),
			responseApplicationError: false,
			responseErr:              errors.New("failure"),
			expectedHandlerErr:       errors.New("failure"),
		},
		{
			name:                     "empty response body",
			requestBody:              []byte("this is the request"),
			requestHeaders:           transport.NewHeaders().With("key", "val"),
			responseBody:             nil,
			responseHeaders:          transport.NewHeaders().With("respKey", "respVal"),
			responseApplicationError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx := context.Background()
			giveReq := &transport.Request{
				Caller:          "caller-" + tc.name,
				Service:         "service-" + tc.name,
				Procedure:       "procedure-" + tc.name,
				Encoding:        transport.Encoding("encoding-" + tc.name),
				Body:            bytes.NewBuffer(tc.requestBody),
				Headers:         tc.requestHeaders,
				RoutingKey:      "test",
				RoutingDelegate: "test",
			}
			wantReq := &transport.Request{
				Caller:    "caller-" + tc.name,
				Service:   "service-" + tc.name,
				Procedure: "procedure-" + tc.name,
				Encoding:  transport.Encoding("encoding-" + tc.name),
				Body:      bytes.NewBuffer(tc.requestBody),
				Headers:   tc.requestHeaders,
			}
			resp := &transport.Response{
				Headers:          tc.responseHeaders,
				Body:             ioutil.NopCloser(bytes.NewBuffer(tc.responseBody)),
				ApplicationError: tc.responseApplicationError,
			}
			resw := new(transporttest.FakeResponseWriter)

			o := transporttest.NewMockUnaryOutbound(mockCtrl)
			o.EXPECT().Call(ctx, wantReq).Return(resp, tc.responseErr)

			handler := UnaryProxyHandler(o)
			err := handler.Handle(ctx, giveReq, resw)

			if tc.expectedHandlerErr != nil {
				assert.Equal(t, tc.expectedHandlerErr, err)
				return
			}
			assert.Nil(t, err, "expected no handler error")
			assert.Equal(t, tc.responseBody, resw.Body.Bytes())
			assert.Equal(t, tc.responseHeaders, resw.Headers)
			assert.Equal(t, tc.responseApplicationError, resw.IsApplicationError)
		})
	}
}

func TestOnewayProxy(t *testing.T) {
	testCases := []struct {
		name               string
		requestBody        []byte
		requestHeaders     transport.Headers
		responseErr        error
		expectedHandlerErr error
	}{
		{
			name:           "successful oneway proxy",
			requestBody:    []byte("this is the request"),
			requestHeaders: transport.NewHeaders().With("key", "val"),
		},
		{
			name:               "propagate error",
			requestBody:        []byte("this is the request"),
			requestHeaders:     transport.NewHeaders().With("key", "val"),
			responseErr:        errors.New("failure"),
			expectedHandlerErr: errors.New("failure"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx := context.Background()
			giveReq := &transport.Request{
				Caller:          "caller-" + tc.name,
				Service:         "service-" + tc.name,
				Procedure:       "procedure-" + tc.name,
				Encoding:        transport.Encoding("encoding-" + tc.name),
				Body:            bytes.NewBuffer(tc.requestBody),
				Headers:         tc.requestHeaders,
				RoutingDelegate: "test",
				RoutingKey:      "test",
			}
			wantReq := &transport.Request{
				Caller:    "caller-" + tc.name,
				Service:   "service-" + tc.name,
				Procedure: "procedure-" + tc.name,
				Encoding:  transport.Encoding("encoding-" + tc.name),
				Body:      bytes.NewBuffer(tc.requestBody),
				Headers:   tc.requestHeaders,
			}

			o := transporttest.NewMockOnewayOutbound(mockCtrl)
			o.EXPECT().CallOneway(ctx, wantReq).Return(time.Now(), tc.responseErr)

			handler := OnewayProxyHandler(o)
			err := handler.HandleOneway(ctx, giveReq)

			assert.Equal(t, tc.expectedHandlerErr, err)
		})
	}
}
