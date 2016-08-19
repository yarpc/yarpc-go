// Code generated by thriftrw
// @generated

package thrifttest

import (
	"fmt"
	"github.com/thriftrw/thriftrw-go/wire"
	"strings"
)

type TestVoidArgs struct{}

func (v *TestVoidArgs) ToWire() (wire.Value, error) {
	var (
		fields [0]wire.Field
		i      int = 0
	)
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *TestVoidArgs) FromWire(w wire.Value) error {
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		}
	}
	return nil
}

func (v *TestVoidArgs) String() string {
	var fields [0]string
	i := 0
	return fmt.Sprintf("TestVoidArgs{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestVoidArgs) MethodName() string {
	return "testVoid"
}

func (v *TestVoidArgs) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

type TestVoidResult struct{}

func (v *TestVoidResult) ToWire() (wire.Value, error) {
	var (
		fields [0]wire.Field
		i      int = 0
	)
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *TestVoidResult) FromWire(w wire.Value) error {
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		}
	}
	return nil
}

func (v *TestVoidResult) String() string {
	var fields [0]string
	i := 0
	return fmt.Sprintf("TestVoidResult{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestVoidResult) MethodName() string {
	return "testVoid"
}

func (v *TestVoidResult) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}

var TestVoidHelper = struct {
	IsException    func(error) bool
	Args           func() *TestVoidArgs
	WrapResponse   func(error) (*TestVoidResult, error)
	UnwrapResponse func(*TestVoidResult) error
}{}

func init() {
	TestVoidHelper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	TestVoidHelper.Args = func() *TestVoidArgs {
		return &TestVoidArgs{}
	}
	TestVoidHelper.WrapResponse = func(err error) (*TestVoidResult, error) {
		if err == nil {
			return &TestVoidResult{}, nil
		}
		return nil, err
	}
	TestVoidHelper.UnwrapResponse = func(result *TestVoidResult) (err error) {
		return
	}
}
