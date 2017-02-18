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
	switch {
	case from == to:
		// Decoding from the same type
		return data, nil
	case from == reflect.PtrTo(to):
		// Decoding from a pointer to the target type
		value := reflect.New(to).Elem()
		value.Set(reflect.ValueOf(data).Elem())
		return value.Interface(), nil
	case reflect.PtrTo(from) == to:
		// Decoding from a value to a pointer of the target type
		value := reflect.New(to.Elem())
		value.Elem().Set(reflect.ValueOf(data))
		return value.Interface(), nil
	}

	var (
		value reflect.Value
		dec   Decoder
	)

	if to.Kind() == reflect.Ptr && to.Implements(_typeOfDecoder) {
		value = reflect.New(to.Elem())
		dec = value.Interface().(Decoder)
	} else if reflect.PtrTo(to).Implements(_typeOfDecoder) {
		value = reflect.New(to).Elem()
		dec = value.Addr().Interface().(Decoder)
	} else {
		return data, nil
	}

	err := dec.Decode(decodeFrom(data))
	if err != nil {
		err = fmt.Errorf("could not decode %v: %v", to, err)
	}
	return value.Interface(), err
}
