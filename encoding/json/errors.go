// Copyright (c) 2016 Uber Technologies, Inc.
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

package json

import "fmt"

// unmarshalError is returned when there's an error parsing JSON input.
type unmarshalError struct {
	Reason error
}

func (e unmarshalError) Error() string {
	return fmt.Sprintf("failed to parse JSON: %v", e.Reason)
}

// marshalError is returned when there's an error serializing to JSON.
type marshalError struct {
	Reason error
}

func (e marshalError) Error() string {
	return fmt.Sprintf("failed to serialize JSON: %v", e.Reason)
}

// IsEncodingError returns true if the given error is an error encountered
// while trying to serialize or deserialize JSON.
func IsEncodingError(e error) bool {
	switch e.(type) {
	case marshalError, unmarshalError:
		return true
	default:
		return false
	}
}
