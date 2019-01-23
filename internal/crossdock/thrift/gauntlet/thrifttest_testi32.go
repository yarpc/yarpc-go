// Code generated by thriftrw v1.16.0. DO NOT EDIT.
// @generated

// Copyright (c) 2019 Uber Technologies, Inc.
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
	"go.uber.org/zap/zapcore"
	"strings"
)

// ThriftTest_TestI32_Args represents the arguments for the ThriftTest.testI32 function.
//
// The arguments for testI32 are sent and received over the wire as this struct.
type ThriftTest_TestI32_Args struct {
	Thing *int32 `json:"thing,omitempty"`
}

// ToWire translates a ThriftTest_TestI32_Args struct into a Thrift-level intermediate
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
func (v *ThriftTest_TestI32_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)

	if v.Thing != nil {
		w, err = wire.NewValueI32(*(v.Thing)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

// FromWire deserializes a ThriftTest_TestI32_Args struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a ThriftTest_TestI32_Args struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v ThriftTest_TestI32_Args
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *ThriftTest_TestI32_Args) FromWire(w wire.Value) error {
	var err error

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TI32 {
				var x int32
				x, err = field.Value.GetI32(), error(nil)
				v.Thing = &x
				if err != nil {
					return err
				}

			}
		}
	}

	return nil
}

// String returns a readable string representation of a ThriftTest_TestI32_Args
// struct.
func (v *ThriftTest_TestI32_Args) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [1]string
	i := 0
	if v.Thing != nil {
		fields[i] = fmt.Sprintf("Thing: %v", *(v.Thing))
		i++
	}

	return fmt.Sprintf("ThriftTest_TestI32_Args{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this ThriftTest_TestI32_Args match the
// provided ThriftTest_TestI32_Args.
//
// This function performs a deep comparison.
func (v *ThriftTest_TestI32_Args) Equals(rhs *ThriftTest_TestI32_Args) bool {
	if v == nil {
		return rhs == nil
	} else if rhs == nil {
		return false
	}
	if !_I32_EqualsPtr(v.Thing, rhs.Thing) {
		return false
	}

	return true
}

// MarshalLogObject implements zapcore.ObjectMarshaler, enabling
// fast logging of ThriftTest_TestI32_Args.
func (v *ThriftTest_TestI32_Args) MarshalLogObject(enc zapcore.ObjectEncoder) (err error) {
	if v == nil {
		return nil
	}
	if v.Thing != nil {
		enc.AddInt32("thing", *v.Thing)
	}
	return err
}

// GetThing returns the value of Thing if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestI32_Args) GetThing() (o int32) {
	if v != nil && v.Thing != nil {
		return *v.Thing
	}

	return
}

