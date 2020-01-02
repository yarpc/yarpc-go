// Copyright (c) 2020 Uber Technologies, Inc.
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
	"reflect"
	"strings"

	"github.com/uber-go/mapdecode"
	"go.uber.org/yarpc/internal/interpolate"
)

const (
	_tagName           = "config"
	_interpolateOption = "interpolate"
)

// DecodeInto will decode the src's data into the dst interface.
func DecodeInto(dst interface{}, src interface{}, opts ...mapdecode.Option) error {
	opts = append(opts, mapdecode.TagName(_tagName))
	return mapdecode.Decode(dst, src, opts...)
}

// InterpolateWith is a MapDecode option that will read a structField's tag
// information, and if the `interpolate` option is set, it will use the
// interpolate resolver to alter data as it's being decoded into the struct.
func InterpolateWith(resolver interpolate.VariableResolver) mapdecode.Option {
	return mapdecode.FieldHook(func(dest reflect.StructField, srcData reflect.Value) (reflect.Value, error) {
		shouldInterpolate := false

		options := strings.Split(dest.Tag.Get(_tagName), ",")[1:]
		for _, option := range options {
			if option == _interpolateOption {
				shouldInterpolate = true
				break
			}
		}

		if !shouldInterpolate {
			return srcData, nil
		}

		// Use Interface().(string) so that we handle the case where data is an
		// interface{} holding a string.
		v, ok := srcData.Interface().(string)
		if !ok {
			// Cannot interpolate non-string type. This shouldn't be an error
			// because an integer field may be marked as interpolatable and may
			// have received an integer as expected.
			return srcData, nil
		}

		s, err := interpolate.Parse(v)
		if err != nil {
			return srcData, fmt.Errorf("failed to parse %q for interpolation: %v", v, err)
		}

		newV, err := s.Render(resolver)
		if err != nil {
			return srcData, fmt.Errorf("failed to render %q with environment variables: %v", v, err)
		}

		return reflect.ValueOf(newV), nil
	})
}
