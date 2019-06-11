// Copyright (c) 2019 Uber Technologies, Inc.
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

package protobuf

import (
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc/codes"
)

var (
	// _codeToGRPCCode maps all Codes to their corresponding gRPC Code.
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
)
