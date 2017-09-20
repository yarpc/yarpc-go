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

// Newf returns a new Status.
//
// The Code should never be CodeOK, if it is, this will return nil.
func Newf(code Code, format string, args ...interface{}) *Status {
	if code == CodeOK {
		return nil
	}
	return &Status{
		code:    code,
		message: sprintf(format, args...),
	}
}

// FromError returns the Status for the error.
//
// If the error is nil, ut returns nil
// If the error is a Status, it returns err.
// If the error is not a Status, it returns a new error with the Code
// CodeUnknown and the message err.Error()
func FromError(err error) *Status {
	if err == nil {
		return nil
	}
	if status, ok := err.(*Status); ok {
		return status
	}
	return &Status{
		code:    CodeUnknown,
		message: err.Error(),
	}
}

// IsStatus returns true if err is a Status.
func IsStatus(err error) bool {
	_, ok := err.(*Status)
	return ok
}

// Status represents a YARPC error.
type Status struct {
	code    Code
	name    string
	message string
}

// WithName returns a new Status with the given name.
//
// If s is nil, this will still return nil.
//
// This should be used for user-defined errors.
//
// The name must only contain lowercase letters from a-z and dashes (-), and
// cannot start or end in a dash. If the name is something else, an error with
// code CodeInternal will be returned.
func (s *Status) WithName(name string) *Status {
	if s == nil {
		return nil
	}
	if err := validateName(name); err != nil {
		return err.(*Status)
	}
	return &Status{
		code:    s.code,
		name:    name,
		message: s.message,
	}
}

// Code returns the code for the Status.
func (s *Status) Code() Code {
	if s == nil {
		return CodeOK
	}
	return s.code
}

// Name returns the name for the Status.
func (s *Status) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

// Message returns the message for the Status.
func (s *Status) Message() string {
	if s == nil {
		return ""
	}
	return s.message
}

// Error implements the error interface.
func (s *Status) Error() string {
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`code:`)
	_, _ = buffer.WriteString(s.code.String())
	if s.name != "" {
		_, _ = buffer.WriteString(` name:`)
		_, _ = buffer.WriteString(s.name)
	}
	if s.message != "" {
		_, _ = buffer.WriteString(` message:`)
		_, _ = buffer.WriteString(s.message)
	}
	return buffer.String()
}

// ErrorCode returns the Code for the given error, CodeOK if the error is nil,
// or CodeUnknown if the given error is not a YARPC error.
//
// Deprecated: Use FromError and Code instead.
func ErrorCode(err error) Code {
	return FromError(err).Code()
}

// ErrorName returns the name for the given error, or "" if the given
// error is not a YARPC error created with NamedErrorf that has a non-empty name.
//
// Deprecated: Use FromError and Name instead.
func ErrorName(err error) string {
	return FromError(err).Name()
}

// ErrorMessage returns the message for the given error, or "" if the given
// error is nil, or err.Error() if the given error is not a YARPC error or
// the YARPC error had no message.
//
// Deprecated: Use FromError and Message instead.
func ErrorMessage(err error) string {
	return FromError(err).Message()
}

// CancelledErrorf returns a new Status with code CodeCancelled.
//
// Deprecated: Use Newf instead.
func CancelledErrorf(format string, args ...interface{}) error {
	return Newf(CodeCancelled, format, args...)
}

// UnknownErrorf returns a new Status with code CodeUnknown.
//
// Deprecated: Use Newf instead.
func UnknownErrorf(format string, args ...interface{}) error {
	return Newf(CodeUnknown, format, args...)
}

// InvalidArgumentErrorf returns a new Status with code CodeInvalidArgument.
//
// Deprecated: Use Newf instead.
func InvalidArgumentErrorf(format string, args ...interface{}) error {
	return Newf(CodeInvalidArgument, format, args...)
}

// DeadlineExceededErrorf returns a new Status with code CodeDeadlineExceeded.
//
// Deprecated: Use Newf instead.
func DeadlineExceededErrorf(format string, args ...interface{}) error {
	return Newf(CodeDeadlineExceeded, format, args...)
}

