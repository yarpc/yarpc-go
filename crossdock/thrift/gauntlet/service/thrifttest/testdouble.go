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
	"go.uber.org/thriftrw/wire"
	"strings"
)

type TestDoubleArgs struct {
	Thing *float64 `json:"thing,omitempty"`
}

func (v *TestDoubleArgs) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Thing != nil {
		w, err = wire.NewValueDouble(*(v.Thing)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *TestDoubleArgs) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TDouble {
				var x float64
				x, err = field.Value.GetDouble(), error(nil)
				v.Thing = &x
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *TestDoubleArgs) String() string {
	var fields [1]string
	i := 0
	if v.Thing != nil {
		fields[i] = fmt.Sprintf("Thing: %v", *(v.Thing))
		i++
	}
	return fmt.Sprintf("TestDoubleArgs{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestDoubleArgs) MethodName() string {
	return "testDouble"
}

func (v *TestDoubleArgs) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

type TestDoubleResult struct {
	Success *float64 `json:"success,omitempty"`
}

func (v *TestDoubleResult) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Success != nil {
		w, err = wire.NewValueDouble(*(v.Success)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}
	if i != 1 {
		return wire.Value{}, fmt.Errorf("TestDoubleResult should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *TestDoubleResult) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TDouble {
				var x float64
				x, err = field.Value.GetDouble(), error(nil)
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
		return fmt.Errorf("TestDoubleResult should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *TestDoubleResult) String() string {
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", *(v.Success))
		i++
	}
	return fmt.Sprintf("TestDoubleResult{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestDoubleResult) MethodName() string {
	return "testDouble"
}

func (v *TestDoubleResult) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}

var TestDoubleHelper = struct {
	IsException    func(error) bool
	Args           func(thing *float64) *TestDoubleArgs
	WrapResponse   func(float64, error) (*TestDoubleResult, error)
	UnwrapResponse func(*TestDoubleResult) (float64, error)
}{}

func init() {
	TestDoubleHelper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	TestDoubleHelper.Args = func(thing *float64) *TestDoubleArgs {
		return &TestDoubleArgs{Thing: thing}
	}
	TestDoubleHelper.WrapResponse = func(success float64, err error) (*TestDoubleResult, error) {
		if err == nil {
			return &TestDoubleResult{Success: &success}, nil
		}
		return nil, err
	}
	TestDoubleHelper.UnwrapResponse = func(result *TestDoubleResult) (success float64, err error) {
		if result.Success != nil {
			success = *result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}
