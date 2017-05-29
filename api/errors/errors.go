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

package errors

import (
	"bytes"
	"fmt"

	"go.uber.org/yarpc/yarpcproto"
)

// TODO: What to do with ERROR_TYPE_UNKNOWN? Should we use use this for ERROR_TYPE_APPLICATION?
// TODO: What to do with ERROR_TYPE_INTERNAL? Should we have a function to create an error of this type?
// TODO: should we have specific fields for each error type instead of format string, args ...interface{}?

// Type will extract the ErrorType from a yarpc error.
//
// This will return ERROR_TYPE_NONE if the given error is nil or not a yarpc error.
// This function is the defined way to test if an error is a yarpc error.
func Type(err error) yarpcproto.ErrorType {
	if err == nil {
		return yarpcproto.ERROR_TYPE_NONE
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return yarpcproto.ERROR_TYPE_NONE
	}
	return yarpcError.Type
}

// Name will extract the user-defined error name from a yarpc error.
//
// This will return empty is the given error is nil, not a yarpc error, or the
// ErrorType is not ERROR_TYPE_APPLICATION.
func Name(err error) string {
	if err == nil {
		return ""
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return ""
	}
	if yarpcError.Type != yarpcproto.ERROR_TYPE_APPLICATION {
		return ""
	}
	return yarpcError.Name
}

// Message will extract the error message from a yarpc error.
//
// This will return empty if the given error is nil or not a yarpc error.
func Message(err error) string {
	if err == nil {
		return ""
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return ""
	}
	return yarpcError.Message
}

// Cancelled returns a new yarpc error with type ERROR_TYPE_CANCELLED.
func Cancelled(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_CANCELLED, format, args...)
}

// InvalidArgument returns a new yarpc error with type ERROR_TYPE_INVALID_ARGUMENT.
func InvalidArgument(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_INVALID_ARGUMENT, format, args...)
}

// DeadlineExceeded returns a new yarpc error with type ERROR_TYPE_DEADLINE_EXCEEDED.
func DeadlineExceeded(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_DEADLINE_EXCEEDED, format, args...)
}

// NotFound returns a new yarpc error with type ERROR_TYPE_NOT_FOUND.
func NotFound(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_NOT_FOUND, format, args...)
}

// AlreadyExists returns a new yarpc error with type ERROR_TYPE_ALREADY_EXISTS.
func AlreadyExists(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_ALREADY_EXISTS, format, args...)
}

// PermissionDenied returns a new yarpc error with type ERROR_TYPE_PERMISSION_DENIED.
func PermissionDenied(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_PERMISSION_DENIED, format, args...)
}

// ResourceExhausted returns a new yarpc error with type ERROR_TYPE_RESOURCE_EXHAUSTED.
func ResourceExhausted(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_RESOURCE_EXHAUSTED, format, args...)
}

// FailedPrecondition returns a new yarpc error with type ERROR_TYPE_FAILED_PRECONDITION.
func FailedPrecondition(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_FAILED_PRECONDITION, format, args...)
}

// Aborted returns a new yarpc error with type ERROR_TYPE_ABORTED.
func Aborted(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_ABORTED, format, args...)
}

// OutOfRange returns a new yarpc error with type ERROR_TYPE_OUT_OF_RANGE.
func OutOfRange(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_OUT_OF_RANGE, format, args...)
}

// Unimplemented returns a new yarpc error with type ERROR_TYPE_UNIMPLEMENTED.
func Unimplemented(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_UNIMPLEMENTED, format, args...)
}

// Unavailable returns a new yarpc error with type ERROR_TYPE_UNAVAILABLE.
func Unavailable(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_UNAVAILABLE, format, args...)
}

// DataLoss returns a new yarpc error with type ERROR_TYPE_DATA_LOSS.
func DataLoss(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_DATA_LOSS, format, args...)
}

// Unauthenticated returns a new yarpc error with type ERROR_TYPE_UNAUTHENTICATED.
func Unauthenticated(format string, args ...interface{}) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_UNAUTHENTICATED, format, args...)
}

// Application returns a new yarpc error with type ERROR_TYPE_APPLICATION and the given user-defined name.
//
// TODO: Validate that the name is only lowercase letters and dashes, and figure out what to do if not.
func Application(name string, format string, args ...interface{}) error {
	return newUserDefinedYarpcError(name, format, args...)
}

type yarpcError struct {
	// Type is the type of the error. This should never be set to ERROR_TYPE_NONE.
	Type yarpcproto.ErrorType
	// Name is the user-defined name of the error. This is only valid if Type is
	// ERROR_TYPE_APPLICATION, otherwise the return value for Name will be empty.
	Name string
	// Message contains the error message.
	Message string
}

func (e *yarpcError) Error() string {
	// TODO: this needs escaping etc
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`{"type":"`)
	_, _ = buffer.WriteString(e.Type.String())
	_, _ = buffer.WriteString(`"`)
	if e.Name != "" {
		_, _ = buffer.WriteString(`, "name":"`)
		_, _ = buffer.WriteString(e.Name)
		_, _ = buffer.WriteString(`"`)
	}
	if e.Message != "" {
		_, _ = buffer.WriteString(`, "message":"`)
		_, _ = buffer.WriteString(e.Message)
		_, _ = buffer.WriteString(`"`)
	}
	_, _ = buffer.WriteString(`}`)
	return buffer.String()
}

func newWellKnownYarpcError(errorType yarpcproto.ErrorType, format string, args ...interface{}) *yarpcError {
	return &yarpcError{
		Type:    errorType,
		Message: fmt.Sprintf(format, args...),
	}
}

func newUserDefinedYarpcError(name string, format string, args ...interface{}) *yarpcError {
	return &yarpcError{
		Type:    yarpcproto.ERROR_TYPE_APPLICATION,
		Name:    name,
		Message: fmt.Sprintf(format, args...),
	}
}
