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

package yarpcerror

import (
	"bytes"
	"fmt"
)

// Info holds metadata about an error.
type Info struct {
	Code    Code
	Message string
	Name    string
}

// New returns a new error.
//
// The Code should never be CodeOK; if it is, this will return nil.
func New(code Code, message string, options ...ErrorOption) error {
	if code == CodeOK {
		return nil
	}
	encodingError := &encodingError{
		info: Info{
			Code:    code,
			Message: message,
		},
	}
	for _, opt := range options {
		opt.apply(encodingError)
	}

	if err := validateName(encodingError.info.Name); err != nil {
		return err
	}
	return encodingError
}

// // ExtractInfo returns t Status for the provided error. If the provided error
// // is not a Status, a new error with code CodeUnknown is returned.
// //
// // Returns nil if the provided error is nil.
// func FromError(err error) encodingError {
// 	if err == nil {
// 		return nil
// 	}
// 	if status, ok := err.(*encodingError); ok {
// 		return status
// 	}
// 	return &encodingError{
// 		code:    CodeUnknown,
// 		message: err.Error(),
// 	}
// }

// WrapError returns an error wrapped as an encodingError.
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	if encodingError, ok := err.(*encodingError); ok {
		return encodingError
	}
	return &encodingError{
		info: Info{
			Code:    CodeUnknown,
			Message: err.Error(),
		},
	}
}

// ExtractInfo casts the error into an encodingError and returns its info field.
// If it fails to cast, it returns an empty Info struct.
func ExtractInfo(err error) Info {
	if err == nil {
		return Info{}
	}
	if encodingError, ok := err.(*encodingError); ok {
		return encodingError.info
	}
	return Info{}
}

// ExtractDetails casts the error into an encodingError and returns its details
// field. If it fails to cast, it returns nil.
func ExtractDetails(err error) interface{} {
	if err == nil {
		return nil
	}
	if encodingError, ok := err.(*encodingError); ok {
		return encodingError.details
	}
	return nil
}

// IsStatus returns whether the provided error is a YARPC error.
//
// This is always false if the error is nil.
func IsStatus(err error) bool {
	_, ok := err.(*encodingError)
	return ok
}

// encodingError represents a YARPC error.
type encodingError struct {
	info    Info
	details interface{}
}

// ErrorOption provides options that may be called to the constructor.
type ErrorOption struct{ apply func(*encodingError) }

// WithName returns a new status with the given name.
//
// This should be used for user-defined errors.
func WithName(name string) ErrorOption {
	return ErrorOption{func(s *encodingError) { s.info.Name = name }}
}

// WithDetails adds semantic metadata to a Status.
func WithDetails(details interface{}) ErrorOption {
	return ErrorOption{func(s *encodingError) { s.details = details }}
}

// Error implements the error interface.
func (s *encodingError) Error() string {
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`code:`)
	_, _ = buffer.WriteString(s.info.Code.String())
	if s.info.Name != "" {
		_, _ = buffer.WriteString(` name:`)
		_, _ = buffer.WriteString(s.info.Name)
	}
	if s.info.Message != "" {
		_, _ = buffer.WriteString(` message:`)
		_, _ = buffer.WriteString(s.info.Message)
	}
	return buffer.String()
}

// CancelledErrorf returns a new Status with code CodeCancelled
// by calling New(CodeCancelled, sprintf(format, args...)).
func CancelledErrorf(format string, args ...interface{}) error {
	return New(CodeCancelled, sprintf(format, args...))
}

// UnknownErrorf returns a new Status with code CodeUnknown
// by calling New(CodeUnknown, sprintf(format, args...)).
func UnknownErrorf(format string, args ...interface{}) error {
	return New(CodeUnknown, sprintf(format, args...))
}

// InvalidArgumentErrorf returns a new Status with code CodeInvalidArgument
// by calling New(CodeInvalidArgument, sprintf(format, args...)).
func InvalidArgumentErrorf(format string, args ...interface{}) error {
	return New(CodeInvalidArgument, sprintf(format, args...))
}

// DeadlineExceededErrorf returns a new Status with code CodeDeadlineExceeded
// by calling New(CodeDeadlineExceeded, sprintf(format, args...)).
func DeadlineExceededErrorf(format string, args ...interface{}) error {
	return New(CodeDeadlineExceeded, sprintf(format, args...))
}

// NotFoundErrorf returns a new Status with code CodeNotFound
// by calling New(CodeNotFound, sprintf(format, args...)).
func NotFoundErrorf(format string, args ...interface{}) error {
	return New(CodeNotFound, sprintf(format, args...))
}

// AlreadyExistsErrorf returns a new Status with code CodeAlreadyExists
// by calling New(CodeAlreadyExists, sprintf(format, args...)).
func AlreadyExistsErrorf(format string, args ...interface{}) error {
	return New(CodeAlreadyExists, sprintf(format, args...))
}

// PermissionDeniedErrorf returns a new Status with code CodePermissionDenied
// by calling New(CodePermissionDenied, sprintf(format, args...)).
func PermissionDeniedErrorf(format string, args ...interface{}) error {
	return New(CodePermissionDenied, sprintf(format, args...))
}

// ResourceExhaustedErrorf returns a new Status with code CodeResourceExhausted
// by calling New(CodeResourceExhausted, sprintf(format, args...)).
func ResourceExhaustedErrorf(format string, args ...interface{}) error {
	return New(CodeResourceExhausted, sprintf(format, args...))
}

