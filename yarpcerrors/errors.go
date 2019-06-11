// Copyright (c) 2019 Uber Technologies, Inc.
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

// FromError returns the Status for the provided error. If the provided error
// is not a Status, a new error with code CodeUnknown is returned.
//
// Returns nil if the provided error is nil.
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

// IsStatus returns whether the provided error is a YARPC error.
//
// This is always false if the error is nil.
func IsStatus(err error) bool {
	_, ok := err.(*Status)
	return ok
}

// Status represents a YARPC error.
type Status struct {
	code    Code
	name    string
	message string
	details []byte
}

// WithName returns a new Status with the given name.
//
// This should be used for user-defined errors.
//
// Deprecated: Use only error codes to represent the type of the error.
func (s *Status) WithName(name string) *Status {
	// TODO: We plan to add a WithDetails method to add semantic metadata to
	// Statuses soon.
	if s == nil {
		return nil
	}

	return &Status{
		code:    s.code,
		name:    name,
		message: s.message,
		details: s.details,
	}
}

// WithDetails returns a new status with the given details bytes.
func (s *Status) WithDetails(details []byte) *Status {
	if s == nil {
		return nil
	}
	if len(details) == 0 {
		// this ensures that the details field is not set to some pointer if
		// there's nothing in details.
		details = nil
	}
	return &Status{
		code:    s.code,
		name:    s.name,
		message: s.message,
		details: details,
	}
}

// Code returns the error code for this Status.
func (s *Status) Code() Code {
	if s == nil {
		return CodeOK
	}
	return s.code
}

// Name returns the name of the error for this Status.
//
// This is an empty string for all built-in YARPC errors. It may be customized
// by using WithName.
func (s *Status) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

// Message returns the error message for this Status.
func (s *Status) Message() string {
	if s == nil {
		return ""
	}
	return s.message
}

