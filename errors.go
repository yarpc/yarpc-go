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

import "go.uber.org/yarpc/api/transport"

// IsBadRequestError returns true on an error returned by RPC clients if the
// request was rejected by YARPC because it was invalid.
//
// 	res, err := client.Call(...)
// 	if yarpc.IsBadRequestError(err) {
// 		fmt.Println("invalid request:", err)
// 	}
//
// Deprecated: use IsInvalidArgument(err) instead.
func IsBadRequestError(err error) bool {
	return IsInvalidArgument(err)
}

// IsUnexpectedError returns true on an error returned by RPC clients if the
// server panicked or failed with an unhandled error.
//
// 	res, err := client.Call(...)
// 	if yarpc.IsUnexpectedError(err) {
// 		fmt.Println("internal server error:", err)
// 	}
//
// Deprecated: use IsInternal(err) instead.
func IsUnexpectedError(err error) bool {
	return IsInternal(err)
}

// IsTimeoutError returns true on an error returned by RPC clients if the given
// error is a TimeoutError.
//
// 	res, err := client.Call(...)
// 	if yarpc.IsTimeoutError(err) {
// 		fmt.Println("request timed out:", err)
// 	}
//
// Deprecated: use IsDeadlineExceeded(err).
func IsTimeoutError(err error) bool {
	return IsDeadlineExceeded(err)
}

// IsYARPCError returns true if the given error is a non-nil YARPC error.
//
// This is equivalent to ErrorCode(err) != CodeOK.
func IsYARPCError(err error) bool {
	return transport.IsYARPCError(err)
}

// ToYARPCError converts the given error into a YARPC error. This function
// returns nil if the error is nil.
//
// If the error is already a YARPC error, it will be returned as-is.
// Otherwise it will be converted into an error with CodeUnknown.
func ToYARPCError(err error) error {
	return transport.ToYARPCError(err)
}

// ErrorCode returns the Code for the given error, or CodeOK if the given
// error is not a YARPC error.
func ErrorCode(err error) transport.Code {
	return transport.ErrorCode(err)
}

// ErrorName returns the name for the given error, or "" if the given
// error is not a YARPC error created with NamedErrorf that has a non-empty name.
func ErrorName(err error) string {
	return transport.ErrorName(err)
}

// ErrorMessage returns the message for the given error, or "" if the given
// error is not a YARPC error or the YARPC error had no message.
func ErrorMessage(err error) string {
	return transport.ErrorMessage(err)
}

// NamedErrorf returns a new YARPC error with code CodeUnknown and the given
// name.
//
// This should be used for user-defined errors.
//
// The name must only contain lowercase letters from a-z and dashes (-), and
// cannot start or end in a dash. If the name is something else, an error with
// code CodeInternal will be returned.
func NamedErrorf(name string, format string, args ...interface{}) error {
	return transport.NamedErrorf(name, format, args...)
}

// CancelledErrorf returns a new yarpc error with code CodeCancelled.
func CancelledErrorf(format string, args ...interface{}) error {
	return transport.CancelledErrorf(format, args...)
}

// UnknownErrorf returns a new yarpc error with code CodeUnknown.
func UnknownErrorf(format string, args ...interface{}) error {
	return transport.UnknownErrorf(format, args...)
}

// InvalidArgumentErrorf returns a new yarpc error with code CodeInvalidArgument.
func InvalidArgumentErrorf(format string, args ...interface{}) error {
	return transport.InvalidArgumentErrorf(format, args...)
}

// DeadlineExceededErrorf returns a new yarpc error with code CodeDeadlineExceeded.
func DeadlineExceededErrorf(format string, args ...interface{}) error {
	return transport.DeadlineExceededErrorf(format, args...)
}

// NotFoundErrorf returns a new yarpc error with code CodeNotFound.
func NotFoundErrorf(format string, args ...interface{}) error {
	return transport.NotFoundErrorf(format, args...)
}

// AlreadyExistsErrorf returns a new yarpc error with code CodeAlreadyExists.
func AlreadyExistsErrorf(format string, args ...interface{}) error {
	return transport.AlreadyExistsErrorf(format, args...)
}

// PermissionDeniedErrorf returns a new yarpc error with code CodePermissionDenied.
func PermissionDeniedErrorf(format string, args ...interface{}) error {
	return transport.PermissionDeniedErrorf(format, args...)
}

// ResourceExhaustedErrorf returns a new yarpc error with code CodeResourceExhausted.
func ResourceExhaustedErrorf(format string, args ...interface{}) error {
	return transport.ResourceExhaustedErrorf(format, args...)
}

