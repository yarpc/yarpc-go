// Copyright (c) 2016 Uber Technologies, Inc.
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

package thrift

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/ast"
	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/wire"
)

func makeWireList(t wire.Type, num int, f func(i int) wire.Value) wire.Value {
	elems := make([]wire.Value, num)
	for i := range elems {
		elems[i] = f(i)
	}

	return wire.NewValueList(wire.ValueListFromSlice(t, elems))
}

func makeWireSet(t wire.Type, num int, f func(i int) wire.Value) wire.Value {
	// Since wire.Set and wire.List are the exact same type, we can cast between them.
	vlist := makeWireList(t, num, f)
	return wire.NewValueSet(vlist.GetList())
}

func makeWireMap(keyType, valueType wire.Type, num int, f func(i int) (key, value wire.Value)) wire.Value {
	elems := make([]wire.MapItem, num)
	for i := range elems {
		k, v := f(i)
		elems[i] = wire.MapItem{Key: k, Value: v}
	}

	return wire.NewValueMap(wire.MapItemListFromSlice(keyType, valueType, elems))
}

func TestValueFromWireSuccess(t *testing.T) {
	i32s := func(nums ...int) []interface{} {
		result := make([]interface{}, len(nums))
		for i, num := range nums {
			result[i] = int32(num)
		}
		return result
	}

	tests := []struct {
		w    wire.Value
		spec compile.TypeSpec
		v    interface{}

		skipToWire bool
	}{
		{
			w:    wire.NewValueBool(true),
			spec: &compile.BoolSpec{},
			v:    true,
		},
		{
			w:    wire.NewValueI8(8),
			spec: &compile.I8Spec{},
			v:    int8(8),
		},
		{
			w:    wire.NewValueI16(16),
			spec: &compile.I16Spec{},
			v:    int16(16),
		},
		{
			w:    wire.NewValueI32(32),
			spec: &compile.I32Spec{},
			v:    int32(32),
		},
		{
			w:    wire.NewValueI64(64),
			spec: &compile.I64Spec{},
			v:    int64(64),
		},
		{
			w:    wire.NewValueDouble(1.45),
			spec: &compile.DoubleSpec{},
			v:    1.45,
		},
		{
			w:    wire.NewValueString("str"),
			spec: &compile.StringSpec{},
			v:    "str",
		},
		{
			w:    wire.NewValueBinary([]byte("foo")),
			spec: &compile.BinarySpec{},
			v:    []byte("foo"),
		},
		{
			w: wire.NewValueBinary([]byte("foo")),
			spec: &compile.TypedefSpec{
				Name:   "Blob",
				Target: &compile.BinarySpec{},
			},
			v: []byte("foo"),
		},
		{
			w: wire.NewValueString("str"),
			spec: &compile.TypedefSpec{
				Name:   "UUID",
				Target: &compile.StringSpec{},
			},
			v: "str",
		},
		{
			w: wire.NewValueString("str"),
			spec: &compile.TypedefSpec{
				Name: "UUID1",
				Target: &compile.TypedefSpec{
					Name:   "UUID2",
					Target: &compile.StringSpec{},
				},
			},
			v: "str",
		},
		{
			w: makeWireList(wire.TI32, 4, func(i int) wire.Value {
				return wire.NewValueI32(int32(i))
			}),
			spec: &compile.ListSpec{ValueSpec: &compile.I32Spec{}},
			v:    i32s(0, 1, 2, 3),
		},
		{
			w: makeWireList(wire.TI32, 4, func(i int) wire.Value {
				return wire.NewValueI32(int32(i))
			}),
			spec: &compile.TypedefSpec{
				Name: "ListOfInts",
				Target: &compile.ListSpec{
					ValueSpec: &compile.TypedefSpec{
						Name:   "Int",
						Target: &compile.I32Spec{},
					},
				},
			},
			v: i32s(0, 1, 2, 3),
		},
		{
			w: makeWireSet(wire.TI32, 4, func(i int) wire.Value {
				return wire.NewValueI32(10 + int32(i))
			}),
			spec: &compile.SetSpec{ValueSpec: &compile.I32Spec{}},
			v:    i32s(10, 11, 12, 13),
		},
		{
			w: makeWireMap(wire.TBinary, wire.TBinary, 3, func(i int) (key, value wire.Value) {
				return wire.NewValueString(fmt.Sprintf("%v-k", i)), wire.NewValueString(fmt.Sprintf("%v-v", i))
			}),
			spec: &compile.MapSpec{
				KeySpec:   &compile.StringSpec{},
				ValueSpec: &compile.StringSpec{},
			},
			v: map[string]interface{}{
				"0-k": "0-v",
				"1-k": "1-v",
				"2-k": "2-v",
			},
		},
		{
			// map<list<string>,string>
			w: makeWireMap(wire.TList, wire.TBinary, 3, func(i int) (key, value wire.Value) {
				key = makeWireList(wire.TBinary, i, func(i int) wire.Value {
					return wire.NewValueString(fmt.Sprintf("k%v", i))
				})
				return key, wire.NewValueString(fmt.Sprintf("%v-v", i))
			}),
			spec: &compile.MapSpec{
				KeySpec:   &compile.ListSpec{ValueSpec: &compile.StringSpec{}},
				ValueSpec: &compile.StringSpec{},
			},
			// The key is JSON serialized if it's not a string.
			v: map[string]interface{}{
				"[]":          "0-v",
				`["k0"]`:      "1-v",
				`["k0","k1"]`: "2-v",
			},
		},
		{
			// list<list<int>>
			w: makeWireList(wire.TList, 3, func(i int) wire.Value {
				return makeWireList(wire.TI32, i+1, func(i int) wire.Value {
					return wire.NewValueI32(int32(i))
				})
			}),
			spec: &compile.ListSpec{ValueSpec: &compile.ListSpec{ValueSpec: &compile.I32Spec{}}},
			v: []interface{}{
				i32s(0),
				i32s(0, 1),
				i32s(0, 1, 2),
			},
		},
		{
			// struct S {1: s string}
			w: wire.NewValueStruct(wire.Struct{
				Fields: []wire.Field{{
					ID:    1,
					Value: wire.NewValueString("foo"),
				}},
			}),
			spec: &compile.StructSpec{
				Name: "S",
				Type: ast.StructType,
				Fields: compile.FieldGroup{{
					ID:   1,
					Name: "s",
					Type: &compile.StringSpec{},
				}},
			},
			v: map[string]interface{}{
				"s": "foo",
			},
		},
		{
			// struct S {1: s string}
			w: wire.NewValueStruct(wire.Struct{
				Fields: []wire.Field{{
					ID:    1,
					Value: wire.NewValueString("foo"),
				}},
			}),
			spec: &compile.TypedefSpec{
				Name: "T",
				Target: &compile.StructSpec{
					Name: "S",
					Type: ast.StructType,
					Fields: compile.FieldGroup{{
						ID:   1,
						Name: "s",
						Type: &compile.TypedefSpec{
							Name:   "UUID",
							Target: &compile.StringSpec{},
						},
					}},
				},
			},
			v: map[string]interface{}{
				"s": "foo",
			},
		},
		{
			// struct S {}, unknown field shouldn't cause an error.
			// TODO: should we add unknown fields to the result with a special _unknown_field_1 key?
			w: wire.NewValueStruct(wire.Struct{
				Fields: []wire.Field{{
					ID:    1,
					Value: wire.NewValueString("foo"),
				}},
			}),
			spec: &compile.StructSpec{
				Name:   "S",
				Type:   ast.StructType,
				Fields: compile.FieldGroup{},
			},
			v:          map[string]interface{}{},
			skipToWire: true, // reason: unrecognized field
		},
		{
			// struct S {1: optional string s = 'foo'}, default fields should always be set.
			w: wire.NewValueStruct(wire.Struct{}),
			spec: &compile.StructSpec{
				Name: "S",
				Type: ast.StructType,
				Fields: compile.FieldGroup{{
					ID:      1,
					Name:    "s",
					Type:    &compile.StringSpec{},
					Default: compile.ConstantString("foo"),
				}},
			},
			v: map[string]interface{}{
				"s": "foo",
			},
			skipToWire: true, // reason: default value
		},
		{
			// Enum with recognized value.
			w: wire.NewValueI32(1),
			spec: &compile.EnumSpec{
				Name: "Op",
				Items: []compile.EnumItem{{
					Name:  "Add",
					Value: 1,
				}},
			},
			v: "Add",
		},
		{
			// Enum with with unrecognized value.
			w: wire.NewValueI32(1),
			spec: &compile.EnumSpec{
				Name:  "Op",
				Items: nil,
			},
			v: "Op(1)",
		},
		{
			// Enum with duplicated value.
			w: wire.NewValueI32(1),
			spec: &compile.EnumSpec{
				Name: "Op",
				Items: []compile.EnumItem{
					{
						Name:  "Add",
						Value: 1,
					},
					{
						Name:  "Sum",
						Value: 1,
					},
				},
			},
			v: "Add",
		},
		{
			// Struct with default int.
			w: wire.NewValueStruct(wire.Struct{}),
			spec: &compile.StructSpec{
				Name: "S",
				Type: ast.StructType,
				Fields: compile.FieldGroup{{
					ID:      1,
					Name:    "s",
					Type:    &compile.I32Spec{},
					Default: compile.ConstantInt(1),
				}},
			},
			v: map[string]interface{}{
				"s": int64(1),
			},
			skipToWire: true, // defaults are not serialized.
		},
	}

	for _, tt := range tests {
		spec, err := tt.spec.Link(compile.EmptyScope("fake"))
		require.NoError(t, err, "Failed to link %v", tt.spec)

		got, err := FromWireValue(spec, tt.w)
		if assert.NoError(t, err, "Failed for FromWireValue(%v, %v)", spec, tt.w) {
			assert.Equal(t, tt.v, got, "Unexpected value for FromWireValue(%v, %v)", tt.spec, tt.w)
		}

		if tt.skipToWire {
			continue
		}
		w, err := ToWireValue(spec, tt.v)
		if assert.NoError(t, err, "Failed for ToWireValue(%v, %v)", spec, tt.v) {
			assert.True(
				t, wire.ValuesAreEqual(w, tt.w),
				"Unexpected value for ToWireValue(%v, %v):"+
					"\n\t   %v (got)"+
					"\n\t!= %v (expected)", spec, tt.v, w, tt.w,
			)
		}
	}
}

