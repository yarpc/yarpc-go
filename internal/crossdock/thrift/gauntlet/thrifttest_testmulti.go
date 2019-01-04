// Code generated by thriftrw v1.14.0. DO NOT EDIT.
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
	"go.uber.org/multierr"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/zap/zapcore"
	"strings"
)

// ThriftTest_TestMulti_Args represents the arguments for the ThriftTest.testMulti function.
//
// The arguments for testMulti are sent and received over the wire as this struct.
type ThriftTest_TestMulti_Args struct {
	Arg0 *int8            `json:"arg0,omitempty"`
	Arg1 *int32           `json:"arg1,omitempty"`
	Arg2 *int64           `json:"arg2,omitempty"`
	Arg3 map[int16]string `json:"arg3,omitempty"`
	Arg4 *Numberz         `json:"arg4,omitempty"`
	Arg5 *UserId          `json:"arg5,omitempty"`
}

type _Map_I16_String_MapItemList map[int16]string

func (m _Map_I16_String_MapItemList) ForEach(f func(wire.MapItem) error) error {
	for k, v := range m {
		kw, err := wire.NewValueI16(k), error(nil)
		if err != nil {
			return err
		}

		vw, err := wire.NewValueString(v), error(nil)
		if err != nil {
			return err
		}
		err = f(wire.MapItem{Key: kw, Value: vw})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m _Map_I16_String_MapItemList) Size() int {
	return len(m)
}

func (_Map_I16_String_MapItemList) KeyType() wire.Type {
	return wire.TI16
}

func (_Map_I16_String_MapItemList) ValueType() wire.Type {
	return wire.TBinary
}

func (_Map_I16_String_MapItemList) Close() {}

// ToWire translates a ThriftTest_TestMulti_Args struct into a Thrift-level intermediate
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
func (v *ThriftTest_TestMulti_Args) ToWire() (wire.Value, error) {
	var (
		fields [6]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)

	if v.Arg0 != nil {
		w, err = wire.NewValueI8(*(v.Arg0)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	if v.Arg1 != nil {
		w, err = wire.NewValueI32(*(v.Arg1)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 2, Value: w}
		i++
	}
	if v.Arg2 != nil {
		w, err = wire.NewValueI64(*(v.Arg2)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 3, Value: w}
		i++
	}
	if v.Arg3 != nil {
		w, err = wire.NewValueMap(_Map_I16_String_MapItemList(v.Arg3)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 4, Value: w}
		i++
	}
	if v.Arg4 != nil {
		w, err = v.Arg4.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 5, Value: w}
		i++
	}
	if v.Arg5 != nil {
		w, err = v.Arg5.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 6, Value: w}
		i++
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _Map_I16_String_Read(m wire.MapItemList) (map[int16]string, error) {
	if m.KeyType() != wire.TI16 {
		return nil, nil
	}

	if m.ValueType() != wire.TBinary {
		return nil, nil
	}

	o := make(map[int16]string, m.Size())
	err := m.ForEach(func(x wire.MapItem) error {
		k, err := x.Key.GetI16(), error(nil)
		if err != nil {
			return err
		}

		v, err := x.Value.GetString(), error(nil)
		if err != nil {
			return err
		}

		o[k] = v
		return nil
	})
	m.Close()
	return o, err
}

// FromWire deserializes a ThriftTest_TestMulti_Args struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a ThriftTest_TestMulti_Args struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v ThriftTest_TestMulti_Args
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *ThriftTest_TestMulti_Args) FromWire(w wire.Value) error {
	var err error

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TI8 {
				var x int8
				x, err = field.Value.GetI8(), error(nil)
				v.Arg0 = &x
				if err != nil {
					return err
				}

			}
		case 2:
			if field.Value.Type() == wire.TI32 {
				var x int32
				x, err = field.Value.GetI32(), error(nil)
				v.Arg1 = &x
				if err != nil {
					return err
				}

			}
		case 3:
			if field.Value.Type() == wire.TI64 {
				var x int64
				x, err = field.Value.GetI64(), error(nil)
				v.Arg2 = &x
				if err != nil {
					return err
				}

			}
		case 4:
			if field.Value.Type() == wire.TMap {
				v.Arg3, err = _Map_I16_String_Read(field.Value.GetMap())
				if err != nil {
					return err
				}

			}
		case 5:
			if field.Value.Type() == wire.TI32 {
				var x Numberz
				x, err = _Numberz_Read(field.Value)
				v.Arg4 = &x
				if err != nil {
					return err
				}

			}
		case 6:
			if field.Value.Type() == wire.TI64 {
				var x UserId
				x, err = _UserId_Read(field.Value)
				v.Arg5 = &x
				if err != nil {
					return err
				}

			}
		}
	}

	return nil
}

