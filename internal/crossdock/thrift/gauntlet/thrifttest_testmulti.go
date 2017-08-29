// Code generated by thriftrw v1.6.0. DO NOT EDIT.
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

type ThriftTest_TestMulti_Args struct {
	Arg0 *int8            `json:"arg0,omitempty"`
	Arg1 *int32           `json:"arg1,omitempty"`
	Arg2 *int64           `json:"arg2,omitempty"`
	Arg3 map[int16]string `json:"arg3"`
	Arg4 *Numberz         `json:"arg4,omitempty"`
	Arg5 *UserId          `json:"arg5,omitempty"`
}

type _Map_I16_String_MapItemList map[int16]string

func (m _Map_I16_String_MapItemList) ForEach(f func(wire.MapItem) error) error {
	for k, v := range m {
		kw, err := wire.NewValueI16(k), error(nil)
		if err != nil {
			return err
		}
		vw, err := wire.NewValueString(v), error(nil)
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

func (m _Map_I16_String_MapItemList) Size() int {
	return len(m)
}

func (_Map_I16_String_MapItemList) KeyType() wire.Type {
	return wire.TI16
}

func (_Map_I16_String_MapItemList) ValueType() wire.Type {
	return wire.TBinary
}

func (_Map_I16_String_MapItemList) Close() {
}

func (v *ThriftTest_TestMulti_Args) ToWire() (wire.Value, error) {
	var (
		fields [6]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Arg0 != nil {
		w, err = wire.NewValueI8(*(v.Arg0)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	if v.Arg1 != nil {
		w, err = wire.NewValueI32(*(v.Arg1)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 2, Value: w}
		i++
	}
	if v.Arg2 != nil {
		w, err = wire.NewValueI64(*(v.Arg2)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 3, Value: w}
		i++
	}
	if v.Arg3 != nil {
		w, err = wire.NewValueMap(_Map_I16_String_MapItemList(v.Arg3)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 4, Value: w}
		i++
	}
	if v.Arg4 != nil {
		w, err = v.Arg4.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 5, Value: w}
		i++
	}
	if v.Arg5 != nil {
		w, err = v.Arg5.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 6, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _Map_I16_String_Read(m wire.MapItemList) (map[int16]string, error) {
	if m.KeyType() != wire.TI16 {
		return nil, nil
	}
	if m.ValueType() != wire.TBinary {
		return nil, nil
	}
	o := make(map[int16]string, m.Size())
	err := m.ForEach(func(x wire.MapItem) error {
		k, err := x.Key.GetI16(), error(nil)
		if err != nil {
			return err
		}
		v, err := x.Value.GetString(), error(nil)
		if err != nil {
			return err
		}
		o[k] = v
		return nil
	})
	m.Close()
	return o, err
}

func (v *ThriftTest_TestMulti_Args) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TI8 {
				var x int8
				x, err = field.Value.GetI8(), error(nil)
				v.Arg0 = &x
				if err != nil {
					return err
				}
			}
		case 2:
			if field.Value.Type() == wire.TI32 {
				var x int32
				x, err = field.Value.GetI32(), error(nil)
				v.Arg1 = &x
				if err != nil {
					return err
				}
			}
		case 3:
			if field.Value.Type() == wire.TI64 {
				var x int64
				x, err = field.Value.GetI64(), error(nil)
				v.Arg2 = &x
				if err != nil {
					return err
				}
			}
		case 4:
			if field.Value.Type() == wire.TMap {
				v.Arg3, err = _Map_I16_String_Read(field.Value.GetMap())
				if err != nil {
					return err
				}
			}
		case 5:
			if field.Value.Type() == wire.TI32 {
				var x Numberz
				x, err = _Numberz_Read(field.Value)
				v.Arg4 = &x
				if err != nil {
					return err
				}
			}
		case 6:
			if field.Value.Type() == wire.TI64 {
				var x UserId
				x, err = _UserId_Read(field.Value)
				v.Arg5 = &x
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *ThriftTest_TestMulti_Args) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [6]string
	i := 0
	if v.Arg0 != nil {
		fields[i] = fmt.Sprintf("Arg0: %v", *(v.Arg0))
		i++
	}
	if v.Arg1 != nil {
		fields[i] = fmt.Sprintf("Arg1: %v", *(v.Arg1))
		i++
	}
	if v.Arg2 != nil {
		fields[i] = fmt.Sprintf("Arg2: %v", *(v.Arg2))
		i++
	}
	if v.Arg3 != nil {
		fields[i] = fmt.Sprintf("Arg3: %v", v.Arg3)
		i++
	}
	if v.Arg4 != nil {
		fields[i] = fmt.Sprintf("Arg4: %v", *(v.Arg4))
		i++
	}
	if v.Arg5 != nil {
		fields[i] = fmt.Sprintf("Arg5: %v", *(v.Arg5))
		i++
	}
	return fmt.Sprintf("ThriftTest_TestMulti_Args{%v}", strings.Join(fields[:i], ", "))
}

func _Map_I16_String_Equals(lhs, rhs map[int16]string) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for lk, lv := range lhs {
		rv, ok := rhs[lk]
		if !ok {
			return false
		}
		if !(lv == rv) {
			return false
		}
	}
	return true
}

func _UserId_EqualsPtr(lhs, rhs *UserId) bool {
	if lhs != nil && rhs != nil {
		x := *lhs
		y := *rhs
		return (x == y)
	}
	return lhs == nil && rhs == nil
}

func (v *ThriftTest_TestMulti_Args) Equals(rhs *ThriftTest_TestMulti_Args) bool {
	if !_Byte_EqualsPtr(v.Arg0, rhs.Arg0) {
		return false
	}
	if !_I32_EqualsPtr(v.Arg1, rhs.Arg1) {
		return false
	}
	if !_I64_EqualsPtr(v.Arg2, rhs.Arg2) {
		return false
	}
	if !((v.Arg3 == nil && rhs.Arg3 == nil) || (v.Arg3 != nil && rhs.Arg3 != nil && _Map_I16_String_Equals(v.Arg3, rhs.Arg3))) {
		return false
	}
	if !_Numberz_EqualsPtr(v.Arg4, rhs.Arg4) {
		return false
	}
	if !_UserId_EqualsPtr(v.Arg5, rhs.Arg5) {
		return false
	}
	return true
}

func (v *ThriftTest_TestMulti_Args) GetArg0() (o int8) {
	if v.Arg0 != nil {
		return *v.Arg0
	}
	return
}

func (v *ThriftTest_TestMulti_Args) GetArg1() (o int32) {
	if v.Arg1 != nil {
		return *v.Arg1
	}
	return
}

func (v *ThriftTest_TestMulti_Args) GetArg2() (o int64) {
	if v.Arg2 != nil {
		return *v.Arg2
	}
	return
}

func (v *ThriftTest_TestMulti_Args) GetArg4() (o Numberz) {
	if v.Arg4 != nil {
		return *v.Arg4
	}
	return
}

func (v *ThriftTest_TestMulti_Args) GetArg5() (o UserId) {
	if v.Arg5 != nil {
		return *v.Arg5
	}
	return
}

func (v *ThriftTest_TestMulti_Args) MethodName() string {
	return "testMulti"
}

func (v *ThriftTest_TestMulti_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var ThriftTest_TestMulti_Helper = struct {
	Args           func(arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *Numberz, arg5 *UserId) *ThriftTest_TestMulti_Args
	IsException    func(error) bool
	WrapResponse   func(*Xtruct, error) (*ThriftTest_TestMulti_Result, error)
	UnwrapResponse func(*ThriftTest_TestMulti_Result) (*Xtruct, error)
}{}

func init() {
	ThriftTest_TestMulti_Helper.Args = func(arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *Numberz, arg5 *UserId) *ThriftTest_TestMulti_Args {
		return &ThriftTest_TestMulti_Args{Arg0: arg0, Arg1: arg1, Arg2: arg2, Arg3: arg3, Arg4: arg4, Arg5: arg5}
	}
	ThriftTest_TestMulti_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	ThriftTest_TestMulti_Helper.WrapResponse = func(success *Xtruct, err error) (*ThriftTest_TestMulti_Result, error) {
		if err == nil {
			return &ThriftTest_TestMulti_Result{Success: success}, nil
		}
		return nil, err
	}
	ThriftTest_TestMulti_Helper.UnwrapResponse = func(result *ThriftTest_TestMulti_Result) (success *Xtruct, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}

type ThriftTest_TestMulti_Result struct {
	Success *Xtruct `json:"success,omitempty"`
}

func (v *ThriftTest_TestMulti_Result) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Success != nil {
		w, err = v.Success.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}
	if i != 1 {
		return wire.Value{}, fmt.Errorf("ThriftTest_TestMulti_Result should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *ThriftTest_TestMulti_Result) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TStruct {
				v.Success, err = _Xtruct_Read(field.Value)
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
		return fmt.Errorf("ThriftTest_TestMulti_Result should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *ThriftTest_TestMulti_Result) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("ThriftTest_TestMulti_Result{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestMulti_Result) Equals(rhs *ThriftTest_TestMulti_Result) bool {
	if !((v.Success == nil && rhs.Success == nil) || (v.Success != nil && rhs.Success != nil && v.Success.Equals(rhs.Success))) {
		return false
	}
	return true
}

func (v *ThriftTest_TestMulti_Result) MethodName() string {
	return "testMulti"
}

func (v *ThriftTest_TestMulti_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
