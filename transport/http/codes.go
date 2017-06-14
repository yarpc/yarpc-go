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

	"go.uber.org/yarpc/api/yarpcerrors"
)

// TODO: Should we expose the maps as public variables to document the mappings?

var (
	_codeToHTTPStatusCode = map[yarpcerrors.Code]int{
		yarpcerrors.CodeOK:                 200,
		yarpcerrors.CodeCancelled:          499,
		yarpcerrors.CodeUnknown:            500,
		yarpcerrors.CodeInvalidArgument:    400,
		yarpcerrors.CodeDeadlineExceeded:   504,
		yarpcerrors.CodeNotFound:           404,
		yarpcerrors.CodeAlreadyExists:      409,
		yarpcerrors.CodePermissionDenied:   403,
		yarpcerrors.CodeResourceExhausted:  429,
		yarpcerrors.CodeFailedPrecondition: 400,
		yarpcerrors.CodeAborted:            409,
		yarpcerrors.CodeOutOfRange:         400,
		yarpcerrors.CodeUnimplemented:      501,
		yarpcerrors.CodeInternal:           500,
		yarpcerrors.CodeUnavailable:        503,
		yarpcerrors.CodeDataLoss:           500,
		yarpcerrors.CodeUnauthenticated:    401,
	}

	_httpStatusCodeToCodes = map[int][]yarpcerrors.Code{
		200: {yarpcerrors.CodeOK},
		400: {
			yarpcerrors.CodeInvalidArgument,
			yarpcerrors.CodeFailedPrecondition,
			yarpcerrors.CodeOutOfRange,
		},
		401: {yarpcerrors.CodeUnauthenticated},
		403: {yarpcerrors.CodePermissionDenied},
		404: {yarpcerrors.CodeNotFound},
		409: {
			yarpcerrors.CodeAborted,
			yarpcerrors.CodeAlreadyExists,
		},
		429: {yarpcerrors.CodeResourceExhausted},
		499: {yarpcerrors.CodeCancelled},
		500: {
			yarpcerrors.CodeUnknown,
			yarpcerrors.CodeInternal,
			yarpcerrors.CodeDataLoss,
		},
		501: {yarpcerrors.CodeUnimplemented},
		503: {yarpcerrors.CodeUnavailable},
		504: {yarpcerrors.CodeDeadlineExceeded},
	}
)

// codeToHTTPStatusCode returns the HTTP status code for the given Code,
// or error if the Code is unknown.
func codeToHTTPStatusCode(code yarpcerrors.Code) (int, error) {
	statusCode, ok := _codeToHTTPStatusCode[code]
	if !ok {
		return 0, fmt.Errorf("unknown code: %v", code)
	}
	return statusCode, nil
}

// httpStatusCodeToCodes returns the Codes that correspond to the given HTTP status
// code, or nil if no Codes correspond to the given HTTP status code.
func httpStatusCodeToCodes(httpStatusCode int) []yarpcerrors.Code {
	codes, ok := _httpStatusCodeToCodes[httpStatusCode]
	if !ok {
		return nil
	}
	c := make([]yarpcerrors.Code, len(codes))
	copy(c, codes)
	return c
}

// httpStatusCodeToBestCode does a best-effort conversion from the given HTTP status
// code to a Code.
//
// If one Code maps to the given HTTP status code, that Code is returned.
// If more than one Code maps to the given HTTP status Code, one Code is returned.
// If the Code is >=400 and < 500, yarpcerrors.CodeInvalidArgument is returned.
// Else, yarpcerrors.CodeUnknown is returned.
func httpStatusCodeToBestCode(httpStatusCode int) yarpcerrors.Code {
	codes, ok := _httpStatusCodeToCodes[httpStatusCode]
	if !ok || len(codes) == 0 {
		if httpStatusCode >= 400 && httpStatusCode < 500 {
			return yarpcerrors.CodeInvalidArgument
		}
		return yarpcerrors.CodeUnknown
	}
	return codes[0]
}
