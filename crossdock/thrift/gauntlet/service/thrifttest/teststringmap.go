// Code generated by thriftrw
// @generated

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

package thrifttest

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/thriftrw/wire"
)

type TestStringMapArgs struct {
	Thing map[string]string `json:"thing"`
}

type _Map_String_String_MapItemList map[string]string

func (m _Map_String_String_MapItemList) ForEach(f func(wire.MapItem) error) error {
	for k, v := range m {
		kw, err := wire.NewValueString(k), error(nil)
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

func (m _Map_String_String_MapItemList) Size() int {
	return len(m)
}

func (_Map_String_String_MapItemList) KeyType() wire.Type {
	return wire.TBinary
}

func (_Map_String_String_MapItemList) ValueType() wire.Type {
	return wire.TBinary
}

func (_Map_String_String_MapItemList) Close() {
}

func (v *TestStringMapArgs) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Thing != nil {
		w, err = wire.NewValueMap(_Map_String_String_MapItemList(v.Thing)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _Map_String_String_Read(m wire.MapItemList) (map[string]string, error) {
	if m.KeyType() != wire.TBinary {
		return nil, nil
	}
	if m.ValueType() != wire.TBinary {
		return nil, nil
	}
	o := make(map[string]string, m.Size())
	err := m.ForEach(func(x wire.MapItem) error {
		k, err := x.Key.GetString(), error(nil)
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

func (v *TestStringMapArgs) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TMap {
				v.Thing, err = _Map_String_String_Read(field.Value.GetMap())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *TestStringMapArgs) String() string {
	var fields [1]string
	i := 0
	if v.Thing != nil {
		fields[i] = fmt.Sprintf("Thing: %v", v.Thing)
		i++
	}
	return fmt.Sprintf("TestStringMapArgs{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestStringMapArgs) MethodName() string {
	return "testStringMap"
}

func (v *TestStringMapArgs) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

type TestStringMapResult struct {
	Success map[string]string `json:"success"`
}

func (v *TestStringMapResult) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Success != nil {
		w, err = wire.NewValueMap(_Map_String_String_MapItemList(v.Success)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}
	if i != 1 {
		return wire.Value{}, fmt.Errorf("TestStringMapResult should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *TestStringMapResult) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TMap {
				v.Success, err = _Map_String_String_Read(field.Value.GetMap())
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
		return fmt.Errorf("TestStringMapResult should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *TestStringMapResult) String() string {
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("TestStringMapResult{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestStringMapResult) MethodName() string {
	return "testStringMap"
}

func (v *TestStringMapResult) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}

var TestStringMapHelper = struct {
	IsException    func(error) bool
	Args           func(thing map[string]string) *TestStringMapArgs
	WrapResponse   func(map[string]string, error) (*TestStringMapResult, error)
	UnwrapResponse func(*TestStringMapResult) (map[string]string, error)
}{}

func init() {
	TestStringMapHelper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	TestStringMapHelper.Args = func(thing map[string]string) *TestStringMapArgs {
		return &TestStringMapArgs{Thing: thing}
	}
	TestStringMapHelper.WrapResponse = func(success map[string]string, err error) (*TestStringMapResult, error) {
		if err == nil {
			return &TestStringMapResult{Success: success}, nil
		}
		return nil, err
	}
	TestStringMapHelper.UnwrapResponse = func(result *TestStringMapResult) (success map[string]string, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}
