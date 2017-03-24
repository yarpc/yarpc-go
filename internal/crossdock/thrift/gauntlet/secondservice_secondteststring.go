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

type SecondService_SecondtestString_Args struct {
	Thing *string `json:"thing,omitempty"`
}

func (v *SecondService_SecondtestString_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Thing != nil {
		w, err = wire.NewValueString(*(v.Thing)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *SecondService_SecondtestString_Args) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Thing = &x
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *SecondService_SecondtestString_Args) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Thing != nil {
		fields[i] = fmt.Sprintf("Thing: %v", *(v.Thing))
		i++
	}
	return fmt.Sprintf("SecondService_SecondtestString_Args{%v}", strings.Join(fields[:i], ", "))
}

func (v *SecondService_SecondtestString_Args) Equals(rhs *SecondService_SecondtestString_Args) bool {
	if !_String_EqualsPtr(v.Thing, rhs.Thing) {
		return false
	}
	return true
}

func (v *SecondService_SecondtestString_Args) MethodName() string {
	return "secondtestString"
}

func (v *SecondService_SecondtestString_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var SecondService_SecondtestString_Helper = struct {
	Args           func(thing *string) *SecondService_SecondtestString_Args
	IsException    func(error) bool
	WrapResponse   func(string, error) (*SecondService_SecondtestString_Result, error)
	UnwrapResponse func(*SecondService_SecondtestString_Result) (string, error)
}{}

func init() {
	SecondService_SecondtestString_Helper.Args = func(thing *string) *SecondService_SecondtestString_Args {
		return &SecondService_SecondtestString_Args{Thing: thing}
	}
	SecondService_SecondtestString_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	SecondService_SecondtestString_Helper.WrapResponse = func(success string, err error) (*SecondService_SecondtestString_Result, error) {
		if err == nil {
			return &SecondService_SecondtestString_Result{Success: &success}, nil
		}
		return nil, err
	}
	SecondService_SecondtestString_Helper.UnwrapResponse = func(result *SecondService_SecondtestString_Result) (success string, err error) {
		if result.Success != nil {
			success = *result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}

type SecondService_SecondtestString_Result struct {
	Success *string `json:"success,omitempty"`
}

func (v *SecondService_SecondtestString_Result) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Success != nil {
		w, err = wire.NewValueString(*(v.Success)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}
	if i != 1 {
		return wire.Value{}, fmt.Errorf("SecondService_SecondtestString_Result should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *SecondService_SecondtestString_Result) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Success = &x
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
		return fmt.Errorf("SecondService_SecondtestString_Result should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *SecondService_SecondtestString_Result) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", *(v.Success))
		i++
	}
	return fmt.Sprintf("SecondService_SecondtestString_Result{%v}", strings.Join(fields[:i], ", "))
}

func (v *SecondService_SecondtestString_Result) Equals(rhs *SecondService_SecondtestString_Result) bool {
	if !_String_EqualsPtr(v.Success, rhs.Success) {
		return false
	}
	return true
}

func (v *SecondService_SecondtestString_Result) MethodName() string {
	return "secondtestString"
}

func (v *SecondService_SecondtestString_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
