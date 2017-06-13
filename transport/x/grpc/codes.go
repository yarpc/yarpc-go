// Copyright (c) 2017 Uber Technologies, Inc.
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

package grpc

import (
	"fmt"

	"go.uber.org/yarpc/api/yarpcerrors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	_codeToGRPCCode = map[yarpcerrors.Code]codes.Code{
		yarpcerrors.CodeOK:                 codes.OK,
		yarpcerrors.CodeCancelled:          codes.Canceled,
		yarpcerrors.CodeUnknown:            codes.Unknown,
		yarpcerrors.CodeInvalidArgument:    codes.InvalidArgument,
		yarpcerrors.CodeDeadlineExceeded:   codes.DeadlineExceeded,
		yarpcerrors.CodeNotFound:           codes.NotFound,
		yarpcerrors.CodeAlreadyExists:      codes.AlreadyExists,
		yarpcerrors.CodePermissionDenied:   codes.PermissionDenied,
		yarpcerrors.CodeResourceExhausted:  codes.ResourceExhausted,
		yarpcerrors.CodeFailedPrecondition: codes.FailedPrecondition,
		yarpcerrors.CodeAborted:            codes.Aborted,
		yarpcerrors.CodeOutOfRange:         codes.OutOfRange,
		yarpcerrors.CodeUnimplemented:      codes.Unimplemented,
		yarpcerrors.CodeInternal:           codes.Internal,
		yarpcerrors.CodeUnavailable:        codes.Unavailable,
		yarpcerrors.CodeDataLoss:           codes.DataLoss,
		yarpcerrors.CodeUnauthenticated:    codes.Unauthenticated,
	}

	_grpcCodeToCode = map[codes.Code]yarpcerrors.Code{
		codes.OK:                 yarpcerrors.CodeOK,
		codes.Canceled:           yarpcerrors.CodeCancelled,
		codes.Unknown:            yarpcerrors.CodeUnknown,
		codes.InvalidArgument:    yarpcerrors.CodeInvalidArgument,
		codes.DeadlineExceeded:   yarpcerrors.CodeDeadlineExceeded,
		codes.NotFound:           yarpcerrors.CodeNotFound,
		codes.AlreadyExists:      yarpcerrors.CodeAlreadyExists,
		codes.PermissionDenied:   yarpcerrors.CodePermissionDenied,
		codes.ResourceExhausted:  yarpcerrors.CodeResourceExhausted,
		codes.FailedPrecondition: yarpcerrors.CodeFailedPrecondition,
		codes.Aborted:            yarpcerrors.CodeAborted,
		codes.OutOfRange:         yarpcerrors.CodeOutOfRange,
		codes.Unimplemented:      yarpcerrors.CodeUnimplemented,
		codes.Internal:           yarpcerrors.CodeInternal,
		codes.Unavailable:        yarpcerrors.CodeUnavailable,
		codes.DataLoss:           yarpcerrors.CodeDataLoss,
		codes.Unauthenticated:    yarpcerrors.CodeUnauthenticated,
	}

	// TODO: Don't want to expose this in yarpcerrors, what to do?
	_codeToErrorConstructor = map[yarpcerrors.Code]func(string, ...interface{}) error{
		yarpcerrors.CodeCancelled:          yarpcerrors.CancelledErrorf,
		yarpcerrors.CodeUnknown:            yarpcerrors.UnknownErrorf,
		yarpcerrors.CodeInvalidArgument:    yarpcerrors.InvalidArgumentErrorf,
		yarpcerrors.CodeDeadlineExceeded:   yarpcerrors.DeadlineExceededErrorf,
		yarpcerrors.CodeNotFound:           yarpcerrors.NotFoundErrorf,
		yarpcerrors.CodeAlreadyExists:      yarpcerrors.AlreadyExistsErrorf,
		yarpcerrors.CodePermissionDenied:   yarpcerrors.PermissionDeniedErrorf,
		yarpcerrors.CodeResourceExhausted:  yarpcerrors.ResourceExhaustedErrorf,
		yarpcerrors.CodeFailedPrecondition: yarpcerrors.FailedPreconditionErrorf,
		yarpcerrors.CodeAborted:            yarpcerrors.AbortedErrorf,
		yarpcerrors.CodeOutOfRange:         yarpcerrors.OutOfRangeErrorf,
		yarpcerrors.CodeUnimplemented:      yarpcerrors.UnimplementedErrorf,
		yarpcerrors.CodeInternal:           yarpcerrors.InternalErrorf,
		yarpcerrors.CodeUnavailable:        yarpcerrors.UnavailableErrorf,
		yarpcerrors.CodeDataLoss:           yarpcerrors.DataLossErrorf,
		yarpcerrors.CodeUnauthenticated:    yarpcerrors.UnauthenticatedErrorf,
	}
)

// codeToGRPCCode returns the gRPC Code for the given Code,
// or error if the Code is unknown.
func codeToGRPCCode(code yarpcerrors.Code) (codes.Code, error) {
	grpcCode, ok := _codeToGRPCCode[code]
	if !ok {
		return 0, fmt.Errorf("unknown code: %v", code)
	}
	return grpcCode, nil
}

// grpcCodeToCode returns the Code for the given gRPC Code,
// or error if the gRPC Code is unknown.
func grpcCodeToCode(grpcCode codes.Code) (yarpcerrors.Code, error) {
	code, ok := _grpcCodeToCode[grpcCode]
	if !ok {
		return 0, fmt.Errorf("unknown gRPC code: %v", grpcCode)
	}
	return code, nil
}

func grpcErrorToYARPCError(err error) error {
	code, ok := _grpcCodeToCode[grpc.Code(err)]
	if !ok {
		code = yarpcerrors.CodeUnknown
	}
	errorConstructor, ok := _codeToErrorConstructor[code]
	if !ok {
		errorConstructor = yarpcerrors.UnknownErrorf
	}
	return errorConstructor(grpc.ErrorDesc(err))
}
