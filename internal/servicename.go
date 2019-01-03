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

package internal

import (
	"errors"
	"fmt"
	"regexp"

	"go.uber.org/multierr"
)

// We disallow UUIDs explicitly, though they're otherwise valid patterns.
var _uuidRegexp = regexp.MustCompile("[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}")

// ValidateServiceName returns an error if the given servive name is invalid.
// Valid names are at least two characters long, start with [a-z], contain only
// [0-9a-z] and non-consecutive hyphens, and end in [0-9a-z]. Furthermore,
// names may not contain UUIDs.
func ValidateServiceName(name string) error {
	if len(name) < 2 {
		// Short names aren't safe to check any further.
		return errors.New("service name must be at least two characters long")
	}
	return multierr.Combine(
		checkHyphens(name),
		checkFirstCharacter(name),
		checkForbiddenCharacters(name),
		checkUUIDs(name),
	)
}

func checkHyphens(name string) error {
	for i := 1; i < len(name); i++ {
		if name[i-1] == '-' && name[i] == '-' {
			return fmt.Errorf("service name %q contains consecutive hyphens", name)
		}
	}
	if name[len(name)-1] == '-' {
		return fmt.Errorf("service name %q ends with a hyphen", name)
	}
	return nil
}

func checkFirstCharacter(name string) error {
	if name[0] < 'a' || name[0] > 'z' {
		return fmt.Errorf("service name %q doesn't start with a lowercase ASCII letter", name)
	}
	return nil
}

func checkForbiddenCharacters(name string) error {
	for _, c := range name {
		switch {
		case 'a' <= c && c <= 'z':
			continue
		case '0' <= c && c <= '9':
			continue
		case c == '-':
			continue
		default:
			return fmt.Errorf("service name %q contains characters other than [0-9a-z] and hyphens", name)
		}
	}
	return nil
}

func checkUUIDs(name string) error {
	if _uuidRegexp.MatchString(name) {
		return fmt.Errorf("service name %q contains a UUID", name)
	}
	return nil
}
