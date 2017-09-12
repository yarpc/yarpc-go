// Code generated by thriftrw v1.7.0. DO NOT EDIT.
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

package kv

import (
	"errors"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

// KeyValue_GetValue_Args represents the arguments for the KeyValue.getValue function.
//
// The arguments for getValue are sent and received over the wire as this struct.
type KeyValue_GetValue_Args struct {
	Key *string `json:"key,omitempty"`
}

// ToWire translates a KeyValue_GetValue_Args struct into a Thrift-level intermediate
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
func (v *KeyValue_GetValue_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)

	if v.Key != nil {
		w, err = wire.NewValueString(*(v.Key)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

// FromWire deserializes a KeyValue_GetValue_Args struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a KeyValue_GetValue_Args struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v KeyValue_GetValue_Args
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *KeyValue_GetValue_Args) FromWire(w wire.Value) error {
	var err error

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Key = &x
				if err != nil {
					return err
				}

			}
		}
	}

	return nil
}

// String returns a readable string representation of a KeyValue_GetValue_Args
// struct.
func (v *KeyValue_GetValue_Args) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [1]string
	i := 0
	if v.Key != nil {
		fields[i] = fmt.Sprintf("Key: %v", *(v.Key))
		i++
	}

	return fmt.Sprintf("KeyValue_GetValue_Args{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this KeyValue_GetValue_Args match the
// provided KeyValue_GetValue_Args.
//
// This function performs a deep comparison.
func (v *KeyValue_GetValue_Args) Equals(rhs *KeyValue_GetValue_Args) bool {
	if !_String_EqualsPtr(v.Key, rhs.Key) {
		return false
	}

	return true
}

// GetKey returns the value of Key if it is set or its
// zero value if it is unset.
func (v *KeyValue_GetValue_Args) GetKey() (o string) {
	if v.Key != nil {
		return *v.Key
	}

	return
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the arguments.
//
// This will always be "getValue" for this struct.
func (v *KeyValue_GetValue_Args) MethodName() string {
	return "getValue"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Call for this struct.
func (v *KeyValue_GetValue_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

// KeyValue_GetValue_Helper provides functions that aid in handling the
// parameters and return values of the KeyValue.getValue
// function.
var KeyValue_GetValue_Helper = struct {
	// Args accepts the parameters of getValue in-order and returns
	// the arguments struct for the function.
	Args func(
		key *string,
	) *KeyValue_GetValue_Args

	// IsException returns true if the given error can be thrown
	// by getValue.
	//
	// An error can be thrown by getValue only if the
	// corresponding exception type was mentioned in the 'throws'
	// section for it in the Thrift file.
	IsException func(error) bool

	// WrapResponse returns the result struct for getValue
	// given its return value and error.
	//
	// This allows mapping values and errors returned by
	// getValue into a serializable result struct.
	// WrapResponse returns a non-nil error if the provided
	// error cannot be thrown by getValue
	//
	//   value, err := getValue(args)
	//   result, err := KeyValue_GetValue_Helper.WrapResponse(value, err)
	//   if err != nil {
	//     return fmt.Errorf("unexpected error from getValue: %v", err)
	//   }
	//   serialize(result)
	WrapResponse func(string, error) (*KeyValue_GetValue_Result, error)

	// UnwrapResponse takes the result struct for getValue
	// and returns the value or error returned by it.
	//
	// The error is non-nil only if getValue threw an
	// exception.
	//
	//   result := deserialize(bytes)
	//   value, err := KeyValue_GetValue_Helper.UnwrapResponse(result)
	UnwrapResponse func(*KeyValue_GetValue_Result) (string, error)
}{}

func init() {
	KeyValue_GetValue_Helper.Args = func(
		key *string,
	) *KeyValue_GetValue_Args {
		return &KeyValue_GetValue_Args{
			Key: key,
		}
	}

	KeyValue_GetValue_Helper.IsException = func(err error) bool {
		switch err.(type) {
		case *ResourceDoesNotExist:
			return true
		default:
			return false
		}
	}

	KeyValue_GetValue_Helper.WrapResponse = func(success string, err error) (*KeyValue_GetValue_Result, error) {
		if err == nil {
			return &KeyValue_GetValue_Result{Success: &success}, nil
		}

		switch e := err.(type) {
		case *ResourceDoesNotExist:
			if e == nil {
				return nil, errors.New("WrapResponse received non-nil error type with nil value for KeyValue_GetValue_Result.DoesNotExist")
			}
			return &KeyValue_GetValue_Result{DoesNotExist: e}, nil
		}

		return nil, err
	}
	KeyValue_GetValue_Helper.UnwrapResponse = func(result *KeyValue_GetValue_Result) (success string, err error) {
		if result.DoesNotExist != nil {
			err = result.DoesNotExist
			return
		}

		if result.Success != nil {
			success = *result.Success
			return
		}

		err = errors.New("expected a non-void result")
		return
	}

}

// KeyValue_GetValue_Result represents the result of a KeyValue.getValue function call.
//
// The result of a getValue execution is sent and received over the wire as this struct.
//
// Success is set only if the function did not throw an exception.
type KeyValue_GetValue_Result struct {
	// Value returned by getValue after a successful execution.
	Success      *string               `json:"success,omitempty"`
	DoesNotExist *ResourceDoesNotExist `json:"doesNotExist,omitempty"`
}

// ToWire translates a KeyValue_GetValue_Result struct into a Thrift-level intermediate
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
func (v *KeyValue_GetValue_Result) ToWire() (wire.Value, error) {
	var (
		fields [2]wire.Field
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
	if v.DoesNotExist != nil {
		w, err = v.DoesNotExist.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}

	if i != 1 {
		return wire.Value{}, fmt.Errorf("KeyValue_GetValue_Result should have exactly one field: got %v fields", i)
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _ResourceDoesNotExist_Read(w wire.Value) (*ResourceDoesNotExist, error) {
	var v ResourceDoesNotExist
	err := v.FromWire(w)
	return &v, err
}

// FromWire deserializes a KeyValue_GetValue_Result struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a KeyValue_GetValue_Result struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v KeyValue_GetValue_Result
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *KeyValue_GetValue_Result) FromWire(w wire.Value) error {
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
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.DoesNotExist, err = _ResourceDoesNotExist_Read(field.Value)
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
	if v.DoesNotExist != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("KeyValue_GetValue_Result should have exactly one field: got %v fields", count)
	}

	return nil
}

// String returns a readable string representation of a KeyValue_GetValue_Result
// struct.
func (v *KeyValue_GetValue_Result) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [2]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", *(v.Success))
		i++
	}
	if v.DoesNotExist != nil {
		fields[i] = fmt.Sprintf("DoesNotExist: %v", v.DoesNotExist)
		i++
	}

	return fmt.Sprintf("KeyValue_GetValue_Result{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this KeyValue_GetValue_Result match the
// provided KeyValue_GetValue_Result.
//
// This function performs a deep comparison.
func (v *KeyValue_GetValue_Result) Equals(rhs *KeyValue_GetValue_Result) bool {
	if !_String_EqualsPtr(v.Success, rhs.Success) {
		return false
	}
	if !((v.DoesNotExist == nil && rhs.DoesNotExist == nil) || (v.DoesNotExist != nil && rhs.DoesNotExist != nil && v.DoesNotExist.Equals(rhs.DoesNotExist))) {
		return false
	}

	return true
}

// GetSuccess returns the value of Success if it is set or its
// zero value if it is unset.
func (v *KeyValue_GetValue_Result) GetSuccess() (o string) {
	if v.Success != nil {
		return *v.Success
	}

	return
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the result.
//
// This will always be "getValue" for this struct.
func (v *KeyValue_GetValue_Result) MethodName() string {
	return "getValue"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Reply for this struct.
func (v *KeyValue_GetValue_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
