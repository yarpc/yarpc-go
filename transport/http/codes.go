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

package http

import (
	"fmt"

	"go.uber.org/yarpc"
)

var (
	_codeToHTTPStatusCode = map[yarpc.Code]int{
		yarpc.CodeOK:                 200,
		yarpc.CodeCancelled:          499,
		yarpc.CodeUnknown:            500,
		yarpc.CodeInvalidArgument:    400,
		yarpc.CodeDeadlineExceeded:   504,
		yarpc.CodeNotFound:           404,
		yarpc.CodeAlreadyExists:      409,
		yarpc.CodePermissionDenied:   403,
		yarpc.CodeResourceExhausted:  429,
		yarpc.CodeFailedPrecondition: 400,
		yarpc.CodeAborted:            409,
		yarpc.CodeOutOfRange:         400,
		yarpc.CodeUnimplemented:      501,
		yarpc.CodeInternal:           500,
		yarpc.CodeUnavailable:        503,
		yarpc.CodeDataLoss:           500,
		yarpc.CodeUnauthenticated:    401,
	}
)

// CodeToHTTPStatusCode returns the HTTP status code for the given Code.
func CodeToHTTPStatusCode(code yarpc.Code) (int, error) {
	statusCode, ok := _codeToHTTPStatusCode[code]
	if !ok {
		return 0, fmt.Errorf("unknown code: %v", code)
	}
	return statusCode, nil
}