// NotFoundErrorf returns a new Status with code CodeNotFound.
//
// Deprecated: Use Newf instead.
func NotFoundErrorf(format string, args ...interface{}) error {
	return Newf(CodeNotFound, format, args...)
}

// AlreadyExistsErrorf returns a new Status with code CodeAlreadyExists.
//
// Deprecated: Use Newf instead.
func AlreadyExistsErrorf(format string, args ...interface{}) error {
	return Newf(CodeAlreadyExists, format, args...)
}

// PermissionDeniedErrorf returns a new Status with code CodePermissionDenied.
//
// Deprecated: Use Newf instead.
func PermissionDeniedErrorf(format string, args ...interface{}) error {
	return Newf(CodePermissionDenied, format, args...)
}

// ResourceExhaustedErrorf returns a new Status with code CodeResourceExhausted.
//
// Deprecated: Use Newf instead.
func ResourceExhaustedErrorf(format string, args ...interface{}) error {
	return Newf(CodeResourceExhausted, format, args...)
}

// FailedPreconditionErrorf returns a new Status with code CodeFailedPrecondition.
//
// Deprecated: Use Newf instead.
func FailedPreconditionErrorf(format string, args ...interface{}) error {
	return Newf(CodeFailedPrecondition, format, args...)
}

// AbortedErrorf returns a new Status with code CodeAborted.
//
// Deprecated: Use Newf instead.
func AbortedErrorf(format string, args ...interface{}) error {
	return Newf(CodeAborted, format, args...)
}

// OutOfRangeErrorf returns a new Status with code CodeOutOfRange.
//
// Deprecated: Use Newf instead.
func OutOfRangeErrorf(format string, args ...interface{}) error {
	return Newf(CodeOutOfRange, format, args...)
}

// UnimplementedErrorf returns a new Status with code CodeUnimplemented.
//
// Deprecated: Use Newf instead.
func UnimplementedErrorf(format string, args ...interface{}) error {
	return Newf(CodeUnimplemented, format, args...)
}

// InternalErrorf returns a new Status with code CodeInternal.
//
// Deprecated: Use Newf instead.
func InternalErrorf(format string, args ...interface{}) error {
	return Newf(CodeInternal, format, args...)
}

// UnavailableErrorf returns a new Status with code CodeUnavailable.
//
// Deprecated: Use Newf instead.
func UnavailableErrorf(format string, args ...interface{}) error {
	return Newf(CodeUnavailable, format, args...)
}

// DataLossErrorf returns a new Status with code CodeDataLoss.
//
// Deprecated: Use Newf instead.
func DataLossErrorf(format string, args ...interface{}) error {
	return Newf(CodeDataLoss, format, args...)
}

// UnauthenticatedErrorf returns a new Status with code CodeUnauthenticated.
//
// Deprecated: Use Newf instead.
func UnauthenticatedErrorf(format string, args ...interface{}) error {
	return Newf(CodeUnauthenticated, format, args...)
}

// IsCancelled returns true if ErrorCode(err) == CodeCancelled.
//
// Deprecated: Use FromError and Code instead.
func IsCancelled(err error) bool {
	return ErrorCode(err) == CodeCancelled
}

// IsUnknown returns true if ErrorCode(err) == CodeUnknown.
//
// Deprecated: Use FromError and Code instead.
func IsUnknown(err error) bool {
	return ErrorCode(err) == CodeUnknown
}

// IsInvalidArgument returns true if ErrorCode(err) == CodeInvalidArgument.
//
// Deprecated: Use FromError and Code instead.
func IsInvalidArgument(err error) bool {
	return ErrorCode(err) == CodeInvalidArgument
}

// IsDeadlineExceeded returns true if ErrorCode(err) == CodeDeadlineExceeded.
//
// Deprecated: Use FromError and Code instead.
func IsDeadlineExceeded(err error) bool {
	return ErrorCode(err) == CodeDeadlineExceeded
}

