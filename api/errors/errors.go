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
	"strings"

	"go.uber.org/yarpc/api/errors/codes"
)

// TODO: What to do with Unknown? Should we use use this for Application?
// TODO: What to do with Internal? Should we have a function to create an error of this code?
// TODO: should we have specific fields for each error code instead of keyValues ...string?

// Code will extract the Code from a yarpc error.
//
// This will return None if the given error is nil or not a yarpc error.
// This function is the defined way to test if an error is a yarpc error.
func Code(err error) codes.Code {
	if err == nil {
		return codes.None
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return codes.None
	}
	return yarpcError.Code
}

// Name will extract the user-defined error name from a yarpc error.
//
// This will return empty is the given error is nil, not a yarpc error, or the
// Code is not Application.
func Name(err error) string {
	if err == nil {
		return ""
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return ""
	}
	if yarpcError.Code != codes.Application {
		return ""
	}
	return yarpcError.Name
}

// Details will extract a copy the error details from a yarpc error.
//
// This will return empty if the given error is nil or not a yarpc error.
func Details(err error) map[string]string {
	if err == nil {
		return nil
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return nil
	}
	return copyStringStringMap(yarpcError.Details)
}

// WithKeyValues adds the given keyValues to the error.
//
// If the error is a yarpc error, this will just add the keyValues.
// If the error is not a yarpc error, this will return a new yarpc error of code Unknown
// with an additional keyValue pair of "error", err.Error().
// If keyValues contains a key of "error" and a new yarpc error is created, this key will be overwritten.
func WithKeyValues(err error, keyValues ...string) error {
	if err == nil {
		return nil
	}
	yarpcError, ok := err.(*yarpcError)
	if !ok {
		return newWellKnownYarpcError(codes.Unknown, append([]string{"error", err.Error()}, keyValues...))
	}
	c := yarpcError.copyAndAdd(keyValues...)
	return c
}

// Cancelled returns a new yarpc error with code Cancelled.
func Cancelled(keyValues ...string) error {
	return newWellKnownYarpcError(codes.Cancelled, keyValues)
}

// InvalidArgument returns a new yarpc error with code InvalidArgument.
func InvalidArgument(keyValues ...string) error {
	return newWellKnownYarpcError(codes.InvalidArgument, keyValues)
}

// DeadlineExceeded returns a new yarpc error with code DeadlineExceeded.
func DeadlineExceeded(keyValues ...string) error {
	return newWellKnownYarpcError(codes.DeadlineExceeded, keyValues)
}

// NotFound returns a new yarpc error with code NotFound.
func NotFound(keyValues ...string) error {
	return newWellKnownYarpcError(codes.NotFound, keyValues)
}

// AlreadyExists returns a new yarpc error with code AlreadyExists.
func AlreadyExists(keyValues ...string) error {
	return newWellKnownYarpcError(codes.AlreadyExists, keyValues)
}

// PermissionDenied returns a new yarpc error with code PermissionDenied.
func PermissionDenied(keyValues ...string) error {
	return newWellKnownYarpcError(codes.PermissionDenied, keyValues)
}

// ResourceExhausted returns a new yarpc error with code ResourceExhausted.
func ResourceExhausted(keyValues ...string) error {
	return newWellKnownYarpcError(codes.ResourceExhausted, keyValues)
}

// FailedPrecondition returns a new yarpc error with code FailedPrecondition.
func FailedPrecondition(keyValues ...string) error {
	return newWellKnownYarpcError(codes.FailedPrecondition, keyValues)
}

// Aborted returns a new yarpc error with code Aborted.
func Aborted(keyValues ...string) error {
	return newWellKnownYarpcError(codes.Aborted, keyValues)
}

// OutOfRange returns a new yarpc error with code OutOfRange.
func OutOfRange(keyValues ...string) error {
	return newWellKnownYarpcError(codes.OutOfRange, keyValues)
}

// Unimplemented returns a new yarpc error with code Unimplemented.
func Unimplemented(keyValues ...string) error {
	return newWellKnownYarpcError(codes.Unimplemented, keyValues)
}

