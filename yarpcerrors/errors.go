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

package yarpcerrors

import (
	"bytes"
	"fmt"
)

// IsYARPCError is a convenience function that returns true if the given error
// is a non-nil YARPC error.
//
// This is equivalent to yarpcerrors.ErrorCode(err) != yarpcerrors.CodeOK.
func IsYARPCError(err error) bool {
	return ErrorCode(err) != CodeOK
}

// ErrorCode returns the Code for the given error, or CodeOK if the given
// error is not a YARPC error.
func ErrorCode(err error) Code {
	if err == nil {
		return CodeOK
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return CodeOK
	}
	return yarpcError.Code
}

// ErrorMessage returns the message for the given error, or "" if the given
// error is not a YARPC error or the YARPC error had no message.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return ""
	}
	return yarpcError.Message
}

// CancelledErrorf returns a new yarpc error with code CodeCancelled.
func CancelledErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeCancelled, sprintf(format, args...))
}

// UnknownErrorf returns a new yarpc error with code CodeUnknown.
func UnknownErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeUnknown, sprintf(format, args...))
}

// InvalidArgumentErrorf returns a new yarpc error with code CodeInvalidArgument.
func InvalidArgumentErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeInvalidArgument, sprintf(format, args...))
}

// DeadlineExceededErrorf returns a new yarpc error with code CodeDeadlineExceeded.
func DeadlineExceededErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeDeadlineExceeded, sprintf(format, args...))
}

// NotFoundErrorf returns a new yarpc error with code CodeNotFound.
func NotFoundErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeNotFound, sprintf(format, args...))
}

// AlreadyExistsErrorf returns a new yarpc error with code CodeAlreadyExists.
func AlreadyExistsErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeAlreadyExists, sprintf(format, args...))
}

// PermissionDeniedErrorf returns a new yarpc error with code CodePermissionDenied.
func PermissionDeniedErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodePermissionDenied, sprintf(format, args...))
}

// ResourceExhaustedErrorf returns a new yarpc error with code CodeResourceExhausted.
func ResourceExhaustedErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeResourceExhausted, sprintf(format, args...))
}

// FailedPreconditionErrorf returns a new yarpc error with code CodeFailedPrecondition.
func FailedPreconditionErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeFailedPrecondition, sprintf(format, args...))
}

// AbortedErrorf returns a new yarpc error with code CodeAborted.
func AbortedErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeAborted, sprintf(format, args...))
}

// OutOfRangeErrorf returns a new yarpc error with code CodeOutOfRange.
func OutOfRangeErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeOutOfRange, sprintf(format, args...))
}

// UnimplementedErrorf returns a new yarpc error with code CodeUnimplemented.
func UnimplementedErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeUnimplemented, sprintf(format, args...))
}

// InternalErrorf returns a new yarpc error with code CodeInternal.
func InternalErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeInternal, sprintf(format, args...))
}

// UnavailableErrorf returns a new yarpc error with code CodeUnavailable.
func UnavailableErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeUnavailable, sprintf(format, args...))
}

// DataLossErrorf returns a new yarpc error with code CodeDataLoss.
func DataLossErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeDataLoss, sprintf(format, args...))
}

// UnauthenticatedErrorf returns a new yarpc error with code CodeUnauthenticated.
func UnauthenticatedErrorf(format string, args ...interface{}) error {
	return FromHeaders(CodeUnauthenticated, sprintf(format, args...))
}

// IsCancelled returns true if ErrorCode(err) == CodeCancelled.
func IsCancelled(err error) bool {
	return ErrorCode(err) == CodeCancelled
}

// IsUnknown returns true if ErrorCode(err) == CodeUnknown.
func IsUnknown(err error) bool {
	return ErrorCode(err) == CodeUnknown
}

// IsInvalidArgument returns true if ErrorCode(err) == CodeInvalidArgument.
func IsInvalidArgument(err error) bool {
	return ErrorCode(err) == CodeInvalidArgument
}

// IsDeadlineExceeded returns true if ErrorCode(err) == CodeDeadlineExceeded.
func IsDeadlineExceeded(err error) bool {
	return ErrorCode(err) == CodeDeadlineExceeded
}

// IsNotFound returns true if ErrorCode(err) == CodeNotFound.
func IsNotFound(err error) bool {
	return ErrorCode(err) == CodeNotFound
}

// IsAlreadyExists returns true if ErrorCode(err) == CodeAlreadyExists.
func IsAlreadyExists(err error) bool {
	return ErrorCode(err) == CodeAlreadyExists
}

// IsPermissionDenied returns true if ErrorCode(err) == CodePermissionDenied.
func IsPermissionDenied(err error) bool {
	return ErrorCode(err) == CodePermissionDenied
}

// IsResourceExhausted returns true if ErrorCode(err) == CodeResourceExhausted.
func IsResourceExhausted(err error) bool {
	return ErrorCode(err) == CodeResourceExhausted
}

// IsFailedPrecondition returns true if ErrorCode(err) == CodeFailedPrecondition.
func IsFailedPrecondition(err error) bool {
	return ErrorCode(err) == CodeFailedPrecondition
}

// IsAborted returns true if ErrorCode(err) == CodeAborted.
func IsAborted(err error) bool {
	return ErrorCode(err) == CodeAborted
}

// IsOutOfRange returns true if ErrorCode(err) == CodeOutOfRange.
func IsOutOfRange(err error) bool {
	return ErrorCode(err) == CodeOutOfRange
}

// IsUnimplemented returns true if ErrorCode(err) == CodeUnimplemented.
func IsUnimplemented(err error) bool {
	return ErrorCode(err) == CodeUnimplemented
}

// IsInternal returns true if ErrorCode(err) == CodeInternal.
func IsInternal(err error) bool {
	return ErrorCode(err) == CodeInternal
}

// IsUnavailable returns true if ErrorCode(err) == CodeUnavailable.
func IsUnavailable(err error) bool {
	return ErrorCode(err) == CodeUnavailable
}

// IsDataLoss returns true if ErrorCode(err) == CodeDataLoss.
func IsDataLoss(err error) bool {
	return ErrorCode(err) == CodeDataLoss
}

// IsUnauthenticated returns true if ErrorCode(err) == CodeUnauthenticated.
func IsUnauthenticated(err error) bool {
	return ErrorCode(err) == CodeUnauthenticated
}

// FromHeaders returns a new yarpc error from headers transmitted from the server side.
//
// If the specified code is CodeOK, this will return nil.
//
// This function should not be used by server implementations, use the individual
// error constructors instead. This should only be used by transport implementations.
func FromHeaders(code Code, message string) error {
	if code == CodeOK {
		return nil
	}
	return &yarpcError{
		Code:    code,
		Message: message,
	}
}

// ** All constructors of yarpcErrors must always set a Code that is not CodeOK. **
//
// Currently, the only constructor of yarpcErrors is FromHeaders, which enforces this.

type yarpcError struct {
	// Code is the code of the error. This should never be set to CodeOK.
	Code Code `json:"code,omitempty"`
	// Message is the message of the error.
	Message string `json:"message,omitempty"`
}

func (e *yarpcError) Error() string {
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`code:`)
	_, _ = buffer.WriteString(e.Code.String())
	if e.Message != "" {
		_, _ = buffer.WriteString(` message:`)
		_, _ = buffer.WriteString(e.Message)
	}
	return buffer.String()
}

func sprintf(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
