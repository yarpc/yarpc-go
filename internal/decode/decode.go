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

// Package decode implements a generic interface{} decoder. It allows
// implementing custom YAML/JSON decoding logic only once. Instead of
// implementing UnmarshalYAML and UnmarshalJSON differently twice, you would
// implement Decode once, parse the YAML/JSON input into a
// map[string]interface{} and decode it using this package.
//
// 	var data map[string]interface{}
// 	if err := json.Decode(&data, input); err != nil {
// 		log.Fatal(err)
// 	}
//
//	var result MyStruct
// 	if err := decode.Decode(&result, data); err != nil {
// 		log.Fatal(err)
// 	}
//
// This also makes it possible to implement custom markup parsing and
// deserialization strategies that get decoded into a user-provided struct.
package decode

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

const _tagName = "config"

var _typeOfDecoder = reflect.TypeOf((*Decoder)(nil)).Elem()

// Decode from src into dest where dest is a pointer to the value being
// decoded.
//
// Primitives are mapped as-is with pointers created or dereferenced as
// necessary. Maps and slices use Decode recursively for each of their items.
// For structs, the source must be a map[string]interface{} or
// map[interface{}]interface{}. Each key in the map calls Decode recursively
// with the field of the struct that has a name similar to the key (case
// insensitive match).
//
// 	var item struct{ Key, Value string }
// 	err := Decode(&item, map[string]string{"key": "some key", "Value": "some value"})
//
// The name of the field in the map may be customized with the `config` tag.
//
// 	var item struct {
// 		Key   string `config:"name"`
// 		Value string
// 	}
// 	var item struct{ Key, Value string }
// 	err := Decode(&item, map[string]string{"name": "token", "Value": "some value"})
//
// The destination type or any subtype may implement the Decoder interface to
// customize how it gets decoded.
func Decode(dest, src interface{}) error {
	return decodeFrom(src)(dest)
}

// Decoder is any type which has custom decoding logic. Types may implement
// Decode and rely on the given Into function to read values into a different
// shape, validate the result, and fill themselves with it.
//
// For example the following lets users provide a list of strings to decode a
// set.
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
	// into the given target. The argument to Into MUST be a pointer to the
	// target object.
	Decode(Into) error
}

// Into is a function that attempts to decode the source data into the given
// shape.
//
// Types that implement Decoder are provided a reference to an Into object so
// that they can decode a different shape, validate the result and populate
// themselves with the result.
//
// 	var values []string
// 	err := into(&value)
// 	for _, value := range values {
// 		if value == "reserved" {
// 			return errors.New(`a value in the list cannot be "reserved"`)
// 		}
// 		self.Values = append(self.Values, value)
// 	}
//
// The function is safe to call multiple times if you need to try to decode
// different shapes. For example,
//
// 	// Allow the user to just use the string "default" for the default
// 	// configuration.
// 	var name string
// 	if err := into(&name); err == nil {
// 		if name == "default" {
// 			*self = DefaultConfiguration
// 			return
// 		}
// 		return fmt.Errorf("unknown name %q", name)
// 	}
//
// 	// Otherwise, the user must provide {someAttr: "value"} as the input for
// 	// explicit configuration.
// 	var custom struct{ SomeAttr string }
// 	if err := into(&custom); err != nil {
// 		return err
// 	}
//
// 	self.SomeAttr = custom
// 	return nil
//
// If the destination type or any sub-type implements Decoder, that function
// will be called. This means that Into MUST NOT be called on the type whose
// Decode function is currently running or this will end up in an infinite
// loop.
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