func TestValueFromWireError(t *testing.T) {
	tests := []struct {
		w    wire.Value
		spec compile.TypeSpec
		msg  string
		err  error
	}{
		{
			msg:  "i16 -> bool",
			w:    wire.NewValueI16(1),
			spec: &compile.BoolSpec{},
			err:  specTypeMismatch{specified: wire.TBool, got: wire.TI16},
		},
		{
			msg:  "i16 -> i8",
			w:    wire.NewValueI16(1),
			spec: &compile.I8Spec{},
			err:  specTypeMismatch{specified: wire.TI8, got: wire.TI16},
		},
		{
			msg:  "i16 -> list<i16>",
			w:    wire.NewValueI16(1),
			spec: &compile.ListSpec{ValueSpec: &compile.I16Spec{}},
			err:  specTypeMismatch{specified: wire.TList, got: wire.TI16},
		},
		{
			msg: "list<i32> -> list<i16>",
			w: makeWireList(wire.TI32, 3, func(i int) wire.Value {
				return wire.NewValueI32(0)
			}),
			spec: &compile.ListSpec{ValueSpec: &compile.I16Spec{}},
			err: specValueMismatch{"list<i16>",
				specListItemMismatch{index: 0,
					underlying: specTypeMismatch{specified: wire.TI16, got: wire.TI32},
				},
			},
		},
		{
			msg: "map<i32,i32> -> map<i16,i32>",
			w: makeWireMap(wire.TI32, wire.TI32, 3, func(i int) (key, value wire.Value) {
				return wire.NewValueI32(0), wire.NewValueI32(0)
			}),
			spec: &compile.MapSpec{
				KeySpec:   &compile.I16Spec{},
				ValueSpec: &compile.I32Spec{},
			},
			err: specValueMismatch{"map<i16, i32>",
				specMapItemMismatch{"key", specTypeMismatch{specified: wire.TI16, got: wire.TI32}},
			},
		},
		{
			msg: "map<i16,i16> -> map<i16,i32>",
			w: makeWireMap(wire.TI16, wire.TI16, 3, func(i int) (key, value wire.Value) {
				return wire.NewValueI16(0), wire.NewValueI16(0)
			}),
			spec: &compile.MapSpec{
				KeySpec:   &compile.I16Spec{},
				ValueSpec: &compile.I32Spec{},
			},
			err: specValueMismatch{"map<i16, i32>",
				specMapItemMismatch{"value", specTypeMismatch{specified: wire.TI32, got: wire.TI16}},
			},
		},
		{
			msg: "struct S {1: string s} -> struct S {1: i32 s}",
			w: wire.NewValueStruct(wire.Struct{
				Fields: []wire.Field{{
					ID:    1,
					Value: wire.NewValueI32(5),
				}},
			}),
			spec: &compile.StructSpec{
				Name: "S",
				Type: ast.StructType,
				Fields: compile.FieldGroup{{
					ID:   1,
					Name: "s",
					Type: &compile.StringSpec{},
				}},
			},
			err: specValueMismatch{"S",
				specStructFieldMismatch{"s", specTypeMismatch{specified: wire.TBinary, got: wire.TI32}},
			},
		},
	}

	for _, tt := range tests {
		got, err := FromWireValue(tt.spec, tt.w)
		if !assert.Error(t, err, "Expected error for %v", tt.msg) {
			continue
		}
		// Compare the error messages, for better failure comparisons vs the error structs.
		if assert.Equal(t, tt.err.Error(), err.Error(), "Unexpected error for %v", tt.msg) {
			assert.Nil(t, got, "Expected no result for %v", tt.msg)
		}
	}
}
