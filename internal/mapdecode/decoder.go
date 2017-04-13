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

import "reflect"

var _decoderUnmarshaler = unmarshaler{
	Interface: reflect.TypeOf((*Decoder)(nil)).Elem(),
	Unmarshal: func(v reflect.Value, into func(interface{}) error) error {
		return v.Interface().(Decoder).Decode(Into(into))
	},
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
// 	func (ss *StringSet) Decode(into mapdecode.Into) error {
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
