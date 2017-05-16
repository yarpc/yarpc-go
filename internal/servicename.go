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

package internal

import (
	"errors"
	"regexp"
)

// We disallow UUIDs explicitly, though they're otherwise valid patterns.
var uuidRegex = regexp.MustCompile("[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}")

// ValidateServiceName returns an error if the given servive name is invalid.
// Valid names are at least two characters long, start with [a-z], contain only
// [0-9a-z] and non-consecutive hyphens, and end in [0-9a-z].
func ValidateServiceName(name string) error {
	if len(name) < 2 {
		return errors.New("service name must be at least two characters long")
	}
	if err := checkFirstCharacter(name); err != nil {
		return err
	}
	if err := checkForbiddenCharacters(name); err != nil {
		return err
	}
	if err := checkHyphens(name); err != nil {
		return err
	}
	if uuidRegex.MatchString(name) {
		return errors.New("service name must not contain a UUID")
	}
	return nil
}

func checkHyphens(name string) error {
	for i := 1; i < len(name); i++ {
		if name[i-1] == '-' && name[i] == '-' {
			return errors.New("service name must not contain consecutive hyphens")
		}
	}
	if name[len(name)-1] == '-' {
		return errors.New("service name must not end in a hyphen")
	}
	return nil
}

func checkFirstCharacter(name string) error {
	if name[0] < 'a' || name[0] > 'z' {
		return errors.New("service names must start with [a-z]")
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
			return errors.New("service name may only contain [0-9a-z] and non-consecutive hyphens")
		}
	}
	return nil
}