// String returns a readable string representation of a ThriftTest_TestMulti_Args
// struct.
func (v *ThriftTest_TestMulti_Args) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [6]string
	i := 0
	if v.Arg0 != nil {
		fields[i] = fmt.Sprintf("Arg0: %v", *(v.Arg0))
		i++
	}
	if v.Arg1 != nil {
		fields[i] = fmt.Sprintf("Arg1: %v", *(v.Arg1))
		i++
	}
	if v.Arg2 != nil {
		fields[i] = fmt.Sprintf("Arg2: %v", *(v.Arg2))
		i++
	}
	if v.Arg3 != nil {
		fields[i] = fmt.Sprintf("Arg3: %v", v.Arg3)
		i++
	}
	if v.Arg4 != nil {
		fields[i] = fmt.Sprintf("Arg4: %v", *(v.Arg4))
		i++
	}
	if v.Arg5 != nil {
		fields[i] = fmt.Sprintf("Arg5: %v", *(v.Arg5))
		i++
	}

	return fmt.Sprintf("ThriftTest_TestMulti_Args{%v}", strings.Join(fields[:i], ", "))
}

func _Map_I16_String_Equals(lhs, rhs map[int16]string) bool {
	if len(lhs) != len(rhs) {
		return false
	}

	for lk, lv := range lhs {
		rv, ok := rhs[lk]
		if !ok {
			return false
		}
		if !(lv == rv) {
			return false
		}
	}
	return true
}

func _UserId_EqualsPtr(lhs, rhs *UserId) bool {
	if lhs != nil && rhs != nil {

		x := *lhs
		y := *rhs
		return (x == y)
	}
	return lhs == nil && rhs == nil
}

// Equals returns true if all the fields of this ThriftTest_TestMulti_Args match the
// provided ThriftTest_TestMulti_Args.
//
// This function performs a deep comparison.
func (v *ThriftTest_TestMulti_Args) Equals(rhs *ThriftTest_TestMulti_Args) bool {
	if v == nil {
		return rhs == nil
	} else if rhs == nil {
		return false
	}
	if !_Byte_EqualsPtr(v.Arg0, rhs.Arg0) {
		return false
	}
	if !_I32_EqualsPtr(v.Arg1, rhs.Arg1) {
		return false
	}
	if !_I64_EqualsPtr(v.Arg2, rhs.Arg2) {
		return false
	}
	if !((v.Arg3 == nil && rhs.Arg3 == nil) || (v.Arg3 != nil && rhs.Arg3 != nil && _Map_I16_String_Equals(v.Arg3, rhs.Arg3))) {
		return false
	}
	if !_Numberz_EqualsPtr(v.Arg4, rhs.Arg4) {
		return false
	}
	if !_UserId_EqualsPtr(v.Arg5, rhs.Arg5) {
		return false
	}

	return true
}

type _Map_I16_String_Item_Zapper struct {
	Key   int16
	Value string
}

// MarshalLogArray implements zapcore.ArrayMarshaler, enabling
// fast logging of _Map_I16_String_Item_Zapper.
func (v _Map_I16_String_Item_Zapper) MarshalLogObject(enc zapcore.ObjectEncoder) (err error) {
	enc.AddInt16("key", v.Key)
	enc.AddString("value", v.Value)
	return err
}

type _Map_I16_String_Zapper map[int16]string

// MarshalLogArray implements zapcore.ArrayMarshaler, enabling
// fast logging of _Map_I16_String_Zapper.
func (m _Map_I16_String_Zapper) MarshalLogArray(enc zapcore.ArrayEncoder) (err error) {
	for k, v := range m {
		err = multierr.Append(err, enc.AppendObject(_Map_I16_String_Item_Zapper{Key: k, Value: v}))
	}
	return err
}

