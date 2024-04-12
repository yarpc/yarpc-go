// Code generated by thriftrw v1.31.0. DO NOT EDIT.
// @generated

package extends

import (
	errors "errors"
	fmt "fmt"
	stream "go.uber.org/thriftrw/protocol/stream"
	thriftreflect "go.uber.org/thriftrw/thriftreflect"
	wire "go.uber.org/thriftrw/wire"
	zapcore "go.uber.org/zap/zapcore"
	strings "strings"
)

// ThriftModule represents the IDL file used to generate this package.
var ThriftModule = &thriftreflect.ThriftModule{
	Name:     "extends",
	Package:  "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends",
	FilePath: "extends.thrift",
	SHA1:     "5dc89427890e0f8f94285fc648815882150591b9",
	Raw:      rawIDL,
}

const rawIDL = "service Name {\n\tstring name()\n}\nservice Foo extends Name {}\nservice Bar extends Foo {}\n"

// Name_Name_Args represents the arguments for the Name.name function.
//
// The arguments for name are sent and received over the wire as this struct.
type Name_Name_Args struct {
}

// ToWire translates a Name_Name_Args struct into a Thrift-level intermediate
// representation. This intermediate representation may be serialized
// into bytes using a ThriftRW protocol implementation.
//
// An error is returned if the struct or any of its fields failed to
// validate.
//
//	x, err := v.ToWire()
//	if err != nil {
//		return err
//	}
//
//	if err := binaryProtocol.Encode(x, writer); err != nil {
//		return err
//	}
func (v *Name_Name_Args) ToWire() (wire.Value, error) {
	var (
		fields [0]wire.Field
		i      int = 0
	)

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

// FromWire deserializes a Name_Name_Args struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a Name_Name_Args struct
// from the provided intermediate representation.
//
//	x, err := binaryProtocol.Decode(reader, wire.TStruct)
//	if err != nil {
//		return nil, err
//	}
//
//	var v Name_Name_Args
//	if err := v.FromWire(x); err != nil {
//		return nil, err
//	}
//	return &v, nil
func (v *Name_Name_Args) FromWire(w wire.Value) error {

	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		}
	}

	return nil
}

// Encode serializes a Name_Name_Args struct directly into bytes, without going
// through an intermediary type.
//
// An error is returned if a Name_Name_Args struct could not be encoded.
func (v *Name_Name_Args) Encode(sw stream.Writer) error {
	if err := sw.WriteStructBegin(); err != nil {
		return err
	}

	return sw.WriteStructEnd()
}

// Decode deserializes a Name_Name_Args struct directly from its Thrift-level
// representation, without going through an intemediary type.
//
// An error is returned if a Name_Name_Args struct could not be generated from the wire
// representation.
func (v *Name_Name_Args) Decode(sr stream.Reader) error {

	if err := sr.ReadStructBegin(); err != nil {
		return err
	}

	fh, ok, err := sr.ReadFieldBegin()
	if err != nil {
		return err
	}

	for ok {
		switch {
		default:
			if err := sr.Skip(fh.Type); err != nil {
				return err
			}
		}

		if err := sr.ReadFieldEnd(); err != nil {
			return err
		}

		if fh, ok, err = sr.ReadFieldBegin(); err != nil {
			return err
		}
	}

	if err := sr.ReadStructEnd(); err != nil {
		return err
	}

	return nil
}

// String returns a readable string representation of a Name_Name_Args
// struct.
func (v *Name_Name_Args) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [0]string
	i := 0

	return fmt.Sprintf("Name_Name_Args{%v}", strings.Join(fields[:i], ", "))
}

// Equals returns true if all the fields of this Name_Name_Args match the
// provided Name_Name_Args.
//
// This function performs a deep comparison.
func (v *Name_Name_Args) Equals(rhs *Name_Name_Args) bool {
	if v == nil {
		return rhs == nil
	} else if rhs == nil {
		return false
	}

	return true
}

