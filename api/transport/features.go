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

package transport

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// FeatureThriftApplicationError says that the client can handle thrift
	// application errors returned over the wire, not just in the thrift
	// envelope. This may involve a transport-specific header being returned
	// on the response to indicate if an error is an application error or not.
	FeatureThriftApplicationError = 1
)

var (
	_featureToString = map[Feature]string{
		FeatureThriftApplicationError: "1",
	}
	_stringToFeature = map[string]Feature{
		"1": FeatureThriftApplicationError,
	}
)

// Feature is a feature that the client can support.
//
// This makes it easier to add new features to YARPC in a backwards-compatible
// manner so that servers know how to construct responses.
type Feature int

// String returns the the string representation of the Feature.
func (c Feature) String() string {
	s, ok := _featureToString[c]
	if ok {
		return s
	}
	return strconv.Itoa(int(c))
}

// MarshalText implements encoding.TextMarshaler.
func (c Feature) MarshalText() ([]byte, error) {
	s, ok := _featureToString[c]
	if ok {
		return []byte(s), nil
	}
	return nil, fmt.Errorf("unknown feature: %d", int(c))
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *Feature) UnmarshalText(text []byte) error {
	i, ok := _stringToFeature[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("unknown feature string: %s", string(text))
	}
	*c = i
	return nil
}

// MarshalJSON implements json.Marshaler.
func (c Feature) MarshalJSON() ([]byte, error) {
	s, ok := _featureToString[c]
	if ok {
		return []byte(`"` + s + `"`), nil
	}
	return nil, fmt.Errorf("unknown feature: %d", int(c))
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *Feature) UnmarshalJSON(text []byte) error {
	s := string(text)
	if len(s) < 3 || s[0] != '"' || s[len(s)-1] != '"' {
		return fmt.Errorf("invalid feature string: %s", s)
	}
	i, ok := _stringToFeature[strings.ToLower(s[1:len(s)-1])]
	if !ok {
		return fmt.Errorf("unknown feature string: %s", s)
	}
	*c = i
	return nil
}
