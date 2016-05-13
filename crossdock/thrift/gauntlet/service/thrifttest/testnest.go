// Code generated by thriftrw

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
	"github.com/thriftrw/thriftrw-go/wire"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet"
	"strings"
)

type TestNestArgs struct {
	Thing *gauntlet.Xtruct2 `json:"thing,omitempty"`
}

func (v *TestNestArgs) ToWire() wire.Value {
	var fields [1]wire.Field
	i := 0
	if v.Thing != nil {
		fields[i] = wire.Field{ID: 1, Value: v.Thing.ToWire()}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]})
}

func _Xtruct2_Read(w wire.Value) (*gauntlet.Xtruct2, error) {
	var v gauntlet.Xtruct2
	err := v.FromWire(w)
	return &v, err
}

func (v *TestNestArgs) FromWire(w wire.Value) error {
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

func (v *TestNestArgs) String() string {
	var fields [1]string
	i := 0
	if v.Thing != nil {
		fields[i] = fmt.Sprintf("Thing: %v", v.Thing)
		i++
	}
	return fmt.Sprintf("TestNestArgs{%v}", strings.Join(fields[:i], ", "))
}

type TestNestResult struct {
	Success *gauntlet.Xtruct2 `json:"success,omitempty"`
}

func (v *TestNestResult) ToWire() wire.Value {
	var fields [1]wire.Field
	i := 0
	if v.Success != nil {
		fields[i] = wire.Field{ID: 0, Value: v.Success.ToWire()}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]})
}

func (v *TestNestResult) FromWire(w wire.Value) error {
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
	return nil
}

func (v *TestNestResult) String() string {
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("TestNestResult{%v}", strings.Join(fields[:i], ", "))
}

var TestNestHelper = struct {
	IsException    func(error) bool
	Args           func(thing *gauntlet.Xtruct2) *TestNestArgs
	WrapResponse   func(*gauntlet.Xtruct2, error) (*TestNestResult, error)
	UnwrapResponse func(*TestNestResult) (*gauntlet.Xtruct2, error)
}{}

func init() {
	TestNestHelper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	TestNestHelper.Args = func(thing *gauntlet.Xtruct2) *TestNestArgs {
		return &TestNestArgs{Thing: thing}
	}
	TestNestHelper.WrapResponse = func(success *gauntlet.Xtruct2, err error) (*TestNestResult, error) {
		if err == nil {
			return &TestNestResult{Success: success}, nil
		}
		return nil, err
	}
	TestNestHelper.UnwrapResponse = func(result *TestNestResult) (success *gauntlet.Xtruct2, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}
