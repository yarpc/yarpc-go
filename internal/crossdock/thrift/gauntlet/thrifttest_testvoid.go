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
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

type ThriftTest_TestVoid_Args struct{}

func (v *ThriftTest_TestVoid_Args) ToWire() (wire.Value, error) {
	var (
		fields [0]wire.Field
		i      int = 0
	)
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *ThriftTest_TestVoid_Args) FromWire(w wire.Value) error {
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		}
	}
	return nil
}

func (v *ThriftTest_TestVoid_Args) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [0]string
	i := 0
	return fmt.Sprintf("ThriftTest_TestVoid_Args{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestVoid_Args) Equals(rhs *ThriftTest_TestVoid_Args) bool {
	return true
}

func (v *ThriftTest_TestVoid_Args) MethodName() string {
	return "testVoid"
}

func (v *ThriftTest_TestVoid_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var ThriftTest_TestVoid_Helper = struct {
	Args           func() *ThriftTest_TestVoid_Args
	IsException    func(error) bool
	WrapResponse   func(error) (*ThriftTest_TestVoid_Result, error)
	UnwrapResponse func(*ThriftTest_TestVoid_Result) error
}{}

func init() {
	ThriftTest_TestVoid_Helper.Args = func() *ThriftTest_TestVoid_Args {
		return &ThriftTest_TestVoid_Args{}
	}
	ThriftTest_TestVoid_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	ThriftTest_TestVoid_Helper.WrapResponse = func(err error) (*ThriftTest_TestVoid_Result, error) {
		if err == nil {
			return &ThriftTest_TestVoid_Result{}, nil
		}
		return nil, err
	}
	ThriftTest_TestVoid_Helper.UnwrapResponse = func(result *ThriftTest_TestVoid_Result) (err error) {
		return
	}
}

type ThriftTest_TestVoid_Result struct{}

func (v *ThriftTest_TestVoid_Result) ToWire() (wire.Value, error) {
	var (
		fields [0]wire.Field
		i      int = 0
	)
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *ThriftTest_TestVoid_Result) FromWire(w wire.Value) error {
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		}
	}
	return nil
}

func (v *ThriftTest_TestVoid_Result) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [0]string
	i := 0
	return fmt.Sprintf("ThriftTest_TestVoid_Result{%v}", strings.Join(fields[:i], ", "))
}

func (v *ThriftTest_TestVoid_Result) Equals(rhs *ThriftTest_TestVoid_Result) bool {
	return true
}

func (v *ThriftTest_TestVoid_Result) MethodName() string {
	return "testVoid"
}

func (v *ThriftTest_TestVoid_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
