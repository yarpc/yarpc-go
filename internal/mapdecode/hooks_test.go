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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultipleFieldHooks(t *testing.T) {
	var dest struct {
		Int int
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	hook1 := newMockFieldHook(mockCtrl)
	hook2 := newMockFieldHook(mockCtrl)

	typeOfInt := reflect.TypeOf(42)

	hook1.
		Expect(_typeOfEmptyInterface, structField{
			Name: "Int",
			Type: typeOfInt,
		}, reflectEq{"FOO"}).
		Return(valueOf("BAR"), nil)

	hook2.
		Expect(reflect.TypeOf(""), structField{
			Name: "Int",
			Type: typeOfInt,
		}, reflectEq{"BAR"}).
		Return(valueOf(42), nil)

	err := Decode(&dest, map[string]interface{}{"int": "FOO"},
		FieldHook(hook1.Hook()),
		FieldHook(hook2.Hook()),
	)
	require.NoError(t, err)

	assert.Equal(t, 42, dest.Int)
}

func TestMultipleDecodeHooks(t *testing.T) {
	type myStruct struct{ String string }

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	hook1 := newMockDecodeHook(mockCtrl)
	hook2 := newMockDecodeHook(mockCtrl)

	data := map[string]interface{}{"string": "hello world"}

	typeOfMapInterface := reflect.TypeOf(map[string]interface{}{})
	typeOfMyStruct := reflect.TypeOf(myStruct{})
	typeOfString := reflect.TypeOf("")

	gomock.InOrder(
		hook1.
			Expect(typeOfMapInterface, typeOfMyStruct, reflectEq{data}).
			Return(valueOf(data), nil),
		hook2.
			Expect(typeOfMapInterface, typeOfMyStruct, reflectEq{data}).
			Return(valueOf(data), nil),

		hook1.
			Expect(typeOfString, typeOfString, reflectEq{"hello world"}).
			Return(valueOf("foo bar"), nil),
		hook2.
			Expect(typeOfString, typeOfString, reflectEq{"foo bar"}).
			Return(valueOf("baz qux"), nil),
	)

	var dest myStruct
	err := Decode(&dest, data,
		DecodeHook(hook1.Hook()), DecodeHook(hook2.Hook()))
	require.NoError(t, err)
	assert.Equal(t, "baz qux", dest.String)
}
