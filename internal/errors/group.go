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

import "strings"

// ErrorGroup represents a collection of errors.
type ErrorGroup []error

func (e ErrorGroup) Error() string {
	messages := make([]string, 0, len(e)+1)
	messages = append(messages, "the following errors occurred:")
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "\n\t")
}

// MultiError combines a list of errors into one. The list MUST NOT contain nil.
//
// Returns nil if the error list is empty.
func MultiError(errors []error) error {
	switch len(errors) {
	case 0:
		return nil
	case 1:
		return errors[0]
	}

	newErrors := make(ErrorGroup, 0, len(errors))
	for _, err := range errors {
		switch e := err.(type) {
		case ErrorGroup:
			newErrors = append(newErrors, e...)
		default:
			newErrors = append(newErrors, e)
		}
	}

	return newErrors
}

// CombineErrors combines the given collection of errors together. nil values
// will be ignored.
//
// The intention for this is to help chain togeter errors from multiple failing
// operations.
//
// 	CombineErrors(
// 		reader.Close(),
// 		writer.Close(),
// 	)
//
// This may also be used like so,
//
// 	err := reader.Close()
// 	err = internal.CombineErrors(err, writer.Close())
// 	if someCondition {
// 		err = internal.CombineErrors(err, transport.Close())
// 	}
func CombineErrors(errors ...error) error {
	newErrors := errors[:0] // zero-alloc filtering
	for _, err := range errors {
		if err != nil {
			newErrors = append(newErrors, err)
		}
	}

	return MultiError(newErrors)
}
