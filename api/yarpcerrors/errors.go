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

// IsYARPCError returns true if the given error is a non-nil YARPC error.
func IsYARPCError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*yarpcError)
	return ok
}

// ErrorCode returns the Code for the given error, or CodeOK if the given
// error is not a YARPC error.
//
// While a YARPC error will never have CodeOK as an error code, this should not be
// used to test if an error is a YARPC error, use IsYARPCError instead.
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

// ErrorName returns the name for the given error, or "" if the given
// error is not a YARPC error created with NamedErrorf that has a non-empty name.
func ErrorName(err error) string {
	if err == nil {
		return ""
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return ""
	}
	return yarpcError.Name
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

// NamedErrorf returns a new yarpc error with code CodeUnknown and the given name.
// This should be used for user-defined errors.
func NamedErrorf(name string, format string, args ...interface{}) error {
	return newYARPCError(CodeUnknown, name, format, args...)
}

// CancelledErrorf returns a new yarpc error with code CodeCancelled.
func CancelledErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeCancelled, "", format, args...)
}

// UnknownErrorf returns a new yarpc error with code CodeUnknown.
func UnknownErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeUnknown, "", format, args...)
}

// InvalidArgumentErrorf returns a new yarpc error with code CodeInvalidArgument.
func InvalidArgumentErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeInvalidArgument, "", format, args...)
}

// DeadlineExceededErrorf returns a new yarpc error with code CodeDeadlineExceeded.
func DeadlineExceededErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeDeadlineExceeded, "", format, args...)
}

// NotFoundErrorf returns a new yarpc error with code CodeNotFound.
func NotFoundErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeNotFound, "", format, args...)
}

// AlreadyExistsErrorf returns a new yarpc error with code CodeAlreadyExists.
func AlreadyExistsErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeAlreadyExists, "", format, args...)
}

// PermissionDeniedErrorf returns a new yarpc error with code CodePermissionDenied.
func PermissionDeniedErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodePermissionDenied, "", format, args...)
}

// ResourceExhaustedErrorf returns a new yarpc error with code CodeResourceExhausted.
func ResourceExhaustedErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeResourceExhausted, "", format, args...)
}

// FailedPreconditionErrorf returns a new yarpc error with code CodeFailedPrecondition.
func FailedPreconditionErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeFailedPrecondition, "", format, args...)
}

// AbortedErrorf returns a new yarpc error with code CodeAborted.
func AbortedErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeAborted, "", format, args...)
}

// OutOfRangeErrorf returns a new yarpc error with code CodeOutOfRange.
func OutOfRangeErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeOutOfRange, "", format, args...)
}

// UnimplementedErrorf returns a new yarpc error with code CodeUnimplemented.
func UnimplementedErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeUnimplemented, "", format, args...)
}

// InternalErrorf returns a new yarpc error with code CodeInternal.
func InternalErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeInternal, "", format, args...)
}

// UnavailableErrorf returns a new yarpc error with code CodeUnavailable.
func UnavailableErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeUnavailable, "", format, args...)
}

// DataLossErrorf returns a new yarpc error with code CodeDataLoss.
func DataLossErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeDataLoss, "", format, args...)
}

// UnauthenticatedErrorf returns a new yarpc error with code CodeUnauthenticated.
func UnauthenticatedErrorf(format string, args ...interface{}) error {
	return newYARPCError(CodeUnauthenticated, "", format, args...)
}

type yarpcError struct {
	// Code is the code of the error. This should never be set to CodeOK.
	Code Code `json:"code,omitempty"`
	// Name is the name of the error. This should only be set if the
	// Code is CodeUnknown.
	Name string `json:"name,omitempty"`
	// Message is the message of the error.
	Message string `json:"message,omitempty"`
}

func newYARPCError(code Code, name string, format string, args ...interface{}) *yarpcError {
	return &yarpcError{code, name, fmt.Sprintf(format, args...)}
}

func (e *yarpcError) Error() string {
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`code:`)
	_, _ = buffer.WriteString(e.Code.String())
	if e.Name != "" {
		_, _ = buffer.WriteString(` name:`)
		_, _ = buffer.WriteString(e.Name)
	}
	if e.Message != "" {
		_, _ = buffer.WriteString(` message:`)
		_, _ = buffer.WriteString(e.Message)
	}
	return buffer.String()
}