// FailedPreconditionErrorf returns a new Status with code CodeFailedPrecondition
// by calling New(CodeFailedPrecondition, sprintf(format, args...)).
func FailedPreconditionErrorf(format string, args ...interface{}) error {
	return New(CodeFailedPrecondition, sprintf(format, args...))
}

// AbortedErrorf returns a new Status with code CodeAborted
// by calling New(CodeAborted, sprintf(format, args...)).
func AbortedErrorf(format string, args ...interface{}) error {
	return New(CodeAborted, sprintf(format, args...))
}

// OutOfRangeErrorf returns a new Status with code CodeOutOfRange
// by calling New(CodeOutOfRange, sprintf(format, args...)).
func OutOfRangeErrorf(format string, args ...interface{}) error {
	return New(CodeOutOfRange, sprintf(format, args...))
}

// UnimplementedErrorf returns a new Status with code CodeUnimplemented
// by calling New(CodeUnimplemented, sprintf(format, args...)).
func UnimplementedErrorf(format string, args ...interface{}) error {
	return New(CodeUnimplemented, sprintf(format, args...))
}

// InternalErrorf returns a new Status with code CodeInternal
// by calling New(CodeInternal, sprintf(format, args...)).
func InternalErrorf(format string, args ...interface{}) error {
	return New(CodeInternal, sprintf(format, args...))
}

// UnavailableErrorf returns a new Status with code CodeUnavailable
// by calling New(CodeUnavailable, sprintf(format, args...)).
func UnavailableErrorf(format string, args ...interface{}) error {
	return New(CodeUnavailable, sprintf(format, args...))
}

// DataLossErrorf returns a new Status with code CodeDataLoss
// by calling New(CodeDataLoss, sprintf(format, args...)).
func DataLossErrorf(format string, args ...interface{}) error {
	return New(CodeDataLoss, sprintf(format, args...))
}

// UnauthenticatedErrorf returns a new Status with code CodeUnauthenticated
// by calling New(CodeUnauthenticated, sprintf(format, args...)).
func UnauthenticatedErrorf(format string, args ...interface{}) error {
	return New(CodeUnauthenticated, sprintf(format, args...))
}

// IsCancelled returns true if ExtractInfo(err).Code == CodeCancelled.
func IsCancelled(err error) bool {
	return ExtractInfo(err).Code == CodeCancelled
}

// IsUnknown returns true if ExtractInfo(err).Code == CodeUnknown.
func IsUnknown(err error) bool {
	return ExtractInfo(err).Code == CodeUnknown
}

// IsInvalidArgument returns true if ExtractInfo(err).Code == CodeInvalidArgument.
func IsInvalidArgument(err error) bool {
	return ExtractInfo(err).Code == CodeInvalidArgument
}

// IsDeadlineExceeded returns true if ExtractInfo(err).Code == CodeDeadlineExceeded.
func IsDeadlineExceeded(err error) bool {
	return ExtractInfo(err).Code == CodeDeadlineExceeded
}

// IsNotFound returns true if ExtractInfo(err).Code == CodeNotFound.
func IsNotFound(err error) bool {
	return ExtractInfo(err).Code == CodeNotFound
}

// IsAlreadyExists returns true if ExtractInfo(err).Code == CodeAlreadyExists.
func IsAlreadyExists(err error) bool {
	return ExtractInfo(err).Code == CodeAlreadyExists
}

// IsPermissionDenied returns true if ExtractInfo(err).Code == CodePermissionDenied.
func IsPermissionDenied(err error) bool {
	return ExtractInfo(err).Code == CodePermissionDenied
}

// IsResourceExhausted returns true if ExtractInfo(err).Code == CodeResourceExhausted.
func IsResourceExhausted(err error) bool {
	return ExtractInfo(err).Code == CodeResourceExhausted
}

// IsFailedPrecondition returns true if ExtractInfo(err).Code == CodeFailedPrecondition.
func IsFailedPrecondition(err error) bool {
	return ExtractInfo(err).Code == CodeFailedPrecondition
}

// IsAborted returns true if ExtractInfo(err).Code == CodeAborted.
func IsAborted(err error) bool {
	return ExtractInfo(err).Code == CodeAborted
}

// IsOutOfRange returns true if ExtractInfo(err).Code == CodeOutOfRange.
func IsOutOfRange(err error) bool {
	return ExtractInfo(err).Code == CodeOutOfRange
}

// IsUnimplemented returns true if ExtractInfo(err).Code == CodeUnimplemented.
func IsUnimplemented(err error) bool {
	return ExtractInfo(err).Code == CodeUnimplemented
}

// IsInternal returns true if ExtractInfo(err).Code == CodeInternal.
func IsInternal(err error) bool {
	return ExtractInfo(err).Code == CodeInternal
}

// IsUnavailable returns true if ExtractInfo(err).Code == CodeUnavailable.
func IsUnavailable(err error) bool {
	return ExtractInfo(err).Code == CodeUnavailable
}

// IsDataLoss returns true if ExtractInfo(err).Code == CodeDataLoss.
func IsDataLoss(err error) bool {
	return ExtractInfo(err).Code == CodeDataLoss
}

// IsUnauthenticated returns true if ExtractInfo(err).Code == CodeUnauthenticated.
func IsUnauthenticated(err error) bool {
	return ExtractInfo(err).Code == CodeUnauthenticated
}

func sprintf(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
