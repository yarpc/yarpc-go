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

// Package decode implements a generic interface{} decoder. The intention is
// to use it to decode arbitrary map[interface{}]interface{} objects into
// structs or other complex objects.
package decode

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

const _tagName = "config"

var _typeOfDecoder = reflect.TypeOf((*Decoder)(nil)).Elem()

// Decode from src into dest. dest may implement Decoder to customize how src
// is read into it.
func Decode(dest, src interface{}) error {
	return decodeFrom(src)(dest)
}

// Decoder is any type which has custom decoding logic. Types may implement
// Decode and rely on the given Decode function to read values.
//
// For example,
//
// 	type StringSet map[string]struct{}
//
// 	func (ss *StringSet) Decode(into decode.Into) error {
// 		var items []string
// 		if err := into(&items); err != nil {
// 			return err
// 		}
//
// 		*ss = make(map[string]struct{})
// 		for _, item := range items {
// 			(*ss)[item] = struct{}{}
// 		}
// 		return nil
// 	}
type Decoder interface {
	// Decode receives a function that will attempt to decode the source data
	// into the given target. The argument MUST be a pointer to the target
	// object.
	Decode(Into) error
}

// Into is a function that attempts to decode the source data into the given
// destination. dest MUST be a pointer to a value.
//
// 	var (
// 		decode decode.Into = ...
// 		value map[string]MyStruct
// 	)
// 	err := decode(&value)
type Into func(dest interface{}) error

// decodeFrom builds a decode Into function that reads the given value into
// the destination.
func decodeFrom(src interface{}) Into {
	return func(dest interface{}) error {
		cfg := mapstructure.DecoderConfig{
			ErrorUnused: true,
			Result:      dest,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				decoderDecodeHook,
			),
			TagName: _tagName,
		}

		decoder, err := mapstructure.NewDecoder(&cfg)
		if err != nil {
			return fmt.Errorf("failed to set up decoder: %v", err)
		}

		return decoder.Decode(src)
	}
}

// decoderDecodeHook is a DecodeHook for mapstructure which recognizes types
// that implement the Decoder interface.
func decoderDecodeHook(from, to reflect.Type, data interface{}) (interface{}, error) {
	if data == nil {
		return data, nil
	}
	out, err := _decoderDecodeHook(from, to, reflect.ValueOf(data))
	return out.Interface(), err
}

func _decoderDecodeHook(from, to reflect.Type, data reflect.Value) (reflect.Value, error) {
	// Get rid of pointers in either direction. This lets us parse **foo into
	// a foo where *foo implements Decoder, for example.
	switch {
	case from == to:
		return data, nil
	case from.Kind() == reflect.Ptr: // *foo => foo
		return _decoderDecodeHook(from.Elem(), to, data.Elem())
	case to.Kind() == reflect.Ptr: // foo => *foo
		out, err := _decoderDecodeHook(from, to.Elem(), data)
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

	// After eliminating pointers, only destinations whose pointers implement
	// Decoder are supported. Everything else gets the value unchanged.

	if !reflect.PtrTo(to).Implements(_typeOfDecoder) {
		return data, nil
	}

	// value := new(foo)
	// err := value.Decode(...)
	// return *value, err
	value := reflect.New(to)
	err := value.Interface().(Decoder).Decode(decodeFrom(data.Interface()))
	if err != nil {
		err = fmt.Errorf("could not decode %v from %v: %v", to, from, err)
	}
	return value.Elem(), err
}