// MarshalLogObject implements zapcore.ObjectMarshaler, enabling
// fast logging of ThriftTest_TestMulti_Args.
func (v *ThriftTest_TestMulti_Args) MarshalLogObject(enc zapcore.ObjectEncoder) (err error) {
	if v == nil {
		return nil
	}
	if v.Arg0 != nil {
		enc.AddInt8("arg0", *v.Arg0)
	}
	if v.Arg1 != nil {
		enc.AddInt32("arg1", *v.Arg1)
	}
	if v.Arg2 != nil {
		enc.AddInt64("arg2", *v.Arg2)
	}
	if v.Arg3 != nil {
		err = multierr.Append(err, enc.AddArray("arg3", (_Map_I16_String_Zapper)(v.Arg3)))
	}
	if v.Arg4 != nil {
		err = multierr.Append(err, enc.AddObject("arg4", *v.Arg4))
	}
	if v.Arg5 != nil {
		enc.AddInt64("arg5", (int64)(*v.Arg5))
	}
	return err
}

// GetArg0 returns the value of Arg0 if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestMulti_Args) GetArg0() (o int8) {
	if v.Arg0 != nil {
		return *v.Arg0
	}

	return
}

// IsSetArg0 returns true if Arg0 is not nil.
func (v *ThriftTest_TestMulti_Args) IsSetArg0() bool {
	return v.Arg0 != nil
}

// GetArg1 returns the value of Arg1 if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestMulti_Args) GetArg1() (o int32) {
	if v.Arg1 != nil {
		return *v.Arg1
	}

	return
}

// IsSetArg1 returns true if Arg1 is not nil.
func (v *ThriftTest_TestMulti_Args) IsSetArg1() bool {
	return v.Arg1 != nil
}

// GetArg2 returns the value of Arg2 if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestMulti_Args) GetArg2() (o int64) {
	if v.Arg2 != nil {
		return *v.Arg2
	}

	return
}

// IsSetArg2 returns true if Arg2 is not nil.
func (v *ThriftTest_TestMulti_Args) IsSetArg2() bool {
	return v.Arg2 != nil
}

// GetArg3 returns the value of Arg3 if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestMulti_Args) GetArg3() (o map[int16]string) {
	if v.Arg3 != nil {
		return v.Arg3
	}

	return
}

// IsSetArg3 returns true if Arg3 is not nil.
func (v *ThriftTest_TestMulti_Args) IsSetArg3() bool {
	return v.Arg3 != nil
}

// GetArg4 returns the value of Arg4 if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestMulti_Args) GetArg4() (o Numberz) {
	if v.Arg4 != nil {
		return *v.Arg4
	}

	return
}

// IsSetArg4 returns true if Arg4 is not nil.
func (v *ThriftTest_TestMulti_Args) IsSetArg4() bool {
	return v.Arg4 != nil
}

// GetArg5 returns the value of Arg5 if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestMulti_Args) GetArg5() (o UserId) {
	if v.Arg5 != nil {
		return *v.Arg5
	}

	return
}

