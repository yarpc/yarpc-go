// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcgrpc

import (
	"go.uber.org/yarpc/v2/yarpcerror"
	"google.golang.org/grpc/codes"
)

var (
	// _codeToGRPCCode maps all Codes to their corresponding gRPC Code.
	_codeToGRPCCode = map[yarpcerror.Code]codes.Code{
		yarpcerror.CodeOK:                 codes.OK,
		yarpcerror.CodeCancelled:          codes.Canceled,
		yarpcerror.CodeUnknown:            codes.Unknown,
		yarpcerror.CodeInvalidArgument:    codes.InvalidArgument,
		yarpcerror.CodeDeadlineExceeded:   codes.DeadlineExceeded,
		yarpcerror.CodeNotFound:           codes.NotFound,
		yarpcerror.CodeAlreadyExists:      codes.AlreadyExists,
		yarpcerror.CodePermissionDenied:   codes.PermissionDenied,
		yarpcerror.CodeResourceExhausted:  codes.ResourceExhausted,
		yarpcerror.CodeFailedPrecondition: codes.FailedPrecondition,
		yarpcerror.CodeAborted:            codes.Aborted,
		yarpcerror.CodeOutOfRange:         codes.OutOfRange,
		yarpcerror.CodeUnimplemented:      codes.Unimplemented,
		yarpcerror.CodeInternal:           codes.Internal,
		yarpcerror.CodeUnavailable:        codes.Unavailable,
		yarpcerror.CodeDataLoss:           codes.DataLoss,
		yarpcerror.CodeUnauthenticated:    codes.Unauthenticated,
	}

	// _grpcCodeToCode maps all gRPC Codes to their corresponding Code.
	_grpcCodeToCode = map[codes.Code]yarpcerror.Code{
		codes.OK:                 yarpcerror.CodeOK,
		codes.Canceled:           yarpcerror.CodeCancelled,
		codes.Unknown:            yarpcerror.CodeUnknown,
		codes.InvalidArgument:    yarpcerror.CodeInvalidArgument,
		codes.DeadlineExceeded:   yarpcerror.CodeDeadlineExceeded,
		codes.NotFound:           yarpcerror.CodeNotFound,
		codes.AlreadyExists:      yarpcerror.CodeAlreadyExists,
		codes.PermissionDenied:   yarpcerror.CodePermissionDenied,
		codes.ResourceExhausted:  yarpcerror.CodeResourceExhausted,
		codes.FailedPrecondition: yarpcerror.CodeFailedPrecondition,
		codes.Aborted:            yarpcerror.CodeAborted,
		codes.OutOfRange:         yarpcerror.CodeOutOfRange,
		codes.Unimplemented:      yarpcerror.CodeUnimplemented,
		codes.Internal:           yarpcerror.CodeInternal,
		codes.Unavailable:        yarpcerror.CodeUnavailable,
		codes.DataLoss:           yarpcerror.CodeDataLoss,
		codes.Unauthenticated:    yarpcerror.CodeUnauthenticated,
	}
)