// Details returns the error details for this Status.
func (s *Status) Details() []byte {
	if s == nil {
		return nil
	}
	return s.details
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

// CancelledErrorf returns a new Status with code CodeCancelled
// by calling Newf(CodeCancelled, format, args...).
func CancelledErrorf(format string, args ...interface{}) error {
	return Newf(CodeCancelled, format, args...)
}

// UnknownErrorf returns a new Status with code CodeUnknown
// by calling Newf(CodeUnknown, format, args...).
func UnknownErrorf(format string, args ...interface{}) error {
	return Newf(CodeUnknown, format, args...)
}

// InvalidArgumentErrorf returns a new Status with code CodeInvalidArgument
// by calling Newf(CodeInvalidArgument, format, args...).
func InvalidArgumentErrorf(format string, args ...interface{}) error {
	return Newf(CodeInvalidArgument, format, args...)
}

// DeadlineExceededErrorf returns a new Status with code CodeDeadlineExceeded
// by calling Newf(CodeDeadlineExceeded, format, args...).
func DeadlineExceededErrorf(format string, args ...interface{}) error {
	return Newf(CodeDeadlineExceeded, format, args...)
}

// NotFoundErrorf returns a new Status with code CodeNotFound
// by calling Newf(CodeNotFound, format, args...).
func NotFoundErrorf(format string, args ...interface{}) error {
	return Newf(CodeNotFound, format, args...)
}

// AlreadyExistsErrorf returns a new Status with code CodeAlreadyExists
// by calling Newf(CodeAlreadyExists, format, args...).
func AlreadyExistsErrorf(format string, args ...interface{}) error {
	return Newf(CodeAlreadyExists, format, args...)
}

// PermissionDeniedErrorf returns a new Status with code CodePermissionDenied
// by calling Newf(CodePermissionDenied, format, args...).
func PermissionDeniedErrorf(format string, args ...interface{}) error {
	return Newf(CodePermissionDenied, format, args...)
}

// ResourceExhaustedErrorf returns a new Status with code CodeResourceExhausted
// by calling Newf(CodeResourceExhausted, format, args...).
func ResourceExhaustedErrorf(format string, args ...interface{}) error {
	return Newf(CodeResourceExhausted, format, args...)
}

// FailedPreconditionErrorf returns a new Status with code CodeFailedPrecondition
// by calling Newf(CodeFailedPrecondition, format, args...).
func FailedPreconditionErrorf(format string, args ...interface{}) error {
	return Newf(CodeFailedPrecondition, format, args...)
}

// AbortedErrorf returns a new Status with code CodeAborted
// by calling Newf(CodeAborted, format, args...).
func AbortedErrorf(format string, args ...interface{}) error {
	return Newf(CodeAborted, format, args...)
}

// OutOfRangeErrorf returns a new Status with code CodeOutOfRange
// by calling Newf(CodeOutOfRange, format, args...).
func OutOfRangeErrorf(format string, args ...interface{}) error {
	return Newf(CodeOutOfRange, format, args...)
}

// UnimplementedErrorf returns a new Status with code CodeUnimplemented
// by calling Newf(CodeUnimplemented, format, args...).
func UnimplementedErrorf(format string, args ...interface{}) error {
	return Newf(CodeUnimplemented, format, args...)
}

// InternalErrorf returns a new Status with code CodeInternal
// by calling Newf(CodeInternal, format, args...).
func InternalErrorf(format string, args ...interface{}) error {
	return Newf(CodeInternal, format, args...)
}

// UnavailableErrorf returns a new Status with code CodeUnavailable
// by calling Newf(CodeUnavailable, format, args...).
func UnavailableErrorf(format string, args ...interface{}) error {
	return Newf(CodeUnavailable, format, args...)
}

// DataLossErrorf returns a new Status with code CodeDataLoss
// by calling Newf(CodeDataLoss, format, args...).
func DataLossErrorf(format string, args ...interface{}) error {
	return Newf(CodeDataLoss, format, args...)
}

// UnauthenticatedErrorf returns a new Status with code CodeUnauthenticated
// by calling Newf(CodeUnauthenticated, format, args...).
func UnauthenticatedErrorf(format string, args ...interface{}) error {
	return Newf(CodeUnauthenticated, format, args...)
}

// IsCancelled returns true if FromError(err).Code() == CodeCancelled.
func IsCancelled(err error) bool {
	return FromError(err).Code() == CodeCancelled
}

// IsUnknown returns true if FromError(err).Code() == CodeUnknown.
func IsUnknown(err error) bool {
	return FromError(err).Code() == CodeUnknown
}

// IsInvalidArgument returns true if FromError(err).Code() == CodeInvalidArgument.
func IsInvalidArgument(err error) bool {
	return FromError(err).Code() == CodeInvalidArgument
}

// IsDeadlineExceeded returns true if FromError(err).Code() == CodeDeadlineExceeded.
func IsDeadlineExceeded(err error) bool {
	return FromError(err).Code() == CodeDeadlineExceeded
}

// IsNotFound returns true if FromError(err).Code() == CodeNotFound.
func IsNotFound(err error) bool {
	return FromError(err).Code() == CodeNotFound
}

// IsAlreadyExists returns true if FromError(err).Code() == CodeAlreadyExists.
func IsAlreadyExists(err error) bool {
	return FromError(err).Code() == CodeAlreadyExists
}

// IsPermissionDenied returns true if FromError(err).Code() == CodePermissionDenied.
func IsPermissionDenied(err error) bool {
	return FromError(err).Code() == CodePermissionDenied
}

// IsResourceExhausted returns true if FromError(err).Code() == CodeResourceExhausted.
func IsResourceExhausted(err error) bool {
	return FromError(err).Code() == CodeResourceExhausted
}

// IsFailedPrecondition returns true if FromError(err).Code() == CodeFailedPrecondition.
func IsFailedPrecondition(err error) bool {
	return FromError(err).Code() == CodeFailedPrecondition
}

// IsAborted returns true if FromError(err).Code() == CodeAborted.
func IsAborted(err error) bool {
	return FromError(err).Code() == CodeAborted
}

// IsOutOfRange returns true if FromError(err).Code() == CodeOutOfRange.
func IsOutOfRange(err error) bool {
	return FromError(err).Code() == CodeOutOfRange
}

// IsUnimplemented returns true if FromError(err).Code() == CodeUnimplemented.
func IsUnimplemented(err error) bool {
	return FromError(err).Code() == CodeUnimplemented
}

// IsInternal returns true if FromError(err).Code() == CodeInternal.
func IsInternal(err error) bool {
	return FromError(err).Code() == CodeInternal
}

// IsUnavailable returns true if FromError(err).Code() == CodeUnavailable.
func IsUnavailable(err error) bool {
	return FromError(err).Code() == CodeUnavailable
}

// IsDataLoss returns true if FromError(err).Code() == CodeDataLoss.
func IsDataLoss(err error) bool {
	return FromError(err).Code() == CodeDataLoss
}

// IsUnauthenticated returns true if FromError(err).Code() == CodeUnauthenticated.
func IsUnauthenticated(err error) bool {
	return FromError(err).Code() == CodeUnauthenticated
}

// IsYARPCError returns whether the provided error is a YARPC error.
//
// This is always false if the error is nil.
//
// Deprecated: use IsStatus instead.
func IsYARPCError(err error) bool {
	return IsStatus(err)
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
