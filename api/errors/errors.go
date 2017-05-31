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

	"go.uber.org/yarpc/yarpcproto"
)

// TODO: What to do with ERROR_TYPE_UNKNOWN? Should we use use this for ERROR_TYPE_APPLICATION?
// TODO: What to do with ERROR_TYPE_INTERNAL? Should we have a function to create an error of this type?
// TODO: should we have specific fields for each error type instead of keyValues ...string?

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

// Cancelled returns a new yarpc error with type ERROR_TYPE_CANCELLED.
func Cancelled(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_CANCELLED, keyValues)
}

// InvalidArgument returns a new yarpc error with type ERROR_TYPE_INVALID_ARGUMENT.
func InvalidArgument(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_INVALID_ARGUMENT, keyValues)
}

// DeadlineExceeded returns a new yarpc error with type ERROR_TYPE_DEADLINE_EXCEEDED.
func DeadlineExceeded(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_DEADLINE_EXCEEDED, keyValues)
}

// NotFound returns a new yarpc error with type ERROR_TYPE_NOT_FOUND.
func NotFound(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_NOT_FOUND, keyValues)
}

// AlreadyExists returns a new yarpc error with type ERROR_TYPE_ALREADY_EXISTS.
func AlreadyExists(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_ALREADY_EXISTS, keyValues)
}

// PermissionDenied returns a new yarpc error with type ERROR_TYPE_PERMISSION_DENIED.
func PermissionDenied(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_PERMISSION_DENIED, keyValues)
}

// ResourceExhausted returns a new yarpc error with type ERROR_TYPE_RESOURCE_EXHAUSTED.
func ResourceExhausted(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_RESOURCE_EXHAUSTED, keyValues)
}

// FailedPrecondition returns a new yarpc error with type ERROR_TYPE_FAILED_PRECONDITION.
func FailedPrecondition(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_FAILED_PRECONDITION, keyValues)
}

// Aborted returns a new yarpc error with type ERROR_TYPE_ABORTED.
func Aborted(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_ABORTED, keyValues)
}

// OutOfRange returns a new yarpc error with type ERROR_TYPE_OUT_OF_RANGE.
func OutOfRange(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_OUT_OF_RANGE, keyValues)
}

// Unimplemented returns a new yarpc error with type ERROR_TYPE_UNIMPLEMENTED.
func Unimplemented(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_UNIMPLEMENTED, keyValues)
}

// Internal returns a new yarpc error with type ERROR_TYPE_INTERNAL.
func Internal(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_INTERNAL, keyValues)
}

// Unavailable returns a new yarpc error with type ERROR_TYPE_UNAVAILABLE.
func Unavailable(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_UNAVAILABLE, keyValues)
}

// DataLoss returns a new yarpc error with type ERROR_TYPE_DATA_LOSS.
func DataLoss(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_DATA_LOSS, keyValues)
}

// Unauthenticated returns a new yarpc error with type ERROR_TYPE_UNAUTHENTICATED.
func Unauthenticated(keyValues ...string) error {
	return newWellKnownYarpcError(yarpcproto.ERROR_TYPE_UNAUTHENTICATED, keyValues)
}

// Application returns a new yarpc error with type ERROR_TYPE_APPLICATION and the given user-defined name.
//
// TODO: Validate that the name is only lowercase letters and dashes, and figure out what to do if not.
func Application(name string, keyValues ...string) error {
	return newUserDefinedYarpcError(name, keyValues)
}

type yarpcError struct {
	// Type is the type of the error. This should never be set to ERROR_TYPE_NONE.
	Type yarpcproto.ErrorType
	// Name is the user-defined name of the error. This is only valid if Type is
	// ERROR_TYPE_APPLICATION, otherwise the return value for Name will be empty.
	Name string
	// Details contains a map of additional details about the error.
	// The keys will be converted to all lower case, and two keys that are the same
	// when lowercase will have the latter key overridden.
	Details map[string]string

	orderedDetailsKeys []string
}

func (e *yarpcError) Error() string {
	// TODO: this needs escaping etc
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`type: `)
	_, _ = buffer.WriteString(strings.TrimPrefix(e.Type.String(), "ERROR_TYPE_"))
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

func newWellKnownYarpcError(errorType yarpcproto.ErrorType, keyValues []string) *yarpcError {
	details, orderedDetailsKeys := keyValueMapAndOrderedKeys(keyValues)
	return &yarpcError{
		Type:               errorType,
		Details:            details,
		orderedDetailsKeys: orderedDetailsKeys,
	}
}

func newUserDefinedYarpcError(name string, keyValues []string) *yarpcError {
	details, orderedDetailsKeys := keyValueMapAndOrderedKeys(keyValues)
	return &yarpcError{
		Type:               yarpcproto.ERROR_TYPE_APPLICATION,
		Name:               name,
		Details:            details,
		orderedDetailsKeys: orderedDetailsKeys,
	}
}

func keyValueMapAndOrderedKeys(keyValues []string) (map[string]string, []string) {
	if len(keyValues) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(keyValues)/2)
	orderedKeys := make([]string, 0, len(keyValues)/2)
	for i := 0; i < len(keyValues); i += 2 {
		key := canonicalizeMapKey(keyValues[i])
		var value string
		if i == len(keyValues)-1 {
			value = "missing"
		} else {
			value = keyValues[i+1]
		}
		if value != "" {
			m[key] = value
			orderedKeys = append(orderedKeys, key)
		}
	}
	return m, orderedKeys
}

func copyStringStringMap(m map[string]string) map[string]string {
	c := make(map[string]string, len(m))
	for key, value := range m {
		c[key] = value
	}
	return c
}

func canonicalizeMapKey(key string) string {
	return strings.ToLower(key)
}
