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

package hello

import (
	"errors"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/examples/thrift/hello/thrift/hello"
	"strings"
)

type EchoArgs struct {
	Echo *hello.EchoRequest `json:"echo,omitempty"`
}

func (v *EchoArgs) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Echo != nil {
		w, err = v.Echo.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _EchoRequest_Read(w wire.Value) (*hello.EchoRequest, error) {
	var v hello.EchoRequest
	err := v.FromWire(w)
	return &v, err
}

func (v *EchoArgs) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.Echo, err = _EchoRequest_Read(field.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *EchoArgs) String() string {
	var fields [1]string
	i := 0
	if v.Echo != nil {
		fields[i] = fmt.Sprintf("Echo: %v", v.Echo)
		i++
	}
	return fmt.Sprintf("EchoArgs{%v}", strings.Join(fields[:i], ", "))
}

func (v *EchoArgs) MethodName() string {
	return "echo"
}

func (v *EchoArgs) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

type EchoResult struct {
	Success *hello.EchoResponse `json:"success,omitempty"`
}

func (v *EchoResult) ToWire() (wire.Value, error) {
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
		return wire.Value{}, fmt.Errorf("EchoResult should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _EchoResponse_Read(w wire.Value) (*hello.EchoResponse, error) {
	var v hello.EchoResponse
	err := v.FromWire(w)
	return &v, err
}

func (v *EchoResult) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TStruct {
				v.Success, err = _EchoResponse_Read(field.Value)
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
		return fmt.Errorf("EchoResult should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *EchoResult) String() string {
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("EchoResult{%v}", strings.Join(fields[:i], ", "))
}

func (v *EchoResult) MethodName() string {
	return "echo"
}

func (v *EchoResult) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}

var EchoHelper = struct {
	IsException    func(error) bool
	Args           func(echo *hello.EchoRequest) *EchoArgs
	WrapResponse   func(*hello.EchoResponse, error) (*EchoResult, error)
	UnwrapResponse func(*EchoResult) (*hello.EchoResponse, error)
}{}

func init() {
	EchoHelper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	EchoHelper.Args = func(echo *hello.EchoRequest) *EchoArgs {
		return &EchoArgs{Echo: echo}
	}
	EchoHelper.WrapResponse = func(success *hello.EchoResponse, err error) (*EchoResult, error) {
		if err == nil {
			return &EchoResult{Success: success}, nil
		}
		return nil, err
	}
	EchoHelper.UnwrapResponse = func(result *EchoResult) (success *hello.EchoResponse, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}
