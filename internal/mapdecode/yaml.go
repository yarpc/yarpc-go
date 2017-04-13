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

const _yamlTagName = "yaml"

// YAML may be specified to decode go-yaml (gopkg.in/yaml.v2) compatible
// types. Use this option if you have types that implement yaml.Unmarshaler
// and use the `yaml` tag.
//
// 	type StringSet map[string]struct{}
//
// 	func (ss *StringSet) UnmarshalYAML(decode func(interface{}) error) error {
// 		var items []string
// 		if err := decode(&items); err != nil {
// 			return err
// 		}
// 		// ..
// 	}
//
// 	var x StringSet
// 	mapdecode.Decode(&x, data, mapdecode.YAML())
//
// Caveat: None of the go-yaml flags are supported. Only the attribute name
// changes will be respected. Further, note that go-yaml ignores unused
// attributes but mapdecode fails on unused attributes by default. Use
// IgnoreUnused to cusotmize this behavior.
func YAML() Option {
	return func(o *options) {
		o.TagName = _yamlTagName
		o.Unmarshaler = _yamlUnmarshaler
	}
}

// yaml.Unmarshaler as defined in gopkg.in/yaml.v2
type yamlUnmarshaler interface {
	UnmarshalYAML(unmarshal func(interface{}) error) error
}

var _yamlUnmarshaler = unmarshaler{
	Interface: reflect.TypeOf((*yamlUnmarshaler)(nil)).Elem(),
	Unmarshal: func(v reflect.Value, into func(interface{}) error) error {
		return v.Interface().(yamlUnmarshaler).UnmarshalYAML(into)
	},
}
