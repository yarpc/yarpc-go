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

type TestMultiArgs struct {
	Arg0 *int8             `json:"arg0,omitempty"`
	Arg1 *int32            `json:"arg1,omitempty"`
	Arg2 *int64            `json:"arg2,omitempty"`
	Arg3 map[int16]string  `json:"arg3"`
	Arg4 *gauntlet.Numberz `json:"arg4,omitempty"`
	Arg5 *gauntlet.UserId  `json:"arg5,omitempty"`
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

func (_Map_I16_String_MapItemList) Close() {
}

func (v *TestMultiArgs) ToWire() (wire.Value, error) {
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

func (v *TestMultiArgs) FromWire(w wire.Value) error {
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
				var x gauntlet.Numberz
				x, err = _Numberz_Read(field.Value)
				v.Arg4 = &x
				if err != nil {
					return err
				}
			}
		case 6:
			if field.Value.Type() == wire.TI64 {
				var x gauntlet.UserId
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

func (v *TestMultiArgs) String() string {
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
	return fmt.Sprintf("TestMultiArgs{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestMultiArgs) MethodName() string {
	return "testMulti"
}

func (v *TestMultiArgs) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

type TestMultiResult struct {
	Success *gauntlet.Xtruct `json:"success,omitempty"`
}

func (v *TestMultiResult) ToWire() (wire.Value, error) {
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
		return wire.Value{}, fmt.Errorf("TestMultiResult should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _Xtruct_Read(w wire.Value) (*gauntlet.Xtruct, error) {
	var v gauntlet.Xtruct
	err := v.FromWire(w)
	return &v, err
}

func (v *TestMultiResult) FromWire(w wire.Value) error {
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
		return fmt.Errorf("TestMultiResult should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *TestMultiResult) String() string {
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", v.Success)
		i++
	}
	return fmt.Sprintf("TestMultiResult{%v}", strings.Join(fields[:i], ", "))
}

func (v *TestMultiResult) MethodName() string {
	return "testMulti"
}

func (v *TestMultiResult) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}

var TestMultiHelper = struct {
	IsException    func(error) bool
	Args           func(arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *gauntlet.Numberz, arg5 *gauntlet.UserId) *TestMultiArgs
	WrapResponse   func(*gauntlet.Xtruct, error) (*TestMultiResult, error)
	UnwrapResponse func(*TestMultiResult) (*gauntlet.Xtruct, error)
}{}

func init() {
	TestMultiHelper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	TestMultiHelper.Args = func(arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *gauntlet.Numberz, arg5 *gauntlet.UserId) *TestMultiArgs {
		return &TestMultiArgs{Arg0: arg0, Arg1: arg1, Arg2: arg2, Arg3: arg3, Arg4: arg4, Arg5: arg5}
	}
	TestMultiHelper.WrapResponse = func(success *gauntlet.Xtruct, err error) (*TestMultiResult, error) {
		if err == nil {
			return &TestMultiResult{Success: success}, nil
		}
		return nil, err
	}
	TestMultiHelper.UnwrapResponse = func(result *TestMultiResult) (success *gauntlet.Xtruct, err error) {
		if result.Success != nil {
			success = result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}
