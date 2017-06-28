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

package yarpc

import "go.uber.org/yarpc/yarpcerrors"

// IsBadRequestError returns true on an error returned by RPC clients if the
// request was rejected by YARPC because it was invalid.
//
// 	res, err := client.Call(...)
// 	if yarpc.IsBadRequestError(err) {
// 		fmt.Println("invalid request:", err)
// 	}
//
// Deprecated: use yarpcerrors.IsInvalidArgument(err) instead.
func IsBadRequestError(err error) bool {
	return yarpcerrors.IsInvalidArgument(err)
}

// IsUnexpectedError returns true on an error returned by RPC clients if the
// server panicked or failed with an unhandled error.
//
// 	res, err := client.Call(...)
// 	if yarpc.IsUnexpectedError(err) {
// 		fmt.Println("internal server error:", err)
// 	}
//
// Deprecated: use yarpcerrors.IsInternal(err) instead.
func IsUnexpectedError(err error) bool {
	return yarpcerrors.IsInternal(err)
}

// IsTimeoutError returns true on an error returned by RPC clients if the given
// error is a TimeoutError.
//
// 	res, err := client.Call(...)
// 	if yarpc.IsTimeoutError(err) {
// 		fmt.Println("request timed out:", err)
// 	}
//
// Deprecated: use yarpcerrors.IsDeadlineExceeded(err).
func IsTimeoutError(err error) bool {
	return yarpcerrors.IsDeadlineExceeded(err)
}
