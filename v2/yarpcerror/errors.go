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
// If the Code is CodeOK, this will return nil.
func New(code Code, message string, options ...ErrorOption) error {
	if code == CodeOK {
		return nil
	}
	yarpcErr := &yarpcerror{
		info: Info{
			Code:    code,
			Message: message,
		},
	}
	for _, opt := range options {
		opt.apply(yarpcErr)
	}

	if err := validateName(yarpcErr.info.Name); err != nil {
		return err
	}
	return yarpcErr
}

// GetInfo returns the relevant error yarpcerror.Info.
//
// If the error is nil, it returns info with CodeOK. If the error is not a
// YARPC error, it returns info with CodeUnknown and its error message. If the
// error is a YARPC error, it returns its error information.
func GetInfo(err error) Info {
	if err == nil {
		return Info{Code: CodeOK}
	}
	if yarpcErr, ok := err.(*yarpcerror); ok {
		return yarpcErr.info
	}
	return Info{Code: CodeUnknown, Message: err.Error()}
}

// GetDetails returns the details field for the given error. This is only
// non-nil for a YARPC error.
func GetDetails(err error) interface{} {
	if err == nil {
		return nil
	}
	if yarpcErr, ok := err.(*yarpcerror); ok {
		return yarpcErr.details
	}
	return nil
}

type yarpcerror struct {
	info    Info
	details interface{}
}

// ErrorOption provides options that may be called to the constructor.
type ErrorOption struct{ apply func(*yarpcerror) }

// WithName adds the given name to the error.
//
// This should be used for user-defined errors.
func WithName(name string) ErrorOption {
	return ErrorOption{func(s *yarpcerror) { s.info.Name = name }}
}

// WithDetails adds semantic metadata to an error.
func WithDetails(details interface{}) ErrorOption {
	return ErrorOption{func(s *yarpcerror) { s.details = details }}
}

