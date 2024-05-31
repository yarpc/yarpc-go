// Copyright (c) 2024 Uber Technologies, Inc.
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

	"github.com/uber-go/mapdecode"
)

// AttributeMap is a convenience type on top of a map
// that gives us a cleaner interface to validate config values.
type AttributeMap map[string]interface{}

// PopString will pop a value from the attribute map and return the string
// it points to, or an error if it couldn't pop the value and decode.
func (m AttributeMap) PopString(name string) (string, error) {
	var s string
	_, err := m.Pop(name, &s)
	return s, err
}

// PopBool will pop a value from the attribute map and return the bool
// it points to, or an error if it couldn't pop the value and decode.
func (m AttributeMap) PopBool(name string) (bool, error) {
	var b bool
	_, err := m.Pop(name, &b)
	return b, err
}

// Pop removes the named key from the AttributeMap and decodes the value into
// the dst interface.
func (m AttributeMap) Pop(name string, dst interface{}, opts ...mapdecode.Option) (bool, error) {
	ok, err := m.Get(name, dst, opts...)
	if ok {
		delete(m, name)
	}
	return ok, err
}

// Get grabs a value from the attribute map and decodes it into the dst
// interface.
func (m AttributeMap) Get(name string, dst interface{}, opts ...mapdecode.Option) (bool, error) {
	v, ok := m[name]
	if !ok {
		return ok, nil
	}

	err := DecodeInto(dst, v, opts...)
	if err != nil {
		err = fmt.Errorf("failed to read attribute %q: %v. got error: %s", name, v, err)
	}
	return true, err
}

// Keys returns all the keys of the attribute map.
func (m AttributeMap) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// Decode attempts to decode the AttributeMap into the dst interface.
func (m AttributeMap) Decode(dst interface{}, opts ...mapdecode.Option) error {
	return DecodeInto(dst, m, opts...)
}
