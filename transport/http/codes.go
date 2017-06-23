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
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
)

var (
	// CodeToStatusCode maps all Codes to their corresponding HTTP status code.
	CodeToStatusCode = map[transport.Code]int{
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

	// StatusCodeToCodes maps HTTP status codes to a slice of their corresponding Codes.
	StatusCodeToCodes = map[int][]transport.Code{
		200: {yarpc.CodeOK},
		400: {
			yarpc.CodeInvalidArgument,
			yarpc.CodeFailedPrecondition,
			yarpc.CodeOutOfRange,
		},
		401: {yarpc.CodeUnauthenticated},
		403: {yarpc.CodePermissionDenied},
		404: {yarpc.CodeNotFound},
		409: {
			yarpc.CodeAborted,
			yarpc.CodeAlreadyExists,
		},
		429: {yarpc.CodeResourceExhausted},
		499: {yarpc.CodeCancelled},
		500: {
			yarpc.CodeUnknown,
			yarpc.CodeInternal,
			yarpc.CodeDataLoss,
		},
		501: {yarpc.CodeUnimplemented},
		503: {yarpc.CodeUnavailable},
		504: {yarpc.CodeDeadlineExceeded},
	}
)

// StatusCodeToBestCode does a best-effort conversion from the given HTTP status
// code to a Code.
//
// If one Code maps to the given HTTP status code, that Code is returned.
// If more than one Code maps to the given HTTP status Code, one Code is returned.
// If the Code is >=400 and < 500, yarpc.CodeInvalidArgument is returned.
// Else, yarpc.CodeUnknown is returned.
func StatusCodeToBestCode(statusCode int) transport.Code {
	codes, ok := StatusCodeToCodes[statusCode]
	if !ok || len(codes) == 0 {
		if statusCode >= 400 && statusCode < 500 {
			return yarpc.CodeInvalidArgument
		}
		return yarpc.CodeUnknown
	}
	return codes[0]
}
