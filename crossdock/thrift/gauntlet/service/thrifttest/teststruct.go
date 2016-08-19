// Code generated by thriftrw
// @generated

package thrifttest

import (
	"errors"
	"fmt"
	"github.com/thriftrw/thriftrw-go/wire"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet"
	"strings"
)

type TestStructArgs struct {
	Thing *gauntlet.Xtruct `json:"thing,omitempty"`
}

func (v *TestStructArgs) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Thing != nil {
		w, err = v.Thing.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *TestStructArgs) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TStruct {
				v.Thing, err = _Xtruct_Read(field.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *TestStructArgs) String() string {
	var fields [1]string
	i := 0
	if v.Thing != nil {
		fields[i] = fmt.Sprintf("Thing: %v", v.Thing)
		i++
	}
	return fmt.Sprintf("TestStructArgs{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestStructArgs) MethodName() string {
	return "testStruct"
}

func (v *TestStructArgs) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

type TestStructResult struct {
	Success *gauntlet.Xtruct `json:"success,omitempty"`
}

func (v *TestStructResult) ToWire() (wire.Value, error) {
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
		return wire.Value{}, fmt.Errorf("TestStructResult should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *TestStructResult) FromWire(w wire.Value) error {
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
		return fmt.Errorf("TestStructResult should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *TestStructResult) String() string {
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("TestStructResult{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestStructResult) MethodName() string {
	return "testStruct"
}

func (v *TestStructResult) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}

var TestStructHelper = struct {
	IsException    func(error) bool
	Args           func(thing *gauntlet.Xtruct) *TestStructArgs
	WrapResponse   func(*gauntlet.Xtruct, error) (*TestStructResult, error)
	UnwrapResponse func(*TestStructResult) (*gauntlet.Xtruct, error)
}{}

func init() {
	TestStructHelper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	TestStructHelper.Args = func(thing *gauntlet.Xtruct) *TestStructArgs {
		return &TestStructArgs{Thing: thing}
	}
	TestStructHelper.WrapResponse = func(success *gauntlet.Xtruct, err error) (*TestStructResult, error) {
		if err == nil {
			return &TestStructResult{Success: success}, nil
		}
		return nil, err
	}
	TestStructHelper.UnwrapResponse = func(result *TestStructResult) (success *gauntlet.Xtruct, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}
