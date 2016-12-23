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

package internal

import (
	"fmt"
	"regexp"
)

var (
	// Must starts with [a-z].
	// Can contain [a-z] and 0-9] with non-consecutive dashes.
	validNameRegex = regexp.MustCompile("^[a-z]+([a-z0-9]|[^-]-)*[^-]$")

	// We disallow UUIDs explicitly (they would be accepted by validNameRegex alone)
	uuidRegex = regexp.MustCompile("[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}")
)

// ValidateServiceName returns an error if the given name is not a valid
// service name.
func ValidateServiceName(name string) error {
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("service name must begin with a letter and consist only of dash-delimited lower-case ASCII alphanumeric words")
	}
	if uuidRegex.MatchString(name) {
		return fmt.Errorf("service name must not contain a UUID")
	}
	return nil
}