// FailedPreconditionErrorf returns a new yarpc error with code CodeFailedPrecondition.
func FailedPreconditionErrorf(format string, args ...interface{}) error {
	return transport.FailedPreconditionErrorf(format, args...)
}

// AbortedErrorf returns a new yarpc error with code CodeAborted.
func AbortedErrorf(format string, args ...interface{}) error {
	return transport.AbortedErrorf(format, args...)
}

// OutOfRangeErrorf returns a new yarpc error with code CodeOutOfRange.
func OutOfRangeErrorf(format string, args ...interface{}) error {
	return transport.OutOfRangeErrorf(format, args...)
}

// UnimplementedErrorf returns a new yarpc error with code CodeUnimplemented.
func UnimplementedErrorf(format string, args ...interface{}) error {
	return transport.UnimplementedErrorf(format, args...)
}

// InternalErrorf returns a new yarpc error with code CodeInternal.
func InternalErrorf(format string, args ...interface{}) error {
	return transport.InternalErrorf(format, args...)
}

// UnavailableErrorf returns a new yarpc error with code CodeUnavailable.
func UnavailableErrorf(format string, args ...interface{}) error {
	return transport.UnavailableErrorf(format, args...)
}

// DataLossErrorf returns a new yarpc error with code CodeDataLoss.
func DataLossErrorf(format string, args ...interface{}) error {
	return transport.DataLossErrorf(format, args...)
}

// UnauthenticatedErrorf returns a new yarpc error with code CodeUnauthenticated.
func UnauthenticatedErrorf(format string, args ...interface{}) error {
	return transport.UnauthenticatedErrorf(format, args...)
}

// IsCancelled returns true if ErrorCode(err) == CodeCancelled.
func IsCancelled(err error) bool {
	return transport.IsCancelled(err)
}

// IsUnknown returns true if ErrorCode(err) == CodeUnknown.
func IsUnknown(err error) bool {
	return transport.IsUnknown(err)
}

// IsInvalidArgument returns true if ErrorCode(err) == CodeInvalidArgument.
func IsInvalidArgument(err error) bool {
	return transport.IsInvalidArgument(err)
}

// IsDeadlineExceeded returns true if ErrorCode(err) == CodeDeadlineExceeded.
func IsDeadlineExceeded(err error) bool {
	return transport.IsDeadlineExceeded(err)
}

// IsNotFound returns true if ErrorCode(err) == CodeNotFound.
func IsNotFound(err error) bool {
	return transport.IsNotFound(err)
}

// IsAlreadyExists returns true if ErrorCode(err) == CodeAlreadyExists.
func IsAlreadyExists(err error) bool {
	return transport.IsAlreadyExists(err)
}

// IsPermissionDenied returns true if ErrorCode(err) == CodePermissionDenied.
func IsPermissionDenied(err error) bool {
	return transport.IsPermissionDenied(err)
}

// IsResourceExhausted returns true if ErrorCode(err) == CodeResourceExhausted.
func IsResourceExhausted(err error) bool {
	return transport.IsResourceExhausted(err)
}

// IsFailedPrecondition returns true if ErrorCode(err) == CodeFailedPrecondition.
func IsFailedPrecondition(err error) bool {
	return transport.IsFailedPrecondition(err)
}

// IsAborted returns true if ErrorCode(err) == CodeAborted.
func IsAborted(err error) bool {
	return transport.IsAborted(err)
}

// IsOutOfRange returns true if ErrorCode(err) == CodeOutOfRange.
func IsOutOfRange(err error) bool {
	return transport.IsOutOfRange(err)
}

// IsUnimplemented returns true if ErrorCode(err) == CodeUnimplemented.
func IsUnimplemented(err error) bool {
	return transport.IsUnimplemented(err)
}

// IsInternal returns true if ErrorCode(err) == CodeInternal.
func IsInternal(err error) bool {
	return transport.IsInternal(err)
}

// IsUnavailable returns true if ErrorCode(err) == CodeUnavailable.
func IsUnavailable(err error) bool {
	return transport.IsUnavailable(err)
}

// IsDataLoss returns true if ErrorCode(err) == CodeDataLoss.
func IsDataLoss(err error) bool {
	return transport.IsDataLoss(err)
}

// IsUnauthenticated returns true if ErrorCode(err) == CodeUnauthenticated.
func IsUnauthenticated(err error) bool {
	return transport.IsUnauthenticated(err)
}
