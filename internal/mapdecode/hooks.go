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

package mapdecode

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
)

var _typeOfDuration = reflect.TypeOf(time.Duration(0))

// reflectHook is similar to mapstructure's decode hooks except it operates on
// the reflected values rather than interface{}.
type reflectHook func(from, to reflect.Type, data reflect.Value) (reflect.Value, error)

// Builds a mapstructure-compatible hook from a reflectHook.
func fromReflectHook(hook reflectHook) mapstructure.DecodeHookFuncType {
	return func(from, to reflect.Type, data interface{}) (interface{}, error) {
		var value reflect.Value
		if data != nil {
			value = reflect.ValueOf(data)
		} else {
			// mapstructure is pretty good about giving us non-nil data but
			// let's process it gracefully anyway.
			value = reflect.Zero(from)
		}

		out, err := hook(from, to, value)
		if err != nil {
			return nil, err
		}

		return out.Interface(), nil
	}
}

// Compposes multiple reflectHooks into one. The hooks are applied in-order
// and values produced by a hook are fed into the next hook.
func composeReflectHooks(hooks ...reflectHook) reflectHook {
	return func(from, to reflect.Type, data reflect.Value) (reflect.Value, error) {
		var err error
		for _, hook := range hooks {
			data, err = hook(from, to, data)
			if err != nil {
				return data, err
			}

			// Update the `from` type to reflect changes made by the hook.
			from = data.Type()
		}
		return data, err
	}
}

// Wraps a reflectHook to support pointers in either direction (source and
// destination).
func supportPointers(hook reflectHook) (outputHook reflectHook) {
	outputHook = func(from, to reflect.Type, data reflect.Value) (reflect.Value, error) {
		// Get rid of pointers in either direction. This lets us parse **foo if we
		// know how to parse foo.
		switch {
		case from == to:
			return data, nil
		case from.Kind() == reflect.Ptr: // *foo => bar
			// Decoding a pointer type to a non-pointer type. Dereference if
			// non-nil, use zero-value otherwise.
			from = from.Elem()
			if data.IsNil() {
				data = reflect.Zero(from)
			} else {
				data = data.Elem()
			}
			return outputHook(from, to, data)
		case to.Kind() == reflect.Ptr: // foo => *bar
			// Decoding a non-pointer type to a pointer. Decode as usual and take
			// a pointer to the result.
			out, err := outputHook(from, to.Elem(), data)
			if err != nil {
				return out, err
			}

			// If we didn't know what to do with the input, the returned value
			// will just be the data as-is and it won't have the correct type.
			if !out.Type().AssignableTo(to.Elem()) {
				return data, nil
			}

			result := reflect.New(to.Elem())
			result.Elem().Set(out)
			return result, nil
		}

		return hook(from, to, data)
	}
	return
}

// Builds a reflectHook which unmarshals types using the given unmarshaling
// scheme. See the unmarshaler type for more information.
func unmarshalerHook(opts *options) reflectHook {
	return func(from, to reflect.Type, data reflect.Value) (reflect.Value, error) {
		if !reflect.PtrTo(to).Implements(opts.Unmarshaler.Interface) {
			return data, nil
		}

		// The following lines are roughly equivalent to,
		// 	value := new(foo)
		// 	err := value.Decode(...)
		// 	return *value, err
		value := reflect.New(to)
		err := opts.Unmarshaler.Unmarshal(value, decodeFrom(opts, data.Interface()))
		if err != nil {
			err = fmt.Errorf("could not decode %v from %v: %v", to, from, err)
		}
		return value.Elem(), err
	}
}

// A reflectHook which decodes strings into time.Durations.
func durationHook(from, to reflect.Type, data reflect.Value) (reflect.Value, error) {
	if from.Kind() != reflect.String || to != _typeOfDuration {
		return data, nil
	}

	d, err := time.ParseDuration(data.String())
	return reflect.ValueOf(d), err
}

// stringToPrimitivesHook is a reflectHook which decodes strings into
// primitives.
//
// Integers are parsed in base 10.
func strconvHook(from, to reflect.Type, data reflect.Value) (reflect.Value, error) {
	if from.Kind() != reflect.String {
		return data, nil
	}

	s := data.String()
	switch to.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		return reflect.ValueOf(b), err
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, to.Bits())
		return reflect.ValueOf(f).Convert(to), err
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, to.Bits())
		return reflect.ValueOf(i).Convert(to), err
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, to.Bits())
		return reflect.ValueOf(u).Convert(to), err
	}

	return data, nil
}