// MarshalLogObject implements zapcore.ObjectMarshaler, enabling
// fast logging of Name_Name_Args.
func (v *Name_Name_Args) MarshalLogObject(enc zapcore.ObjectEncoder) (err error) {
	if v == nil {
		return nil
	}
	return err
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the arguments.
//
// This will always be "name" for this struct.
func (v *Name_Name_Args) MethodName() string {
	return "name"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Call for this struct.
func (v *Name_Name_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

// Name_Name_Helper provides functions that aid in handling the
// parameters and return values of the Name.name
// function.
var Name_Name_Helper = struct {
	// Args accepts the parameters of name in-order and returns
	// the arguments struct for the function.
	Args func() *Name_Name_Args

	// IsException returns true if the given error can be thrown
	// by name.
	//
	// An error can be thrown by name only if the
	// corresponding exception type was mentioned in the 'throws'
	// section for it in the Thrift file.
	IsException func(error) bool

	// WrapResponse returns the result struct for name
	// given its return value and error.
	//
	// This allows mapping values and errors returned by
	// name into a serializable result struct.
	// WrapResponse returns a non-nil error if the provided
	// error cannot be thrown by name
	//
	//   value, err := name(args)
	//   result, err := Name_Name_Helper.WrapResponse(value, err)
	//   if err != nil {
	//     return fmt.Errorf("unexpected error from name: %v", err)
	//   }
	//   serialize(result)
	WrapResponse func(string, error) (*Name_Name_Result, error)

	// UnwrapResponse takes the result struct for name
	// and returns the value or error returned by it.
	//
	// The error is non-nil only if name threw an
	// exception.
	//
	//   result := deserialize(bytes)
	//   value, err := Name_Name_Helper.UnwrapResponse(result)
	UnwrapResponse func(*Name_Name_Result) (string, error)
}{}

func init() {
	Name_Name_Helper.Args = func() *Name_Name_Args {
		return &Name_Name_Args{}
	}

	Name_Name_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}

	Name_Name_Helper.WrapResponse = func(success string, err error) (*Name_Name_Result, error) {
		if err == nil {
			return &Name_Name_Result{Success: &success}, nil
		}

		return nil, err
	}
	Name_Name_Helper.UnwrapResponse = func(result *Name_Name_Result) (success string, err error) {

		if result.Success != nil {
			success = *result.Success
			return
		}

		err = errors.New("expected a non-void result")
		return
	}

}

// Name_Name_Result represents the result of a Name.name function call.
//
// The result of a name execution is sent and received over the wire as this struct.
//
// Success is set only if the function did not throw an exception.
type Name_Name_Result struct {
	// Value returned by name after a successful execution.
	Success *string `json:"success,omitempty"`
}

// ToWire translates a Name_Name_Result struct into a Thrift-level intermediate
// representation. This intermediate representation may be serialized
// into bytes using a ThriftRW protocol implementation.
//
// An error is returned if the struct or any of its fields failed to
// validate.
//
//	x, err := v.ToWire()
//	if err != nil {
//		return err
//	}
//
//	if err := binaryProtocol.Encode(x, writer); err != nil {
//		return err
//	}
func (v *Name_Name_Result) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
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

	if i != 1 {
		return wire.Value{}, fmt.Errorf("Name_Name_Result should have exactly one field: got %v fields", i)
	}

	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

// FromWire deserializes a Name_Name_Result struct from its Thrift-level
// representation. The Thrift-level representation may be obtained
// from a ThriftRW protocol implementation.
//
// An error is returned if we were unable to build a Name_Name_Result struct
// from the provided intermediate representation.
//
//	x, err := binaryProtocol.Decode(reader, wire.TStruct)
//	if err != nil {
//		return nil, err
//	}
//
//	var v Name_Name_Result
//	if err := v.FromWire(x); err != nil {
//		return nil, err
//	}
//	return &v, nil
func (v *Name_Name_Result) FromWire(w wire.Value) error {
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
		}
	}

	count := 0
	if v.Success != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("Name_Name_Result should have exactly one field: got %v fields", count)
	}

	return nil
}