// IsSetArg5 returns true if Arg5 is not nil.
func (v *ThriftTest_TestMulti_Args) IsSetArg5() bool {
	return v.Arg5 != nil
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the arguments.
//
// This will always be "testMulti" for this struct.
func (v *ThriftTest_TestMulti_Args) MethodName() string {
	return "testMulti"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Call for this struct.
func (v *ThriftTest_TestMulti_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

// ThriftTest_TestMulti_Helper provides functions that aid in handling the
// parameters and return values of the ThriftTest.testMulti
// function.
var ThriftTest_TestMulti_Helper = struct {
	// Args accepts the parameters of testMulti in-order and returns
	// the arguments struct for the function.
	Args func(
		arg0 *int8,
		arg1 *int32,
		arg2 *int64,
		arg3 map[int16]string,
		arg4 *Numberz,
		arg5 *UserId,
	) *ThriftTest_TestMulti_Args

	// IsException returns true if the given error can be thrown
	// by testMulti.
	//
	// An error can be thrown by testMulti only if the
	// corresponding exception type was mentioned in the 'throws'
	// section for it in the Thrift file.
	IsException func(error) bool

	// WrapResponse returns the result struct for testMulti
	// given its return value and error.
	//
	// This allows mapping values and errors returned by
	// testMulti into a serializable result struct.
	// WrapResponse returns a non-nil error if the provided
	// error cannot be thrown by testMulti
	//
	//   value, err := testMulti(args)
	//   result, err := ThriftTest_TestMulti_Helper.WrapResponse(value, err)
	//   if err != nil {
	//     return fmt.Errorf("unexpected error from testMulti: %v", err)
	//   }
	//   serialize(result)
	WrapResponse func(*Xtruct, error) (*ThriftTest_TestMulti_Result, error)

	// UnwrapResponse takes the result struct for testMulti
	// and returns the value or error returned by it.
	//
	// The error is non-nil only if testMulti threw an
	// exception.
	//
	//   result := deserialize(bytes)
	//   value, err := ThriftTest_TestMulti_Helper.UnwrapResponse(result)
	UnwrapResponse func(*ThriftTest_TestMulti_Result) (*Xtruct, error)
}{}

func init() {
	ThriftTest_TestMulti_Helper.Args = func(
		arg0 *int8,
		arg1 *int32,
		arg2 *int64,
		arg3 map[int16]string,
		arg4 *Numberz,
		arg5 *UserId,
	) *ThriftTest_TestMulti_Args {
		return &ThriftTest_TestMulti_Args{
			Arg0: arg0,
			Arg1: arg1,
			Arg2: arg2,
			Arg3: arg3,
			Arg4: arg4,
			Arg5: arg5,
		}
	}

	ThriftTest_TestMulti_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}

	ThriftTest_TestMulti_Helper.WrapResponse = func(success *Xtruct, err error) (*ThriftTest_TestMulti_Result, error) {
		if err == nil {
			return &ThriftTest_TestMulti_Result{Success: success}, nil
		}

		return nil, err
	}
	ThriftTest_TestMulti_Helper.UnwrapResponse = func(result *ThriftTest_TestMulti_Result) (success *Xtruct, err error) {

		if result.Success != nil {
			success = result.Success
			return
		}

		err = errors.New("expected a non-void result")
		return
	}

}

// ThriftTest_TestMulti_Result represents the result of a ThriftTest.testMulti function call.
//
// The result of a testMulti execution is sent and received over the wire as this struct.
//
// Success is set only if the function did not throw an exception.
type ThriftTest_TestMulti_Result struct {
	// Value returned by testMulti after a successful execution.
	Success *Xtruct `json:"success,omitempty"`
}

// ToWire translates a ThriftTest_TestMulti_Result struct into a Thrift-level intermediate
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
func (v *ThriftTest_TestMulti_Result) ToWire() (wire.Value, error) {
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
		return wire.Value{}, fmt.Errorf("ThriftTest_TestMulti_Result should have exactly one field: got %v fields", i)
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

// FromWire deserializes a ThriftTest_TestMulti_Result struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a ThriftTest_TestMulti_Result struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v ThriftTest_TestMulti_Result
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *ThriftTest_TestMulti_Result) FromWire(w wire.Value) error {
	var err error

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TStruct {
				v.Success, err = _Xtruct_Read(field.Value)
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
		return fmt.Errorf("ThriftTest_TestMulti_Result should have exactly one field: got %v fields", count)
	}

	return nil
}

// String returns a readable string representation of a ThriftTest_TestMulti_Result
// struct.
func (v *ThriftTest_TestMulti_Result) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}

	return fmt.Sprintf("ThriftTest_TestMulti_Result{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this ThriftTest_TestMulti_Result match the
// provided ThriftTest_TestMulti_Result.
//
// This function performs a deep comparison.
func (v *ThriftTest_TestMulti_Result) Equals(rhs *ThriftTest_TestMulti_Result) bool {
	if v == nil {
		return rhs == nil
	} else if rhs == nil {
		return false
	}
	if !((v.Success == nil && rhs.Success == nil) || (v.Success != nil && rhs.Success != nil && v.Success.Equals(rhs.Success))) {
		return false
	}

	return true
}

// MarshalLogObject implements zapcore.ObjectMarshaler, enabling
// fast logging of ThriftTest_TestMulti_Result.
func (v *ThriftTest_TestMulti_Result) MarshalLogObject(enc zapcore.ObjectEncoder) (err error) {
	if v == nil {
		return nil
	}
	if v.Success != nil {
		err = multierr.Append(err, enc.AddObject("success", v.Success))
	}
	return err
}

// GetSuccess returns the value of Success if it is set or its
// zero value if it is unset.
func (v *ThriftTest_TestMulti_Result) GetSuccess() (o *Xtruct) {
	if v.Success != nil {
		return v.Success
	}

	return
}

// IsSetSuccess returns true if Success is not nil.
func (v *ThriftTest_TestMulti_Result) IsSetSuccess() bool {
	return v.Success != nil
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the result.
//
// This will always be "testMulti" for this struct.
func (v *ThriftTest_TestMulti_Result) MethodName() string {
	return "testMulti"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Reply for this struct.
func (v *ThriftTest_TestMulti_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
