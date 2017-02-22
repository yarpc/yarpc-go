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

// Into is a function that attempts to decode the source data into the given
// destination. dst MUST be a pointer to a value.
//
// 	var (
// 		decode decode.Into = ...
// 		value map[string]MyStruct
// 	)
// 	err := decode(&value)
type Into func(dst interface{}) error

// Decoder is any type which has custom decoding logic. Types may implement
// Decode and rely on the given Decode function to read values.
//
// For example,
//
// 	type StringSet map[string]struct{}
//
// 	func (ss *StringSet) Decode(dec decode.Into) error {
// 		var items []string
// 		if err := dec(&items); err != nil {
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

// Decode from src into dst. dst may implement Decoder to customize how src is
// read into it.
func Decode(dst, src interface{}) error {
	return decodeFrom(src)(dst)
}

// decodeFrom builds a decode Into function that reads the given value into
// the destination.
func decodeFrom(src interface{}) Into {
	return func(dst interface{}) error {
		cfg := mapstructure.DecoderConfig{
			ErrorUnused: true,
			Result:      dst,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				decoderDecodeHook,
			),
			TagName: _tagName,
		}

		dec, err := mapstructure.NewDecoder(&cfg)
		if err != nil {
			return fmt.Errorf("failed to set up decoder: %v", err)
		}

		return dec.Decode(src)
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
