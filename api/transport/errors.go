// Copyright (c) 2016 Uber Technologies, Inc.
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

package transport

import "go.uber.org/yarpc/internal/errors"

// BadRequestError builds an error which indicates a bad request with the
// given error as the reason.
//
// Transport implementations treat BadRequestErrors as user errors.
func BadRequestError(err error) error {
	return errors.HandlerBadRequestError(err)
}

// IsBadRequestError returns true if the request could not be processed
// because it was invalid.
func IsBadRequestError(err error) bool {
	_, ok := err.(errors.BadRequestError)
	return ok
}

// IsUnexpectedError returns true if the server panicked or failed to process
// the request with an unhandled error.
func IsUnexpectedError(err error) bool {
	_, ok := err.(errors.UnexpectedError)
	return ok
}

// IsTimeoutError return true if the given error is a TimeoutError.
func IsTimeoutError(err error) bool {
	_, ok := err.(errors.TimeoutError)
	return ok
}
