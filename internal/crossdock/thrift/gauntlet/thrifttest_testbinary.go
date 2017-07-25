// Code generated by thriftrw v1.4.0
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
	"bytes"
	"errors"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

type ThriftTest_TestBinary_Args struct {
	Thing []byte `json:"thing"`
}

func (v *ThriftTest_TestBinary_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Thing != nil {
		w, err = wire.NewValueBinary(v.Thing), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *ThriftTest_TestBinary_Args) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				v.Thing, err = field.Value.GetBinary(), error(nil)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *ThriftTest_TestBinary_Args) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Thing != nil {
		fields[i] = fmt.Sprintf("Thing: %v", v.Thing)
		i++
	}
	return fmt.Sprintf("ThriftTest_TestBinary_Args{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestBinary_Args) Equals(rhs *ThriftTest_TestBinary_Args) bool {
	if !((v.Thing == nil && rhs.Thing == nil) || (v.Thing != nil && rhs.Thing != nil && bytes.Equal(v.Thing, rhs.Thing))) {
		return false
	}
	return true
}

func (v *ThriftTest_TestBinary_Args) MethodName() string {
	return "testBinary"
}

func (v *ThriftTest_TestBinary_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var ThriftTest_TestBinary_Helper = struct {
	Args           func(thing []byte) *ThriftTest_TestBinary_Args
	IsException    func(error) bool
	WrapResponse   func([]byte, error) (*ThriftTest_TestBinary_Result, error)
	UnwrapResponse func(*ThriftTest_TestBinary_Result) ([]byte, error)
}{}

func init() {
	ThriftTest_TestBinary_Helper.Args = func(thing []byte) *ThriftTest_TestBinary_Args {
		return &ThriftTest_TestBinary_Args{Thing: thing}
	}
	ThriftTest_TestBinary_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	ThriftTest_TestBinary_Helper.WrapResponse = func(success []byte, err error) (*ThriftTest_TestBinary_Result, error) {
		if err == nil {
			return &ThriftTest_TestBinary_Result{Success: success}, nil
		}
		return nil, err
	}
	ThriftTest_TestBinary_Helper.UnwrapResponse = func(result *ThriftTest_TestBinary_Result) (success []byte, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}

type ThriftTest_TestBinary_Result struct {
	Success []byte `json:"success"`
}

func (v *ThriftTest_TestBinary_Result) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Success != nil {
		w, err = wire.NewValueBinary(v.Success), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}
	if i != 1 {
		return wire.Value{}, fmt.Errorf("ThriftTest_TestBinary_Result should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *ThriftTest_TestBinary_Result) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TBinary {
				v.Success, err = field.Value.GetBinary(), error(nil)
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
		return fmt.Errorf("ThriftTest_TestBinary_Result should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *ThriftTest_TestBinary_Result) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("ThriftTest_TestBinary_Result{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestBinary_Result) Equals(rhs *ThriftTest_TestBinary_Result) bool {
	if !((v.Success == nil && rhs.Success == nil) || (v.Success != nil && rhs.Success != nil && bytes.Equal(v.Success, rhs.Success))) {
		return false
	}
	return true
}

func (v *ThriftTest_TestBinary_Result) MethodName() string {
	return "testBinary"
}

func (v *ThriftTest_TestBinary_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
