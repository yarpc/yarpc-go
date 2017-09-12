// Code generated by thriftrw v1.7.0. DO NOT EDIT.
// @generated

package atomic

import (
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

// Store_Increment_Args represents the arguments for the Store.increment function.
//
// The arguments for increment are sent and received over the wire as this struct.
type Store_Increment_Args struct {
	Key   *string `json:"key,omitempty"`
	Value *int64  `json:"value,omitempty"`
}

// ToWire translates a Store_Increment_Args struct into a Thrift-level intermediate
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
func (v *Store_Increment_Args) ToWire() (wire.Value, error) {
	var (
		fields [2]wire.Field
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
	if v.Value != nil {
		w, err = wire.NewValueI64(*(v.Value)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 2, Value: w}
		i++
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

// FromWire deserializes a Store_Increment_Args struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a Store_Increment_Args struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v Store_Increment_Args
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *Store_Increment_Args) FromWire(w wire.Value) error {
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
		case 2:
			if field.Value.Type() == wire.TI64 {
				var x int64
				x, err = field.Value.GetI64(), error(nil)
				v.Value = &x
				if err != nil {
					return err
				}

			}
		}
	}

	return nil
}

// String returns a readable string representation of a Store_Increment_Args
// struct.
func (v *Store_Increment_Args) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [2]string
	i := 0
	if v.Key != nil {
		fields[i] = fmt.Sprintf("Key: %v", *(v.Key))
		i++
	}
	if v.Value != nil {
		fields[i] = fmt.Sprintf("Value: %v", *(v.Value))
		i++
	}

	return fmt.Sprintf("Store_Increment_Args{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this Store_Increment_Args match the
// provided Store_Increment_Args.
//
// This function performs a deep comparison.
func (v *Store_Increment_Args) Equals(rhs *Store_Increment_Args) bool {
	if !_String_EqualsPtr(v.Key, rhs.Key) {
		return false
	}
	if !_I64_EqualsPtr(v.Value, rhs.Value) {
		return false
	}

	return true
}

// GetKey returns the value of Key if it is set or its
// zero value if it is unset.
func (v *Store_Increment_Args) GetKey() (o string) {
	if v.Key != nil {
		return *v.Key
	}

	return
}

// GetValue returns the value of Value if it is set or its
// zero value if it is unset.
func (v *Store_Increment_Args) GetValue() (o int64) {
	if v.Value != nil {
		return *v.Value
	}

	return
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the arguments.
//
// This will always be "increment" for this struct.
func (v *Store_Increment_Args) MethodName() string {
	return "increment"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Call for this struct.
func (v *Store_Increment_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

// Store_Increment_Helper provides functions that aid in handling the
// parameters and return values of the Store.increment
// function.
var Store_Increment_Helper = struct {
	// Args accepts the parameters of increment in-order and returns
	// the arguments struct for the function.
	Args func(
		key *string,
		value *int64,
	) *Store_Increment_Args

	// IsException returns true if the given error can be thrown
	// by increment.
	//
	// An error can be thrown by increment only if the
	// corresponding exception type was mentioned in the 'throws'
	// section for it in the Thrift file.
	IsException func(error) bool

	// WrapResponse returns the result struct for increment
	// given the error returned by it. The provided error may
	// be nil if increment did not fail.
	//
	// This allows mapping errors returned by increment into a
	// serializable result struct. WrapResponse returns a
	// non-nil error if the provided error cannot be thrown by
	// increment
	//
	//   err := increment(args)
	//   result, err := Store_Increment_Helper.WrapResponse(err)
	//   if err != nil {
	//     return fmt.Errorf("unexpected error from increment: %v", err)
	//   }
	//   serialize(result)
	WrapResponse func(error) (*Store_Increment_Result, error)

	// UnwrapResponse takes the result struct for increment
	// and returns the erorr returned by it (if any).
	//
	// The error is non-nil only if increment threw an
	// exception.
	//
	//   result := deserialize(bytes)
	//   err := Store_Increment_Helper.UnwrapResponse(result)
	UnwrapResponse func(*Store_Increment_Result) error
}{}

func init() {
	Store_Increment_Helper.Args = func(
		key *string,
		value *int64,
	) *Store_Increment_Args {
		return &Store_Increment_Args{
			Key:   key,
			Value: value,
		}
	}

	Store_Increment_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}

	Store_Increment_Helper.WrapResponse = func(err error) (*Store_Increment_Result, error) {
		if err == nil {
			return &Store_Increment_Result{}, nil
		}

		return nil, err
	}
	Store_Increment_Helper.UnwrapResponse = func(result *Store_Increment_Result) (err error) {
		return
	}

}

// Store_Increment_Result represents the result of a Store.increment function call.
//
// The result of a increment execution is sent and received over the wire as this struct.
type Store_Increment_Result struct {
}

// ToWire translates a Store_Increment_Result struct into a Thrift-level intermediate
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
func (v *Store_Increment_Result) ToWire() (wire.Value, error) {
	var (
		fields [0]wire.Field
		i      int = 0
	)

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

// FromWire deserializes a Store_Increment_Result struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a Store_Increment_Result struct
// from the provided intermediate representation.
//
//   x, err := binaryProtocol.Decode(reader, wire.TStruct)
//   if err != nil {
//     return nil, err
//   }
//
//   var v Store_Increment_Result
//   if err := v.FromWire(x); err != nil {
//     return nil, err
//   }
//   return &v, nil
func (v *Store_Increment_Result) FromWire(w wire.Value) error {

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		}
	}

	return nil
}

// String returns a readable string representation of a Store_Increment_Result
// struct.
func (v *Store_Increment_Result) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [0]string
	i := 0

	return fmt.Sprintf("Store_Increment_Result{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this Store_Increment_Result match the
// provided Store_Increment_Result.
//
// This function performs a deep comparison.
func (v *Store_Increment_Result) Equals(rhs *Store_Increment_Result) bool {

	return true
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the result.
//
// This will always be "increment" for this struct.
func (v *Store_Increment_Result) MethodName() string {
	return "increment"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Reply for this struct.
func (v *Store_Increment_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
