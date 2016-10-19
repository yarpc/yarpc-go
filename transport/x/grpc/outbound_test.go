package grpc

import (
	e "errors"
	"testing"
	"time"

	"go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/transport"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestGetErrFromGRPCError(t *testing.T) {
	type testStruct struct {
		inputError    error
		inputReq      *transport.Request
		inputTTL      time.Duration
		expectedError error
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.inputError = nil
			s.expectedError = nil
			return
		}(),
		func() (s testStruct) {
			errStr := "Error Unimplemented"
			s.inputError = grpc.Errorf(codes.Unimplemented, errStr)
			s.expectedError = errors.RemoteBadRequestError(errStr)
			return
		}(),
		func() (s testStruct) {
			errStr := "Error Unexpected"
			s.inputError = grpc.Errorf(codes.Canceled, errStr)
			s.expectedError = errors.RemoteUnexpectedError(errStr)
			return
		}(),
		func() (s testStruct) {
			errStr := "Error Really Unexpected"
			s.inputError = e.New(errStr)
			s.expectedError = errors.RemoteUnexpectedError(errStr)
			return
		}(),
		func() (s testStruct) {
			service := "serv"
			procedure := "proc"
			s.inputError = grpc.Errorf(codes.DeadlineExceeded, "Doesn't matter")
			s.inputReq = &transport.Request{Service: service, Procedure: procedure}
			s.inputTTL = time.Minute
			s.expectedError = errors.ClientTimeoutError(service, procedure, time.Minute)
			return
		}(),
	}

	for _, tt := range tests {
		err := getErrFromGRPCError(tt.inputError, tt.inputReq, tt.inputTTL)

		assert.Equal(t, tt.expectedError, err)
	}
}