// Internal returns a new yarpc error with code Internal.
func Internal(keyValues ...string) error {
	return newWellKnownYarpcError(codes.Internal, keyValues)
}

// Unavailable returns a new yarpc error with code Unavailable.
func Unavailable(keyValues ...string) error {
	return newWellKnownYarpcError(codes.Unavailable, keyValues)
}

// DataLoss returns a new yarpc error with code DataLoss.
func DataLoss(keyValues ...string) error {
	return newWellKnownYarpcError(codes.DataLoss, keyValues)
}

// Unauthenticated returns a new yarpc error with code Unauthenticated.
func Unauthenticated(keyValues ...string) error {
	return newWellKnownYarpcError(codes.Unauthenticated, keyValues)
}

// Application returns a new yarpc error with code Application and the given user-defined name.
//
// TODO: Validate that the name is only lowercase letters and dashes, and figure out what to do if not.
func Application(name string, keyValues ...string) error {
	return newUserDefinedYarpcError(name, keyValues)
}

type yarpcError struct {
	// Code is the code of the error. This should never be set to None.
	Code codes.Code
	// Name is the user-defined name of the error. This is only valid if Code is
	// Application, otherwise the return value for Name will be empty.
	Name string
	// Details contains a map of additional details about the error.
	// The keys will be converted to all lower case, and two keys that are the same
	// when lowercase will have the latter key overridden.
	Details map[string]string

	orderedDetailsKeys []string
}

func newWellKnownYarpcError(code codes.Code, keyValues []string) *yarpcError {
	e := &yarpcError{
		Code:               code,
		Details:            make(map[string]string, len(keyValues)/2),
		orderedDetailsKeys: make([]string, 0, len(keyValues)/2),
	}
	e.add(keyValues...)
	return e
}

func newUserDefinedYarpcError(name string, keyValues []string) *yarpcError {
	e := &yarpcError{
		Code:               codes.Application,
		Name:               name,
		Details:            make(map[string]string, len(keyValues)/2),
		orderedDetailsKeys: make([]string, 0, len(keyValues)/2),
	}
	e.add(keyValues...)
	return e
}

func (e *yarpcError) Error() string {
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`code: `)
	_, _ = buffer.WriteString(e.Code.String())
	if e.Name != "" {
		_, _ = buffer.WriteString(` name: `)
		_, _ = buffer.WriteString(e.Name)
	}
	if len(e.Details) != 0 {
		_, _ = buffer.WriteString(` details: `)
		for i, key := range e.orderedDetailsKeys {
			_, _ = buffer.WriteString(key)
			_, _ = buffer.WriteString(`:`)
			_, _ = buffer.WriteString(e.Details[key])
			if i != len(e.Details)-1 {
				_, _ = buffer.WriteString(` `)
			}
		}
	}
	return buffer.String()
}

func (e *yarpcError) copyAndAdd(keyValues ...string) *yarpcError {
	c := &yarpcError{
		Code:               e.Code,
		Name:               e.Name,
		Details:            copyStringStringMap(e.Details),
		orderedDetailsKeys: copyStringSlice(e.orderedDetailsKeys),
	}
	c.add(keyValues...)
	return c
}

func (e *yarpcError) add(keyValues ...string) {
	for i := 0; i < len(keyValues); i += 2 {
		key := strings.ToLower(keyValues[i])
		var value string
		if i == len(keyValues)-1 {
			value = "missing"
		} else {
			value = keyValues[i+1]
		}
		if value != "" {
			e.Details[key] = value
			e.orderedDetailsKeys = append(e.orderedDetailsKeys, key)
		}
	}
}

func copyStringStringMap(m map[string]string) map[string]string {
	c := make(map[string]string, len(m))
	for key, value := range m {
		c[key] = value
	}
	return c
}

func copyStringSlice(s []string) []string {
	c := make([]string, len(s))
	for i, value := range s {
		c[i] = value
	}
	return c
}
