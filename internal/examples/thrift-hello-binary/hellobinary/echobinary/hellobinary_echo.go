// Code generated by thriftrw v1.9.0. DO NOT EDIT.
// @generated

// Copyright (c) 2018 Uber Technologies, Inc.
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

package echobinary

import (
	"errors"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

// HelloBinary_Echo_Args represents the arguments for the HelloBinary.echo function.
//
// The arguments for echo are sent and received over the wire as this struct.
type HelloBinary_Echo_Args struct {
	Echo *EchoBinaryRequest `json:"echo,omitempty"`
}

// ToWire translates a HelloBinary_Echo_Args struct into a Thrift-level intermediate
// representation. This intermediate representation may be serialized
// into bytes using a ThriftRW protocol implementation.
//
// An error is returned if the struct or any of its fields failed to
// validate.
//
//   x, err := v.ToWire()
//   if err != nil {
//     return err
//   }
//
//   if err := binaryProtocol.Encode(x, writer); err != nil {
//     return err
//   }
func (v *HelloBinary_Echo_Args) ToWire() (wire.Value, error) {
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

func _EchoBinaryRequest_Read(w wire.Value) (*EchoBinaryRequest, error) {
	var v EchoBinaryRequest
	err := v.FromWire(w)
	return &v, err
}

// FromWire deserializes a HelloBinary_Echo_Args struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a HelloBinary_Echo_Args struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v HelloBinary_Echo_Args
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *HelloBinary_Echo_Args) FromWire(w wire.Value) error {
	var err error

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.Echo, err = _EchoBinaryRequest_Read(field.Value)
				if err != nil {
					return err
				}

			}
		}
	}

	return nil
}

// String returns a readable string representation of a HelloBinary_Echo_Args
// struct.
func (v *HelloBinary_Echo_Args) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [1]string
	i := 0
	if v.Echo != nil {
		fields[i] = fmt.Sprintf("Echo: %v", v.Echo)
		i++
	}

	return fmt.Sprintf("HelloBinary_Echo_Args{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this HelloBinary_Echo_Args match the
// provided HelloBinary_Echo_Args.
//
// This function performs a deep comparison.
func (v *HelloBinary_Echo_Args) Equals(rhs *HelloBinary_Echo_Args) bool {
	if !((v.Echo == nil && rhs.Echo == nil) || (v.Echo != nil && rhs.Echo != nil && v.Echo.Equals(rhs.Echo))) {
		return false
	}

	return true
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the arguments.
//
// This will always be "echo" for this struct.
func (v *HelloBinary_Echo_Args) MethodName() string {
	return "echo"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Call for this struct.
func (v *HelloBinary_Echo_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

// HelloBinary_Echo_Helper provides functions that aid in handling the
// parameters and return values of the HelloBinary.echo
// function.
var HelloBinary_Echo_Helper = struct {
	// Args accepts the parameters of echo in-order and returns
	// the arguments struct for the function.
	Args func(
		echo *EchoBinaryRequest,
	) *HelloBinary_Echo_Args

	// IsException returns true if the given error can be thrown
	// by echo.
	//
	// An error can be thrown by echo only if the
	// corresponding exception type was mentioned in the 'throws'
	// section for it in the Thrift file.
	IsException func(error) bool

	// WrapResponse returns the result struct for echo
	// given its return value and error.
	//
	// This allows mapping values and errors returned by
	// echo into a serializable result struct.
	// WrapResponse returns a non-nil error if the provided
	// error cannot be thrown by echo
	//
	//   value, err := echo(args)
	//   result, err := HelloBinary_Echo_Helper.WrapResponse(value, err)
	//   if err != nil {
	//     return fmt.Errorf("unexpected error from echo: %v", err)
	//   }
	//   serialize(result)
	WrapResponse func(*EchoBinaryResponse, error) (*HelloBinary_Echo_Result, error)

	// UnwrapResponse takes the result struct for echo
	// and returns the value or error returned by it.
	//
	// The error is non-nil only if echo threw an
	// exception.
	//
	//   result := deserialize(bytes)
	//   value, err := HelloBinary_Echo_Helper.UnwrapResponse(result)
	UnwrapResponse func(*HelloBinary_Echo_Result) (*EchoBinaryResponse, error)
}{}

func init() {
	HelloBinary_Echo_Helper.Args = func(
		echo *EchoBinaryRequest,
	) *HelloBinary_Echo_Args {
		return &HelloBinary_Echo_Args{
			Echo: echo,
		}
	}

	HelloBinary_Echo_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}

	HelloBinary_Echo_Helper.WrapResponse = func(success *EchoBinaryResponse, err error) (*HelloBinary_Echo_Result, error) {
		if err == nil {
			return &HelloBinary_Echo_Result{Success: success}, nil
		}

		return nil, err
	}
	HelloBinary_Echo_Helper.UnwrapResponse = func(result *HelloBinary_Echo_Result) (success *EchoBinaryResponse, err error) {

		if result.Success != nil {
			success = result.Success
			return
		}

		err = errors.New("expected a non-void result")
		return
	}

}

// HelloBinary_Echo_Result represents the result of a HelloBinary.echo function call.
//
// The result of a echo execution is sent and received over the wire as this struct.
//
// Success is set only if the function did not throw an exception.
type HelloBinary_Echo_Result struct {
	// Value returned by echo after a successful execution.
	Success *EchoBinaryResponse `json:"success,omitempty"`
}

// ToWire translates a HelloBinary_Echo_Result struct into a Thrift-level intermediate
// representation. This intermediate representation may be serialized
// into bytes using a ThriftRW protocol implementation.
//
// An error is returned if the struct or any of its fields failed to
// validate.
//
//   x, err := v.ToWire()
//   if err != nil {
//     return err
//   }
//
//   if err := binaryProtocol.Encode(x, writer); err != nil {
//     return err
//   }
func (v *HelloBinary_Echo_Result) ToWire() (wire.Value, error) {
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
		return wire.Value{}, fmt.Errorf("HelloBinary_Echo_Result should have exactly one field: got %v fields", i)
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _EchoBinaryResponse_Read(w wire.Value) (*EchoBinaryResponse, error) {
	var v EchoBinaryResponse
	err := v.FromWire(w)
	return &v, err
}

// FromWire deserializes a HelloBinary_Echo_Result struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a HelloBinary_Echo_Result struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v HelloBinary_Echo_Result
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *HelloBinary_Echo_Result) FromWire(w wire.Value) error {
	var err error

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TStruct {
				v.Success, err = _EchoBinaryResponse_Read(field.Value)
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
		return fmt.Errorf("HelloBinary_Echo_Result should have exactly one field: got %v fields", count)
	}

	return nil
}

// String returns a readable string representation of a HelloBinary_Echo_Result
// struct.
func (v *HelloBinary_Echo_Result) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}

	return fmt.Sprintf("HelloBinary_Echo_Result{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this HelloBinary_Echo_Result match the
// provided HelloBinary_Echo_Result.
//
// This function performs a deep comparison.
func (v *HelloBinary_Echo_Result) Equals(rhs *HelloBinary_Echo_Result) bool {
	if !((v.Success == nil && rhs.Success == nil) || (v.Success != nil && rhs.Success != nil && v.Success.Equals(rhs.Success))) {
		return false
	}

	return true
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the result.
//
// This will always be "echo" for this struct.
func (v *HelloBinary_Echo_Result) MethodName() string {
	return "echo"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Reply for this struct.
func (v *HelloBinary_Echo_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
