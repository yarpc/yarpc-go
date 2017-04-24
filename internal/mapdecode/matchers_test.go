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
)

// structField is a gomock.Matcher that matches a StructField with the given
// parameters.
type structField struct {
	Name string
	Type reflect.Type
	Tag  string
}

func (m structField) String() string {
	return fmt.Sprintf("StructField{Name: %q, Type: %v}", m.Name, m.Type)
}

func (m structField) Matches(x interface{}) bool {
	s, ok := x.(reflect.StructField)
	if !ok {
		return false
	}

	return s.Name == m.Name && s.Type == m.Type && string(s.Tag) == m.Tag
}

// reflectEq is a gomock.Matcher that matches a reflect.Value whose underlying
// value matches the given value.
type reflectEq struct{ Value interface{} }

func (m reflectEq) String() string {
	return fmt.Sprintf("equal to %#v", m.Value)
}

func (m reflectEq) Matches(x interface{}) bool {
	v, ok := x.(reflect.Value)
	if !ok {
		return false
	}

	return reflect.DeepEqual(m.Value, v.Interface())
}
