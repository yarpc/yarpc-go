// Code generated by thriftrw v1.3.0
// @generated

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

package gauntlet

import (
	"errors"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

type ThriftTest_TestMapMap_Args struct {
	Hello *int32 `json:"hello,omitempty"`
}

func (v *ThriftTest_TestMapMap_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Hello != nil {
		w, err = wire.NewValueI32(*(v.Hello)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *ThriftTest_TestMapMap_Args) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TI32 {
				var x int32
				x, err = field.Value.GetI32(), error(nil)
				v.Hello = &x
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *ThriftTest_TestMapMap_Args) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Hello != nil {
		fields[i] = fmt.Sprintf("Hello: %v", *(v.Hello))
		i++
	}
	return fmt.Sprintf("ThriftTest_TestMapMap_Args{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestMapMap_Args) Equals(rhs *ThriftTest_TestMapMap_Args) bool {
	if !_I32_EqualsPtr(v.Hello, rhs.Hello) {
		return false
	}
	return true
}

func (v *ThriftTest_TestMapMap_Args) MethodName() string {
	return "testMapMap"
}

func (v *ThriftTest_TestMapMap_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var ThriftTest_TestMapMap_Helper = struct {
	Args           func(hello *int32) *ThriftTest_TestMapMap_Args
	IsException    func(error) bool
	WrapResponse   func(map[int32]map[int32]int32, error) (*ThriftTest_TestMapMap_Result, error)
	UnwrapResponse func(*ThriftTest_TestMapMap_Result) (map[int32]map[int32]int32, error)
}{}

func init() {
	ThriftTest_TestMapMap_Helper.Args = func(hello *int32) *ThriftTest_TestMapMap_Args {
		return &ThriftTest_TestMapMap_Args{Hello: hello}
	}
	ThriftTest_TestMapMap_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	ThriftTest_TestMapMap_Helper.WrapResponse = func(success map[int32]map[int32]int32, err error) (*ThriftTest_TestMapMap_Result, error) {
		if err == nil {
			return &ThriftTest_TestMapMap_Result{Success: success}, nil
		}
		return nil, err
	}
	ThriftTest_TestMapMap_Helper.UnwrapResponse = func(result *ThriftTest_TestMapMap_Result) (success map[int32]map[int32]int32, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}

type ThriftTest_TestMapMap_Result struct {
	Success map[int32]map[int32]int32 `json:"success"`
}

type _Map_I32_Map_I32_I32_MapItemList map[int32]map[int32]int32

func (m _Map_I32_Map_I32_I32_MapItemList) ForEach(f func(wire.MapItem) error) error {
	for k, v := range m {
		if v == nil {
			return fmt.Errorf("invalid [%v]: value is nil", k)
		}
		kw, err := wire.NewValueI32(k), error(nil)
		if err != nil {
			return err
		}
		vw, err := wire.NewValueMap(_Map_I32_I32_MapItemList(v)), error(nil)
		if err != nil {
			return err
		}
		err = f(wire.MapItem{Key: kw, Value: vw})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m _Map_I32_Map_I32_I32_MapItemList) Size() int {
	return len(m)
}

func (_Map_I32_Map_I32_I32_MapItemList) KeyType() wire.Type {
	return wire.TI32
}

func (_Map_I32_Map_I32_I32_MapItemList) ValueType() wire.Type {
	return wire.TMap
}

func (_Map_I32_Map_I32_I32_MapItemList) Close() {
}

func (v *ThriftTest_TestMapMap_Result) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Success != nil {
		w, err = wire.NewValueMap(_Map_I32_Map_I32_I32_MapItemList(v.Success)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}
	if i != 1 {
		return wire.Value{}, fmt.Errorf("ThriftTest_TestMapMap_Result should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _Map_I32_Map_I32_I32_Read(m wire.MapItemList) (map[int32]map[int32]int32, error) {
	if m.KeyType() != wire.TI32 {
		return nil, nil
	}
	if m.ValueType() != wire.TMap {
		return nil, nil
	}
	o := make(map[int32]map[int32]int32, m.Size())
	err := m.ForEach(func(x wire.MapItem) error {
		k, err := x.Key.GetI32(), error(nil)
		if err != nil {
			return err
		}
		v, err := _Map_I32_I32_Read(x.Value.GetMap())
		if err != nil {
			return err
		}
		o[k] = v
		return nil
	})
	m.Close()
	return o, err
}

func (v *ThriftTest_TestMapMap_Result) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TMap {
				v.Success, err = _Map_I32_Map_I32_I32_Read(field.Value.GetMap())
				if err != nil {
					return err
				}
			}
		}
	}
	count := 0
	if v.Success != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("ThriftTest_TestMapMap_Result should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *ThriftTest_TestMapMap_Result) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("ThriftTest_TestMapMap_Result{%v}", strings.Join(fields[:i], ", "))
}

func _Map_I32_Map_I32_I32_Equals(lhs, rhs map[int32]map[int32]int32) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for lk, lv := range lhs {
		rv, ok := rhs[lk]
		if !ok {
			return false
		}
		if !_Map_I32_I32_Equals(lv, rv) {
			return false
		}
	}
	return true
}

func (v *ThriftTest_TestMapMap_Result) Equals(rhs *ThriftTest_TestMapMap_Result) bool {
	if !((v.Success == nil && rhs.Success == nil) || (v.Success != nil && rhs.Success != nil && _Map_I32_Map_I32_I32_Equals(v.Success, rhs.Success))) {
		return false
	}
	return true
}

func (v *ThriftTest_TestMapMap_Result) MethodName() string {
	return "testMapMap"
}

func (v *ThriftTest_TestMapMap_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
