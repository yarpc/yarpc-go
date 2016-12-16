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

// IsBadRequestError returns true if the request could not be processed
// because it was invalid.
func IsBadRequestError(err error) bool {
	_, ok := err.(errors.BadRequestError)
	return ok
}

// InboundBadRequestError builds an error which indicates that an inbound
// cannot process a request because it is a bad request.
//
// IsBadRequestError returns true for these errors.
func InboundBadRequestError(err error) error {
	return errors.HandlerBadRequestError(err)
}

// OutboundBadRequestError builds an error which indicates that an outbound
// request failed because it is invalid.
//
// Outbound implementations should use this if the remote server refuses to
// process the request.
//
// IsBadRequestError returns true for these errors.
func OutboundBadRequestError(message string) error {
	return errors.RemoteBadRequestError(message)
}

// IsUnexpectedError returns true if the server panicked or failed to process
// the request with an unhandled error.
func IsUnexpectedError(err error) bool {
	_, ok := err.(errors.UnexpectedError)
	return ok
}

// InboundUnexpectedError builds an error which indicates that an inbound
// cannot process a request due to an unexpected failure.
//
// IsUnexpectedError returns true for these errors.
func InboundUnexpectedError(err error) error {
	return errors.HandlerUnexpectedError(err)
}

// OutboundUnexpectedError builds an error which indicates that an outbound
// request failed due to an unexpected error.
//
// Outbound implementations should use this if the remote server responded
// with a message indicating that it suffered an internal failure.
//
// IsUnexpectedError returns true for these errors.
func OutboundUnexpectedError(message string) error {
	return errors.RemoteUnexpectedError(message)
}

// IsTimeoutError return true if the given error is a TimeoutError.
func IsTimeoutError(err error) bool {
	_, ok := err.(errors.TimeoutError)
	return ok
}

// OutboundTimeoutError builds an error which indicates that an outbound
// request failed due to a timeout.
//
// Outbound implementations should use this if the remote server responded
// with a message indicating that the request timed out. This should NOT be
// used if the server did not send back a response within the request TTL.
//
// IsTimeoutError returns true for these errors.
func OutboundTimeoutError(message string) error {
	return errors.RemoteTimeoutError(message)
}

// ToInboundError converts an error into an inbound-level error. Inbound
// implementations may use this with IsBadRequestError, IsUnexpectedError or
// IsTimeoutError to behave differently for the different error cases. One
// of the three functions is guaranteed to return true for an error returned
// by this function.
//
// 	err := ToInboundError("myservice", "hello", err)
// 	switch {
// 	case IsBadRequestError(err):
// 		return "User error"
// 	case IsUnexpectedError(err):
// 		return "Internal failure"
// 	case IsTimeoutError(err):
// 		return "Timeout"
// 	default:
// 		panic("Impossible!")
// 	}
func ToInboundError(service, procedure string, err error) error {
	return errors.AsHandlerError(service, procedure, err)
}
