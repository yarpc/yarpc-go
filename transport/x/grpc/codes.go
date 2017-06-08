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

	"go.uber.org/yarpc"

	"google.golang.org/grpc/codes"
)

var (
	_codeToGRPCCode = map[yarpc.Code]codes.Code{
		yarpc.CodeOK:                 codes.OK,
		yarpc.CodeCancelled:          codes.Canceled,
		yarpc.CodeUnknown:            codes.Unknown,
		yarpc.CodeInvalidArgument:    codes.InvalidArgument,
		yarpc.CodeDeadlineExceeded:   codes.DeadlineExceeded,
		yarpc.CodeNotFound:           codes.NotFound,
		yarpc.CodeAlreadyExists:      codes.AlreadyExists,
		yarpc.CodePermissionDenied:   codes.PermissionDenied,
		yarpc.CodeResourceExhausted:  codes.ResourceExhausted,
		yarpc.CodeFailedPrecondition: codes.FailedPrecondition,
		yarpc.CodeAborted:            codes.Aborted,
		yarpc.CodeOutOfRange:         codes.OutOfRange,
		yarpc.CodeUnimplemented:      codes.Unimplemented,
		yarpc.CodeInternal:           codes.Internal,
		yarpc.CodeUnavailable:        codes.Unavailable,
		yarpc.CodeDataLoss:           codes.DataLoss,
		yarpc.CodeUnauthenticated:    codes.Unauthenticated,
	}

	_grpcCodeToCode = map[codes.Code]yarpc.Code{
		codes.OK:                 yarpc.CodeOK,
		codes.Canceled:           yarpc.CodeCancelled,
		codes.Unknown:            yarpc.CodeUnknown,
		codes.InvalidArgument:    yarpc.CodeInvalidArgument,
		codes.DeadlineExceeded:   yarpc.CodeDeadlineExceeded,
		codes.NotFound:           yarpc.CodeNotFound,
		codes.AlreadyExists:      yarpc.CodeAlreadyExists,
		codes.PermissionDenied:   yarpc.CodePermissionDenied,
		codes.ResourceExhausted:  yarpc.CodeResourceExhausted,
		codes.FailedPrecondition: yarpc.CodeFailedPrecondition,
		codes.Aborted:            yarpc.CodeAborted,
		codes.OutOfRange:         yarpc.CodeOutOfRange,
		codes.Unimplemented:      yarpc.CodeUnimplemented,
		codes.Internal:           yarpc.CodeInternal,
		codes.Unavailable:        yarpc.CodeUnavailable,
		codes.DataLoss:           yarpc.CodeDataLoss,
		codes.Unauthenticated:    yarpc.CodeUnauthenticated,
	}
)

// codeToGRPCCode returns the gRPC Code for the given Code,
// or error if the Code is unknown.
func codeToGRPCCode(code yarpc.Code) (codes.Code, error) {
	grpcCode, ok := _codeToGRPCCode[code]
	if !ok {
		return 0, fmt.Errorf("unknown code: %v", code)
	}
	return grpcCode, nil
}

// grpcCodeToCode returns the Code for the given gRPC Code,
// or error if the gRPC Code is unknown.
func grpcCodeToCode(grpcCode codes.Code) (yarpc.Code, error) {
	code, ok := _grpcCodeToCode[grpcCode]
	if !ok {
		return 0, fmt.Errorf("unknown gRPC code: %v", grpcCode)
	}
	return code, nil
}
