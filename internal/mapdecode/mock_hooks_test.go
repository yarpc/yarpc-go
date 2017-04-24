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
	"reflect"

	"github.com/golang/mock/gomock"
)

// mockDecodeHook is a mock to control a function with the signature,
//
// 	func(reflect.Type, reflect.Type, reflect.Value) (reflect.Value, error)
//
// Expectations may be set on this function with the Expect function.
type mockDecodeHook struct{ c *gomock.Controller }

func newMockDecodeHook(ctrl *gomock.Controller) *mockDecodeHook {
	return &mockDecodeHook{c: ctrl}
}

// Hook returns the DecodeHookFunc backed by this mock.
func (m *mockDecodeHook) Hook() DecodeHookFunc {
	return DecodeHookFunc(m.Call)
}

// Expect sets up a call expectation on the hook.
func (m *mockDecodeHook) Expect(from, to, data interface{}) *gomock.Call {
	return m.c.RecordCall(m, "Call", from, to, data)
}

func (m *mockDecodeHook) Call(from reflect.Type, to reflect.Type, data reflect.Value) (reflect.Value, error) {
	results := m.c.Call(m, "Call", from, to, data)
	out := results[0].(reflect.Value)
	err, _ := results[1].(error)
	return out, err
}

// mockFieldHook is a mock to control a function with the signature,
//
// 	func(reflect.Type, reflect.StructField, reflect.Value) (reflect.Value, error)
//
// Expectations may be set on this function with the Expect function.
type mockFieldHook struct{ c *gomock.Controller }

func newMockFieldHook(ctrl *gomock.Controller) *mockFieldHook {
	return &mockFieldHook{c: ctrl}
}

// Hook returns the FieldHookFunc backed by this mock.
func (m *mockFieldHook) Hook() FieldHookFunc {
	return FieldHookFunc(m.Call)
}

// Expect sets up a call expectation on the hook.
func (m *mockFieldHook) Expect(from, to, data interface{}) *gomock.Call {
	return m.c.RecordCall(m, "Call", from, to, data)
}

func (m *mockFieldHook) Call(from reflect.Type, to reflect.StructField, data reflect.Value) (reflect.Value, error) {
	results := m.c.Call(m, "Call", from, to, data)
	out := results[0].(reflect.Value)
	err, _ := results[1].(error)
	return out, err
}
