// Code generated by thriftrw v1.4.0
// @generated

package atomic

import (
	"errors"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

type CompareAndSwap struct {
	Key          string `json:"key"`
	CurrentValue int64  `json:"currentValue"`
	NewValue     int64  `json:"newValue"`
}

func (v *CompareAndSwap) ToWire() (wire.Value, error) {
	var (
		fields [3]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	w, err = wire.NewValueString(v.Key), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 1, Value: w}
	i++
	w, err = wire.NewValueI64(v.CurrentValue), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 2, Value: w}
	i++
	w, err = wire.NewValueI64(v.NewValue), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 3, Value: w}
	i++
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *CompareAndSwap) FromWire(w wire.Value) error {
	var err error
	keyIsSet := false
	currentValueIsSet := false
	newValueIsSet := false
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				v.Key, err = field.Value.GetString(), error(nil)
				if err != nil {
					return err
				}
				keyIsSet = true
			}
		case 2:
			if field.Value.Type() == wire.TI64 {
				v.CurrentValue, err = field.Value.GetI64(), error(nil)
				if err != nil {
					return err
				}
				currentValueIsSet = true
			}
		case 3:
			if field.Value.Type() == wire.TI64 {
				v.NewValue, err = field.Value.GetI64(), error(nil)
				if err != nil {
					return err
				}
				newValueIsSet = true
			}
		}
	}
	if !keyIsSet {
		return errors.New("field Key of CompareAndSwap is required")
	}
	if !currentValueIsSet {
		return errors.New("field CurrentValue of CompareAndSwap is required")
	}
	if !newValueIsSet {
		return errors.New("field NewValue of CompareAndSwap is required")
	}
	return nil
}

func (v *CompareAndSwap) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [3]string
	i := 0
	fields[i] = fmt.Sprintf("Key: %v", v.Key)
	i++
	fields[i] = fmt.Sprintf("CurrentValue: %v", v.CurrentValue)
	i++
	fields[i] = fmt.Sprintf("NewValue: %v", v.NewValue)
	i++
	return fmt.Sprintf("CompareAndSwap{%v}", strings.Join(fields[:i], ", "))
}

func (v *CompareAndSwap) Equals(rhs *CompareAndSwap) bool {
	if !(v.Key == rhs.Key) {
		return false
	}
	if !(v.CurrentValue == rhs.CurrentValue) {
		return false
	}
	if !(v.NewValue == rhs.NewValue) {
		return false
	}
	return true
}

type IntegerMismatchError struct {
	ExpectedValue int64 `json:"expectedValue"`
	GotValue      int64 `json:"gotValue"`
}

func (v *IntegerMismatchError) ToWire() (wire.Value, error) {
	var (
		fields [2]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	w, err = wire.NewValueI64(v.ExpectedValue), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 1, Value: w}
	i++
	w, err = wire.NewValueI64(v.GotValue), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 2, Value: w}
	i++
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *IntegerMismatchError) FromWire(w wire.Value) error {
	var err error
	expectedValueIsSet := false
	gotValueIsSet := false
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TI64 {
				v.ExpectedValue, err = field.Value.GetI64(), error(nil)
				if err != nil {
					return err
				}
				expectedValueIsSet = true
			}
		case 2:
			if field.Value.Type() == wire.TI64 {
				v.GotValue, err = field.Value.GetI64(), error(nil)
				if err != nil {
					return err
				}
				gotValueIsSet = true
			}
		}
	}
	if !expectedValueIsSet {
		return errors.New("field ExpectedValue of IntegerMismatchError is required")
	}
	if !gotValueIsSet {
		return errors.New("field GotValue of IntegerMismatchError is required")
	}
	return nil
}

func (v *IntegerMismatchError) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [2]string
	i := 0
	fields[i] = fmt.Sprintf("ExpectedValue: %v", v.ExpectedValue)
	i++
	fields[i] = fmt.Sprintf("GotValue: %v", v.GotValue)
	i++
	return fmt.Sprintf("IntegerMismatchError{%v}", strings.Join(fields[:i], ", "))
}

func (v *IntegerMismatchError) Equals(rhs *IntegerMismatchError) bool {
	if !(v.ExpectedValue == rhs.ExpectedValue) {
		return false
	}
	if !(v.GotValue == rhs.GotValue) {
		return false
	}
	return true
}

func (v *IntegerMismatchError) Error() string {
	return v.String()
}

type KeyDoesNotExist struct {
	Key *string `json:"key,omitempty"`
}

func (v *KeyDoesNotExist) ToWire() (wire.Value, error) {
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

func (v *KeyDoesNotExist) FromWire(w wire.Value) error {
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

func (v *KeyDoesNotExist) String() string {
	if v == nil {
		return "<nil>"
	}
	var fields [1]string
	i := 0
	if v.Key != nil {
		fields[i] = fmt.Sprintf("Key: %v", *(v.Key))
		i++
	}
	return fmt.Sprintf("KeyDoesNotExist{%v}", strings.Join(fields[:i], ", "))
}

func _String_EqualsPtr(lhs, rhs *string) bool {
	if lhs != nil && rhs != nil {
		x := *lhs
		y := *rhs
		return (x == y)
	}
	return lhs == nil && rhs == nil
}

func (v *KeyDoesNotExist) Equals(rhs *KeyDoesNotExist) bool {
	if !_String_EqualsPtr(v.Key, rhs.Key) {
		return false
	}
	return true
}

func (v *KeyDoesNotExist) Error() string {
	return v.String()
}
