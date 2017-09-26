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
//
// Strings will be all lowercase and not contain commas.
func (c Feature) String() string {
	s, ok := _featureToString[c]
	if ok {
		return s
	}
	return strconv.Itoa(int(c))
}

// FeatureFromString returns the Feature for the string, or false
// if the Feature is not known.
func FeatureFromString(s string) (Feature, bool) {
	f, ok := _stringToFeature[strings.ToLower(s)]
	if !ok {
		return Feature(0), false
	}
	return f, true
}