// IsNotFound returns true if ErrorCode(err) == CodeNotFound.
//
// Deprecated: Use FromError and Code instead.
func IsNotFound(err error) bool {
	return ErrorCode(err) == CodeNotFound
}

// IsAlreadyExists returns true if ErrorCode(err) == CodeAlreadyExists.
//
// Deprecated: Use FromError and Code instead.
func IsAlreadyExists(err error) bool {
	return ErrorCode(err) == CodeAlreadyExists
}

// IsPermissionDenied returns true if ErrorCode(err) == CodePermissionDenied.
//
// Deprecated: Use FromError and Code instead.
func IsPermissionDenied(err error) bool {
	return ErrorCode(err) == CodePermissionDenied
}

// IsResourceExhausted returns true if ErrorCode(err) == CodeResourceExhausted.
//
// Deprecated: Use FromError and Code instead.
func IsResourceExhausted(err error) bool {
	return ErrorCode(err) == CodeResourceExhausted
}

// IsFailedPrecondition returns true if ErrorCode(err) == CodeFailedPrecondition.
//
// Deprecated: Use FromError and Code instead.
func IsFailedPrecondition(err error) bool {
	return ErrorCode(err) == CodeFailedPrecondition
}

// IsAborted returns true if ErrorCode(err) == CodeAborted.
//
// Deprecated: Use FromError and Code instead.
func IsAborted(err error) bool {
	return ErrorCode(err) == CodeAborted
}

// IsOutOfRange returns true if ErrorCode(err) == CodeOutOfRange.
//
// Deprecated: Use FromError and Code instead.
func IsOutOfRange(err error) bool {
	return ErrorCode(err) == CodeOutOfRange
}

// IsUnimplemented returns true if ErrorCode(err) == CodeUnimplemented.
//
// Deprecated: Use FromError and Code instead.
func IsUnimplemented(err error) bool {
	return ErrorCode(err) == CodeUnimplemented
}

// IsInternal returns true if ErrorCode(err) == CodeInternal.
//
// Deprecated: Use FromError and Code instead.
func IsInternal(err error) bool {
	return ErrorCode(err) == CodeInternal
}

// IsUnavailable returns true if ErrorCode(err) == CodeUnavailable.
//
// Deprecated: Use FromError and Code instead.
func IsUnavailable(err error) bool {
	return ErrorCode(err) == CodeUnavailable
}

// IsDataLoss returns true if ErrorCode(err) == CodeDataLoss.
//
// Deprecated: Use FromError and Code instead.
func IsDataLoss(err error) bool {
	return ErrorCode(err) == CodeDataLoss
}

// IsUnauthenticated returns true if ErrorCode(err) == CodeUnauthenticated.
//
// Deprecated: Use FromError and Code instead.
func IsUnauthenticated(err error) bool {
	return ErrorCode(err) == CodeUnauthenticated
}

// IsYARPCError is a convenience function that returns true if the given error
// is a non-nil YARPC error.
//
// Deprecated: use IsStatus instead.
func IsYARPCError(err error) bool {
	return IsStatus(err)
}

// NamedErrorf returns a new Status with code CodeUnknown and the given name.
//
// This should be used for user-defined errors.
//
// The name must only contain lowercase letters from a-z and dashes (-), and
// cannot start or end in a dash. If the name is something else, an error with
// code CodeInternal will be returned.
//
// Deprecated: Use Newf and WithName instead.
func NamedErrorf(name string, format string, args ...interface{}) error {
	return Newf(CodeUnknown, format, args...).WithName(name)
}

// FromHeaders returns a new Status from headers transmitted from the server side.
//
// If the specified code is CodeOK, this will return nil.
//
// The name must only contain lowercase letters from a-z and dashes (-), and
// cannot start or end in a dash. If the name is something else, an error with
// code CodeInternal will be returned.
//
// This function should not be used by server implementations, use the individual
// error constructors instead. This should only be used by transport implementations.
//
// Deprecated: Use Newf and WithName instead.
func FromHeaders(code Code, name string, message string) error {
	return Newf(code, message).WithName(name)
}

func sprintf(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
