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

// Package mapdecode implements a generic interface{} decoder. It allows
// implementing custom YAML/JSON decoding logic only once. Instead of
// implementing the same UnmarshalYAML and UnmarshalJSON twice, you can
// implement Decode once, parse the YAML/JSON input into a
// map[string]interface{} and decode it using this package.
//
// 	var data map[string]interface{}
// 	if err := json.Decode(&data, input); err != nil {
// 		log.Fatal(err)
// 	}
//
//	var result MyStruct
// 	if err := mapdecode.Decode(&result, data); err != nil {
// 		log.Fatal(err)
// 	}
//
// This also makes it possible to implement custom markup parsing and
// deserialization strategies that get decoded into a user-provided struct.
package mapdecode

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"go.uber.org/multierr"
)

const _defaultTagName = "mapdecode"

type options struct {
	TagName      string
	IgnoreUnused bool
	Unmarshaler  unmarshaler
}

// Option customizes the behavior of Decode.
type Option func(*options)

// TagName changes the name of the struct tag under which field names are
// expected.
func TagName(name string) Option {
	return func(o *options) {
		o.TagName = name
	}
}

// IgnoreUnused specifies whether we should ignore unused attributes in YAML.
// By default, decoding will fail if an unused attribute is encountered.
func IgnoreUnused(ignore bool) Option {
	return func(o *options) {
		o.IgnoreUnused = ignore
	}
}

// unmarshaler defines a scheme that allows users to do custom unmarshalling.
// The default scheme is _decoderUnmarshaler where we expect users to
// implement the Decoder interface.
type unmarshaler struct {
	// Interface that the type must implement for Unmarshal to be called.
	Interface reflect.Type

	// Unmarshal will be called with a Value that implements the interface
	// specified above and a function to decode the underlying data into
	// another shape. This is analogous to the Into type.
	Unmarshal func(reflect.Value, func(interface{}) error) error
}

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
// The name of the field in the map may be customized with the `mapdecode`
// tag. (Use the TagName option to change the name of the tag.)
//
// 	var item struct {
// 		Key   string `mapdecode:"name"`
// 		Value string
// 	}
// 	var item struct{ Key, Value string }
// 	err := Decode(&item, map[string]string{"name": "token", "Value": "some value"})
//
// The destination type or any subtype may implement the Decoder interface to
// customize how it gets decoded.
func Decode(dest, src interface{}, os ...Option) error {
	opts := options{
		TagName:     _defaultTagName,
		Unmarshaler: _decoderUnmarshaler,
	}
	for _, o := range os {
		o(&opts)
	}
	return decodeFrom(&opts, src)(dest)
}

// decodeFrom builds a decode Into function that reads the given value into
// the destination.
func decodeFrom(opts *options, src interface{}) Into {
	return func(dest interface{}) error {
		cfg := mapstructure.DecoderConfig{
			ErrorUnused: !opts.IgnoreUnused,
			Result:      dest,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				decoderDecodeHook(opts),
			),
			TagName: opts.TagName,
		}

		decoder, err := mapstructure.NewDecoder(&cfg)
		if err != nil {
			return fmt.Errorf("failed to set up decoder: %v", err)
		}

		if err := decoder.Decode(src); err != nil {
			if merr, ok := err.(*mapstructure.Error); ok {
				return multierr.Combine(merr.WrappedErrors()...)
			}
			return err
		}

		return nil
	}
}

// decoderDecodeHook builds a DecodeHook for mapstructure which recognizes
// types that implement the Decoder interface.
func decoderDecodeHook(opts *options) mapstructure.DecodeHookFuncType {
	return func(from, to reflect.Type, data interface{}) (interface{}, error) {
		if data == nil {
			return data, nil
		}
		out, err := _decoderDecodeHook(opts, from, to, reflect.ValueOf(data))
		return out.Interface(), err
	}
}

func _decoderDecodeHook(
	opts *options, from, to reflect.Type, data reflect.Value) (reflect.Value, error) {
	// Get rid of pointers in either direction. This lets us parse **foo into
	// a foo where *foo implements Decoder, for example.
	switch {
	case from == to:
		return data, nil
	case from.Kind() == reflect.Ptr: // *foo => foo
		return _decoderDecodeHook(opts, from.Elem(), to, data.Elem())
	case to.Kind() == reflect.Ptr: // foo => *foo
		out, err := _decoderDecodeHook(opts, from, to.Elem(), data)
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
