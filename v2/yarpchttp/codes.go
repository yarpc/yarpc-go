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

package yarpchttp

import "go.uber.org/yarpc/v2/yarpcerror"

var (
	// _codeToStatusCode maps all Codes to their corresponding HTTP status code.
	_codeToStatusCode = map[yarpcerror.Code]int{
		yarpcerror.CodeOK:                 200,
		yarpcerror.CodeCancelled:          499,
		yarpcerror.CodeUnknown:            500,
		yarpcerror.CodeInvalidArgument:    400,
		yarpcerror.CodeDeadlineExceeded:   504,
		yarpcerror.CodeNotFound:           404,
		yarpcerror.CodeAlreadyExists:      409,
		yarpcerror.CodePermissionDenied:   403,
		yarpcerror.CodeResourceExhausted:  429,
		yarpcerror.CodeFailedPrecondition: 400,
		yarpcerror.CodeAborted:            409,
		yarpcerror.CodeOutOfRange:         400,
		yarpcerror.CodeUnimplemented:      501,
		yarpcerror.CodeInternal:           500,
		yarpcerror.CodeUnavailable:        503,
		yarpcerror.CodeDataLoss:           500,
		yarpcerror.CodeUnauthenticated:    401,
	}

	// _statusCodeToCodes maps HTTP status codes to a slice of their corresponding Codes.
	_statusCodeToCodes = map[int][]yarpcerror.Code{
		200: {yarpcerror.CodeOK},
		400: {
			yarpcerror.CodeInvalidArgument,
			yarpcerror.CodeFailedPrecondition,
			yarpcerror.CodeOutOfRange,
		},
		401: {yarpcerror.CodeUnauthenticated},
		403: {yarpcerror.CodePermissionDenied},
		404: {yarpcerror.CodeNotFound},
		409: {
			yarpcerror.CodeAborted,
			yarpcerror.CodeAlreadyExists,
		},
		429: {yarpcerror.CodeResourceExhausted},
		499: {yarpcerror.CodeCancelled},
		500: {
			yarpcerror.CodeUnknown,
			yarpcerror.CodeInternal,
			yarpcerror.CodeDataLoss,
		},
		501: {yarpcerror.CodeUnimplemented},
		503: {yarpcerror.CodeUnavailable},
		504: {yarpcerror.CodeDeadlineExceeded},
	}
)

// statusCodeToBestCode does a best-effort conversion from the given HTTP status
// code to a Code.
//
// If one Code maps to the given HTTP status code, that Code is returned.
// If more than one Code maps to the given HTTP status Code, one Code is returned.
// If the Code is >=400 and < 500, yarpcerror.CodeInvalidArgument is returned.
// Else, yarpcerror.CodeUnknown is returned.
func statusCodeToBestCode(statusCode int) yarpcerror.Code {
	codes, ok := _statusCodeToCodes[statusCode]
	if !ok || len(codes) == 0 {
		if statusCode >= 400 && statusCode < 500 {
			return yarpcerror.CodeInvalidArgument
		}
		return yarpcerror.CodeUnknown
	}
	return codes[0]
}
