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
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"google.golang.org/grpc/codes"
)

var (
	// CodeToGRPCCode maps all Codes to their corresponding gRPC Code.
	CodeToGRPCCode = map[transport.Code]codes.Code{
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

	// GRPCCodeToCode maps all gRPC Codes to their corresponding Code.
	GRPCCodeToCode = map[codes.Code]transport.Code{
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
