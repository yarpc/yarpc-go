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

type ThriftTest_TestInsanity_Args struct {
	Argument *Insanity `json:"argument,omitempty"`
}

func (v *ThriftTest_TestInsanity_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Argument != nil {
		w, err = v.Argument.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *ThriftTest_TestInsanity_Args) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.Argument, err = _Insanity_Read(field.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *ThriftTest_TestInsanity_Args) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Argument != nil {
		fields[i] = fmt.Sprintf("Argument: %v", v.Argument)
		i++
	}
	return fmt.Sprintf("ThriftTest_TestInsanity_Args{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestInsanity_Args) Equals(rhs *ThriftTest_TestInsanity_Args) bool {
	if !((v.Argument == nil && rhs.Argument == nil) || (v.Argument != nil && rhs.Argument != nil && v.Argument.Equals(rhs.Argument))) {
		return false
	}
	return true
}

func (v *ThriftTest_TestInsanity_Args) MethodName() string {
	return "testInsanity"
}

func (v *ThriftTest_TestInsanity_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var ThriftTest_TestInsanity_Helper = struct {
	Args           func(argument *Insanity) *ThriftTest_TestInsanity_Args
	IsException    func(error) bool
	WrapResponse   func(map[UserId]map[Numberz]*Insanity, error) (*ThriftTest_TestInsanity_Result, error)
	UnwrapResponse func(*ThriftTest_TestInsanity_Result) (map[UserId]map[Numberz]*Insanity, error)
}{}

func init() {
	ThriftTest_TestInsanity_Helper.Args = func(argument *Insanity) *ThriftTest_TestInsanity_Args {
		return &ThriftTest_TestInsanity_Args{Argument: argument}
	}
	ThriftTest_TestInsanity_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	ThriftTest_TestInsanity_Helper.WrapResponse = func(success map[UserId]map[Numberz]*Insanity, err error) (*ThriftTest_TestInsanity_Result, error) {
		if err == nil {
			return &ThriftTest_TestInsanity_Result{Success: success}, nil
		}
		return nil, err
	}
	ThriftTest_TestInsanity_Helper.UnwrapResponse = func(result *ThriftTest_TestInsanity_Result) (success map[UserId]map[Numberz]*Insanity, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}

type ThriftTest_TestInsanity_Result struct {
	Success map[UserId]map[Numberz]*Insanity `json:"success"`
}

type _Map_Numberz_Insanity_MapItemList map[Numberz]*Insanity

func (m _Map_Numberz_Insanity_MapItemList) ForEach(f func(wire.MapItem) error) error {
	for k, v := range m {
		if v == nil {
			return fmt.Errorf("invalid [%v]: value is nil", k)
		}
		kw, err := k.ToWire()
		if err != nil {
			return err
		}
		vw, err := v.ToWire()
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

func (m _Map_Numberz_Insanity_MapItemList) Size() int {
	return len(m)
}

func (_Map_Numberz_Insanity_MapItemList) KeyType() wire.Type {
	return wire.TI32
}

func (_Map_Numberz_Insanity_MapItemList) ValueType() wire.Type {
	return wire.TStruct
}

func (_Map_Numberz_Insanity_MapItemList) Close() {
}

type _Map_UserId_Map_Numberz_Insanity_MapItemList map[UserId]map[Numberz]*Insanity

func (m _Map_UserId_Map_Numberz_Insanity_MapItemList) ForEach(f func(wire.MapItem) error) error {
	for k, v := range m {
		if v == nil {
			return fmt.Errorf("invalid [%v]: value is nil", k)
		}
		kw, err := k.ToWire()
		if err != nil {
			return err
		}
		vw, err := wire.NewValueMap(_Map_Numberz_Insanity_MapItemList(v)), error(nil)
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

func (m _Map_UserId_Map_Numberz_Insanity_MapItemList) Size() int {
	return len(m)
}

func (_Map_UserId_Map_Numberz_Insanity_MapItemList) KeyType() wire.Type {
	return wire.TI64
}

func (_Map_UserId_Map_Numberz_Insanity_MapItemList) ValueType() wire.Type {
	return wire.TMap
}

func (_Map_UserId_Map_Numberz_Insanity_MapItemList) Close() {
}

func (v *ThriftTest_TestInsanity_Result) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Success != nil {
		w, err = wire.NewValueMap(_Map_UserId_Map_Numberz_Insanity_MapItemList(v.Success)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}
	if i != 1 {
		return wire.Value{}, fmt.Errorf("ThriftTest_TestInsanity_Result should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _Map_Numberz_Insanity_Read(m wire.MapItemList) (map[Numberz]*Insanity, error) {
	if m.KeyType() != wire.TI32 {
		return nil, nil
	}
	if m.ValueType() != wire.TStruct {
		return nil, nil
	}
	o := make(map[Numberz]*Insanity, m.Size())
	err := m.ForEach(func(x wire.MapItem) error {
		k, err := _Numberz_Read(x.Key)
		if err != nil {
			return err
		}
		v, err := _Insanity_Read(x.Value)
		if err != nil {
			return err
		}
		o[k] = v
		return nil
	})
	m.Close()
	return o, err
}

func _Map_UserId_Map_Numberz_Insanity_Read(m wire.MapItemList) (map[UserId]map[Numberz]*Insanity, error) {
	if m.KeyType() != wire.TI64 {
		return nil, nil
	}
	if m.ValueType() != wire.TMap {
		return nil, nil
	}
	o := make(map[UserId]map[Numberz]*Insanity, m.Size())
	err := m.ForEach(func(x wire.MapItem) error {
		k, err := _UserId_Read(x.Key)
		if err != nil {
			return err
		}
		v, err := _Map_Numberz_Insanity_Read(x.Value.GetMap())
		if err != nil {
			return err
		}
		o[k] = v
		return nil
	})
	m.Close()
	return o, err
}

func (v *ThriftTest_TestInsanity_Result) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TMap {
				v.Success, err = _Map_UserId_Map_Numberz_Insanity_Read(field.Value.GetMap())
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
		return fmt.Errorf("ThriftTest_TestInsanity_Result should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *ThriftTest_TestInsanity_Result) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("ThriftTest_TestInsanity_Result{%v}", strings.Join(fields[:i], ", "))
}

func _Map_Numberz_Insanity_Equals(lhs, rhs map[Numberz]*Insanity) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for lk, lv := range lhs {
		rv, ok := rhs[lk]
		if !ok {
			return false
		}
		if !lv.Equals(rv) {
			return false
		}
	}
	return true
}

func _Map_UserId_Map_Numberz_Insanity_Equals(lhs, rhs map[UserId]map[Numberz]*Insanity) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for lk, lv := range lhs {
		rv, ok := rhs[lk]
		if !ok {
			return false
		}
		if !_Map_Numberz_Insanity_Equals(lv, rv) {
			return false
		}
	}
	return true
}

func (v *ThriftTest_TestInsanity_Result) Equals(rhs *ThriftTest_TestInsanity_Result) bool {
	if !((v.Success == nil && rhs.Success == nil) || (v.Success != nil && rhs.Success != nil && _Map_UserId_Map_Numberz_Insanity_Equals(v.Success, rhs.Success))) {
		return false
	}
	return true
}

func (v *ThriftTest_TestInsanity_Result) MethodName() string {
	return "testInsanity"
}

func (v *ThriftTest_TestInsanity_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