// Encode serializes a Name_Name_Result struct directly into bytes, without going
// through an intermediary type.
//
// An error is returned if a Name_Name_Result struct could not be encoded.
func (v *Name_Name_Result) Encode(sw stream.Writer) error {
	if err := sw.WriteStructBegin(); err != nil {
		return err
	}

	if v.Success != nil {
		if err := sw.WriteFieldBegin(stream.FieldHeader{ID: 0, Type: wire.TBinary}); err != nil {
			return err
		}
		if err := sw.WriteString(*(v.Success)); err != nil {
			return err
		}
		if err := sw.WriteFieldEnd(); err != nil {
			return err
		}
	}

	count := 0
	if v.Success != nil {
		count++
	}

	if count != 1 {
		return fmt.Errorf("Name_Name_Result should have exactly one field: got %v fields", count)
	}

	return sw.WriteStructEnd()
}

// Decode deserializes a Name_Name_Result struct directly from its Thrift-level
// representation, without going through an intemediary type.
//
// An error is returned if a Name_Name_Result struct could not be generated from the wire
// representation.
func (v *Name_Name_Result) Decode(sr stream.Reader) error {

	if err := sr.ReadStructBegin(); err != nil {
		return err
	}

	fh, ok, err := sr.ReadFieldBegin()
	if err != nil {
		return err
	}

	for ok {
		switch {
		case fh.ID == 0 && fh.Type == wire.TBinary:
			var x string
			x, err = sr.ReadString()
			v.Success = &x
			if err != nil {
				return err
			}

		default:
			if err := sr.Skip(fh.Type); err != nil {
				return err
			}
		}

		if err := sr.ReadFieldEnd(); err != nil {
			return err
		}

		if fh, ok, err = sr.ReadFieldBegin(); err != nil {
			return err
		}
	}

	if err := sr.ReadStructEnd(); err != nil {
		return err
	}

	count := 0
	if v.Success != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("Name_Name_Result should have exactly one field: got %v fields", count)
	}

	return nil
}

// String returns a readable string representation of a Name_Name_Result
// struct.
func (v *Name_Name_Result) String() string {
	if v == nil {
		return "<nil>"
	}

	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", *(v.Success))
		i++
	}

	return fmt.Sprintf("Name_Name_Result{%v}", strings.Join(fields[:i], ", "))
}

func _String_EqualsPtr(lhs, rhs *string) bool {
	if lhs != nil && rhs != nil {

		x := *lhs
		y := *rhs
		return (x == y)
	}
	return lhs == nil && rhs == nil
}

// Equals returns true if all the fields of this Name_Name_Result match the
// provided Name_Name_Result.
//
// This function performs a deep comparison.
func (v *Name_Name_Result) Equals(rhs *Name_Name_Result) bool {
	if v == nil {
		return rhs == nil
	} else if rhs == nil {
		return false
	}
	if !_String_EqualsPtr(v.Success, rhs.Success) {
		return false
	}

	return true
}

// MarshalLogObject implements zapcore.ObjectMarshaler, enabling
// fast logging of Name_Name_Result.
func (v *Name_Name_Result) MarshalLogObject(enc zapcore.ObjectEncoder) (err error) {
	if v == nil {
		return nil
	}
	if v.Success != nil {
		enc.AddString("success", *v.Success)
	}
	return err
}

// GetSuccess returns the value of Success if it is set or its
// zero value if it is unset.
func (v *Name_Name_Result) GetSuccess() (o string) {
	if v != nil && v.Success != nil {
		return *v.Success
	}

	return
}

// IsSetSuccess returns true if Success is not nil.
func (v *Name_Name_Result) IsSetSuccess() bool {
	return v != nil && v.Success != nil
}

// MethodName returns the name of the Thrift function as specified in
// the IDL, for which this struct represent the result.
//
// This will always be "name" for this struct.
func (v *Name_Name_Result) MethodName() string {
	return "name"
}

// EnvelopeType returns the kind of value inside this struct.
//
// This will always be Reply for this struct.
func (v *Name_Name_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}
