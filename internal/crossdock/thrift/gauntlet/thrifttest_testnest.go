// Code generated by thriftrw v1.1.0
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

type ThriftTest_TestNest_Args struct {
	Thing *Xtruct2 `json:"thing,omitempty"`
}

func (v *ThriftTest_TestNest_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Thing != nil {
		w, err = v.Thing.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _Xtruct2_Read(w wire.Value) (*Xtruct2, error) {
	var v Xtruct2
	err := v.FromWire(w)
	return &v, err
}

func (v *ThriftTest_TestNest_Args) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.Thing, err = _Xtruct2_Read(field.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *ThriftTest_TestNest_Args) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Thing != nil {
		fields[i] = fmt.Sprintf("Thing: %v", v.Thing)
		i++
	}
	return fmt.Sprintf("ThriftTest_TestNest_Args{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestNest_Args) Equals(rhs *ThriftTest_TestNest_Args) bool {
	if !((v.Thing == nil && rhs.Thing == nil) || (v.Thing != nil && rhs.Thing != nil && v.Thing.Equals(rhs.Thing))) {
		return false
	}
	return true
}

func (v *ThriftTest_TestNest_Args) MethodName() string {
	return "testNest"
}

func (v *ThriftTest_TestNest_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var ThriftTest_TestNest_Helper = struct {
	Args           func(thing *Xtruct2) *ThriftTest_TestNest_Args
	IsException    func(error) bool
	WrapResponse   func(*Xtruct2, error) (*ThriftTest_TestNest_Result, error)
	UnwrapResponse func(*ThriftTest_TestNest_Result) (*Xtruct2, error)
}{}

func init() {
	ThriftTest_TestNest_Helper.Args = func(thing *Xtruct2) *ThriftTest_TestNest_Args {
		return &ThriftTest_TestNest_Args{Thing: thing}
	}
	ThriftTest_TestNest_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	ThriftTest_TestNest_Helper.WrapResponse = func(success *Xtruct2, err error) (*ThriftTest_TestNest_Result, error) {
		if err == nil {
			return &ThriftTest_TestNest_Result{Success: success}, nil
		}
		return nil, err
	}
	ThriftTest_TestNest_Helper.UnwrapResponse = func(result *ThriftTest_TestNest_Result) (success *Xtruct2, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}

type ThriftTest_TestNest_Result struct {
	Success *Xtruct2 `json:"success,omitempty"`
}

func (v *ThriftTest_TestNest_Result) ToWire() (wire.Value, error) {
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
		return wire.Value{}, fmt.Errorf("ThriftTest_TestNest_Result should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *ThriftTest_TestNest_Result) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TStruct {
				v.Success, err = _Xtruct2_Read(field.Value)
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
		return fmt.Errorf("ThriftTest_TestNest_Result should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *ThriftTest_TestNest_Result) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("ThriftTest_TestNest_Result{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestNest_Result) Equals(rhs *ThriftTest_TestNest_Result) bool {
	if !((v.Success == nil && rhs.Success == nil) || (v.Success != nil && rhs.Success != nil && v.Success.Equals(rhs.Success))) {
		return false
	}
	return true
}

func (v *ThriftTest_TestNest_Result) MethodName() string {
	return "testNest"
}

func (v *ThriftTest_TestNest_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
