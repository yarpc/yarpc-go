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

package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"go.uber.org/yarpc/internal/interpolate"
	"go.uber.org/yarpc/internal/mapdecode"
)

const (
	_tagName           = "config"
	_interpolateOption = "interpolate"
)

func decodeInto(dst, src interface{}) error {
	return mapdecode.Decode(dst, src,
		mapdecode.TagName(_tagName),
		mapdecode.FieldHook(interpolateHook))
}

func interpolateHook(from reflect.Type, to reflect.StructField, data reflect.Value) (reflect.Value, error) {
	shouldInterpolate := false

	options := strings.Split(to.Tag.Get(_tagName), ",")[1:]
	for _, option := range options {
		if option == _interpolateOption {
			shouldInterpolate = true
			break
		}
	}

	if !shouldInterpolate {
		return data, nil
	}

	// Use Interface().(string) so that we handle the case where data is an
	// interface{} holding a string.
	v, ok := data.Interface().(string)
	if !ok {
		// Cannot interpolate non-string type. This shouldn't be an error
		// because an integer field may be marked as interpolatable and may
		// have received an integer as expected.
		return data, nil
	}

	s, err := interpolate.Parse(v)
	if err != nil {
		return data, fmt.Errorf("failed to parse %q for interpolation: %v", v, err)
	}

	newV, err := s.Render(os.LookupEnv)
	if err != nil {
		return data, fmt.Errorf("failed to render %q with environment variables: %v", v, err)
	}

	return reflect.ValueOf(newV), nil
}