// IsSetThing returns true if Thing is not nil.
func (v *ThriftTest_TestI32_Args) IsSetThing() bool {
	return v != nil && v.Thing != nil
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the arguments.
//
// This will always be "testI32" for this struct.
func (v *ThriftTest_TestI32_Args) MethodName() string {
	return "testI32"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Call for this struct.
func (v *ThriftTest_TestI32_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

// ThriftTest_TestI32_Helper provides functions that aid in handling the
// parameters and return values of the ThriftTest.testI32
// function.
var ThriftTest_TestI32_Helper = struct {
	// Args accepts the parameters of testI32 in-order and returns
	// the arguments struct for the function.
	Args func(
		thing *int32,
	) *ThriftTest_TestI32_Args

	// IsException returns true if the given error can be thrown
	// by testI32.
	//
	// An error can be thrown by testI32 only if the
	// corresponding exception type was mentioned in the 'throws'
	// section for it in the Thrift file.
	IsException func(error) bool

	// WrapResponse returns the result struct for testI32
	// given its return value and error.
	//
	// This allows mapping values and errors returned by
	// testI32 into a serializable result struct.
	// WrapResponse returns a non-nil error if the provided
	// error cannot be thrown by testI32
	//
	//   value, err := testI32(args)
	//   result, err := ThriftTest_TestI32_Helper.WrapResponse(value, err)
	//   if err != nil {
	//     return fmt.Errorf("unexpected error from testI32: %v", err)
	//   }
	//   serialize(result)
	WrapResponse func(int32, error) (*ThriftTest_TestI32_Result, error)

	// UnwrapResponse takes the result struct for testI32
	// and returns the value or error returned by it.
	//
	// The error is non-nil only if testI32 threw an
	// exception.
	//
	//   result := deserialize(bytes)
	//   value, err := ThriftTest_TestI32_Helper.UnwrapResponse(result)
	UnwrapResponse func(*ThriftTest_TestI32_Result) (int32, error)
}{}

func init() {
	ThriftTest_TestI32_Helper.Args = func(
		thing *int32,
	) *ThriftTest_TestI32_Args {
		return &ThriftTest_TestI32_Args{
			Thing: thing,
		}
	}

	ThriftTest_TestI32_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}

	ThriftTest_TestI32_Helper.WrapResponse = func(success int32, err error) (*ThriftTest_TestI32_Result, error) {
		if err == nil {
			return &ThriftTest_TestI32_Result{Success: &success}, nil
		}

		return nil, err
	}
	ThriftTest_TestI32_Helper.UnwrapResponse = func(result *ThriftTest_TestI32_Result) (success int32, err error) {

		if result.Success != nil {
			success = *result.Success
			return
		}

		err = errors.New("expected a non-void result")
		return
	}

}

// ThriftTest_TestI32_Result represents the result of a ThriftTest.testI32 function call.
//
// The result of a testI32 execution is sent and received over the wire as this struct.
//
// Success is set only if the function did not throw an exception.
type ThriftTest_TestI32_Result struct {
	// Value returned by testI32 after a successful execution.
	Success *int32 `json:"success,omitempty"`
}

// ToWire translates a ThriftTest_TestI32_Result struct into a Thrift-level intermediate
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
func (v *ThriftTest_TestI32_Result) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)

	if v.Success != nil {
		w, err = wire.NewValueI32(*(v.Success)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}

	if i != 1 {
		return wire.Value{}, fmt.Errorf("ThriftTest_TestI32_Result should have exactly one field: got %v fields", i)
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

// FromWire deserializes a ThriftTest_TestI32_Result struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a ThriftTest_TestI32_Result struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v ThriftTest_TestI32_Result
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *ThriftTest_TestI32_Result) FromWire(w wire.Value) error {
	var err error

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TI32 {
				var x int32
				x, err = field.Value.GetI32(), error(nil)
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
		return fmt.Errorf("ThriftTest_TestI32_Result should have exactly one field: got %v fields", count)
	}

	return nil
}

// String returns a readable string representation of a ThriftTest_TestI32_Result
// struct.
func (v *ThriftTest_TestI32_Result) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", *(v.Success))
		i++
	}

	return fmt.Sprintf("ThriftTest_TestI32_Result{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this ThriftTest_TestI32_Result match the
// provided ThriftTest_TestI32_Result.
//
// This function performs a deep comparison.
func (v *ThriftTest_TestI32_Result) Equals(rhs *ThriftTest_TestI32_Result) bool {
	if v == nil {
		return rhs == nil
	} else if rhs == nil {
		return false
	}
	if !_I32_EqualsPtr(v.Success, rhs.Success) {
		return false
	}

	return true
}

// MarshalLogObject implements zapcore.ObjectMarshaler, enabling
// fast logging of ThriftTest_TestI32_Result.
func (v *ThriftTest_TestI32_Result) MarshalLogObject(enc zapcore.ObjectEncoder) (err error) {
	if v == nil {
		return nil
	}
	if v.Success != nil {
		enc.AddInt32("success", *v.Success)
	}
	return err
}

// GetSuccess returns the value of Success if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestI32_Result) GetSuccess() (o int32) {
	if v != nil && v.Success != nil {
		return *v.Success
	}

	return
}

// IsSetSuccess returns true if Success is not nil.
func (v *ThriftTest_TestI32_Result) IsSetSuccess() bool {
	return v != nil && v.Success != nil
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the result.
//
// This will always be "testI32" for this struct.
func (v *ThriftTest_TestI32_Result) MethodName() string {
	return "testI32"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Reply for this struct.
func (v *ThriftTest_TestI32_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