// Error implements the error interface.
func (s *yarpcerror) Error() string {
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

// CancelledErrorf returns a new error with code CodeCancelled
// by calling New(CodeCancelled, sprintf(format, args...)).
func CancelledErrorf(format string, args ...interface{}) error {
	return New(CodeCancelled, sprintf(format, args...))
}

// UnknownErrorf returns a new error with code CodeUnknown
// by calling New(CodeUnknown, sprintf(format, args...)).
func UnknownErrorf(format string, args ...interface{}) error {
	return New(CodeUnknown, sprintf(format, args...))
}

// InvalidArgumentErrorf returns a new error with code CodeInvalidArgument
// by calling New(CodeInvalidArgument, sprintf(format, args...)).
func InvalidArgumentErrorf(format string, args ...interface{}) error {
	return New(CodeInvalidArgument, sprintf(format, args...))
}

// DeadlineExceededErrorf returns a new error with code CodeDeadlineExceeded
// by calling New(CodeDeadlineExceeded, sprintf(format, args...)).
func DeadlineExceededErrorf(format string, args ...interface{}) error {
	return New(CodeDeadlineExceeded, sprintf(format, args...))
}

// NotFoundErrorf returns a new error with code CodeNotFound
// by calling New(CodeNotFound, sprintf(format, args...)).
func NotFoundErrorf(format string, args ...interface{}) error {
	return New(CodeNotFound, sprintf(format, args...))
}

// AlreadyExistsErrorf returns a new error with code CodeAlreadyExists
// by calling New(CodeAlreadyExists, sprintf(format, args...)).
func AlreadyExistsErrorf(format string, args ...interface{}) error {
	return New(CodeAlreadyExists, sprintf(format, args...))
}

// PermissionDeniedErrorf returns a new error with code CodePermissionDenied
// by calling New(CodePermissionDenied, sprintf(format, args...)).
func PermissionDeniedErrorf(format string, args ...interface{}) error {
	return New(CodePermissionDenied, sprintf(format, args...))
}

// ResourceExhaustedErrorf returns a new error with code CodeResourceExhausted
// by calling New(CodeResourceExhausted, sprintf(format, args...)).
func ResourceExhaustedErrorf(format string, args ...interface{}) error {
	return New(CodeResourceExhausted, sprintf(format, args...))
}

// FailedPreconditionErrorf returns a new error with code CodeFailedPrecondition
// by calling New(CodeFailedPrecondition, sprintf(format, args...)).
func FailedPreconditionErrorf(format string, args ...interface{}) error {
	return New(CodeFailedPrecondition, sprintf(format, args...))
}

// AbortedErrorf returns a new error with code CodeAborted
// by calling New(CodeAborted, sprintf(format, args...)).
func AbortedErrorf(format string, args ...interface{}) error {
	return New(CodeAborted, sprintf(format, args...))
}

// OutOfRangeErrorf returns a new error with code CodeOutOfRange
// by calling New(CodeOutOfRange, sprintf(format, args...)).
func OutOfRangeErrorf(format string, args ...interface{}) error {
	return New(CodeOutOfRange, sprintf(format, args...))
}

// UnimplementedErrorf returns a new error with code CodeUnimplemented
// by calling New(CodeUnimplemented, sprintf(format, args...)).
func UnimplementedErrorf(format string, args ...interface{}) error {
	return New(CodeUnimplemented, sprintf(format, args...))
}

// InternalErrorf returns a new error with code CodeInternal
// by calling New(CodeInternal, sprintf(format, args...)).
func InternalErrorf(format string, args ...interface{}) error {
	return New(CodeInternal, sprintf(format, args...))
}

// UnavailableErrorf returns a new error with code CodeUnavailable
// by calling New(CodeUnavailable, sprintf(format, args...)).
func UnavailableErrorf(format string, args ...interface{}) error {
	return New(CodeUnavailable, sprintf(format, args...))
}

// DataLossErrorf returns a new error with code CodeDataLoss
// by calling New(CodeDataLoss, sprintf(format, args...)).
func DataLossErrorf(format string, args ...interface{}) error {
	return New(CodeDataLoss, sprintf(format, args...))
}

// UnauthenticatedErrorf returns a new error with code CodeUnauthenticated
// by calling New(CodeUnauthenticated, sprintf(format, args...)).
func UnauthenticatedErrorf(format string, args ...interface{}) error {
	return New(CodeUnauthenticated, sprintf(format, args...))
}

// IsCancelled returns true if GetInfo(err).Code == CodeCancelled.
func IsCancelled(err error) bool {
	return GetInfo(err).Code == CodeCancelled
}

// IsUnknown returns true if GetInfo(err).Code == CodeUnknown.
func IsUnknown(err error) bool {
	return GetInfo(err).Code == CodeUnknown
}

// IsInvalidArgument returns true if GetInfo(err).Code == CodeInvalidArgument.
func IsInvalidArgument(err error) bool {
	return GetInfo(err).Code == CodeInvalidArgument
}

// IsDeadlineExceeded returns true if GetInfo(err).Code == CodeDeadlineExceeded.
func IsDeadlineExceeded(err error) bool {
	return GetInfo(err).Code == CodeDeadlineExceeded
}

// IsNotFound returns true if GetInfo(err).Code == CodeNotFound.
func IsNotFound(err error) bool {
	return GetInfo(err).Code == CodeNotFound
}

// IsAlreadyExists returns true if GetInfo(err).Code == CodeAlreadyExists.
func IsAlreadyExists(err error) bool {
	return GetInfo(err).Code == CodeAlreadyExists
}

// IsPermissionDenied returns true if GetInfo(err).Code == CodePermissionDenied.
func IsPermissionDenied(err error) bool {
	return GetInfo(err).Code == CodePermissionDenied
}

// IsResourceExhausted returns true if GetInfo(err).Code == CodeResourceExhausted.
func IsResourceExhausted(err error) bool {
	return GetInfo(err).Code == CodeResourceExhausted
}

// IsFailedPrecondition returns true if GetInfo(err).Code == CodeFailedPrecondition.
func IsFailedPrecondition(err error) bool {
	return GetInfo(err).Code == CodeFailedPrecondition
}

// IsAborted returns true if GetInfo(err).Code == CodeAborted.
func IsAborted(err error) bool {
	return GetInfo(err).Code == CodeAborted
}

// IsOutOfRange returns true if GetInfo(err).Code == CodeOutOfRange.
func IsOutOfRange(err error) bool {
	return GetInfo(err).Code == CodeOutOfRange
}

// IsUnimplemented returns true if GetInfo(err).Code == CodeUnimplemented.
func IsUnimplemented(err error) bool {
	return GetInfo(err).Code == CodeUnimplemented
}

// IsInternal returns true if GetInfo(err).Code == CodeInternal.
func IsInternal(err error) bool {
	return GetInfo(err).Code == CodeInternal
}

// IsUnavailable returns true if GetInfo(err).Code == CodeUnavailable.
func IsUnavailable(err error) bool {
	return GetInfo(err).Code == CodeUnavailable
}

// IsDataLoss returns true if GetInfo(err).Code == CodeDataLoss.
func IsDataLoss(err error) bool {
	return GetInfo(err).Code == CodeDataLoss
}

// IsUnauthenticated returns true if GetInfo(err).Code == CodeUnauthenticated.
func IsUnauthenticated(err error) bool {
	return GetInfo(err).Code == CodeUnauthenticated
}

func sprintf(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
