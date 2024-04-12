// Code generated by thriftrw-plugin-yarpc
// @generated

// Copyright (c) 2024 Uber Technologies, Inc.
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

package thrifttestclient

import (
	context "context"
	wire "go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc"
	transport "go.uber.org/yarpc/api/transport"
	thrift "go.uber.org/yarpc/encoding/thrift"
	gauntlet "go.uber.org/yarpc/internal/crossdock/thrift/gauntlet"
	reflect "reflect"
)

// Interface is a client for the ThriftTest service.
type Interface interface {
	TestBinary(
		ctx context.Context,
		Thing []byte,
		opts ...yarpc.CallOption,
	) ([]byte, error)

	TestByte(
		ctx context.Context,
		Thing *int8,
		opts ...yarpc.CallOption,
	) (int8, error)

	TestDouble(
		ctx context.Context,
		Thing *float64,
		opts ...yarpc.CallOption,
	) (float64, error)

	TestEnum(
		ctx context.Context,
		Thing *gauntlet.Numberz,
		opts ...yarpc.CallOption,
	) (gauntlet.Numberz, error)

	TestException(
		ctx context.Context,
		Arg *string,
		opts ...yarpc.CallOption,
	) error

	TestI32(
		ctx context.Context,
		Thing *int32,
		opts ...yarpc.CallOption,
	) (int32, error)

	TestI64(
		ctx context.Context,
		Thing *int64,
		opts ...yarpc.CallOption,
	) (int64, error)

	TestInsanity(
		ctx context.Context,
		Argument *gauntlet.Insanity,
		opts ...yarpc.CallOption,
	) (map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, error)

	TestList(
		ctx context.Context,
		Thing []int32,
		opts ...yarpc.CallOption,
	) ([]int32, error)

	TestMap(
		ctx context.Context,
		Thing map[int32]int32,
		opts ...yarpc.CallOption,
	) (map[int32]int32, error)

	TestMapMap(
		ctx context.Context,
		Hello *int32,
		opts ...yarpc.CallOption,
	) (map[int32]map[int32]int32, error)

	TestMulti(
		ctx context.Context,
		Arg0 *int8,
		Arg1 *int32,
		Arg2 *int64,
		Arg3 map[int16]string,
		Arg4 *gauntlet.Numberz,
		Arg5 *gauntlet.UserId,
		opts ...yarpc.CallOption,
	) (*gauntlet.Xtruct, error)

	TestMultiException(
		ctx context.Context,
		Arg0 *string,
		Arg1 *string,
		opts ...yarpc.CallOption,
	) (*gauntlet.Xtruct, error)

	TestNest(
		ctx context.Context,
		Thing *gauntlet.Xtruct2,
		opts ...yarpc.CallOption,
	) (*gauntlet.Xtruct2, error)

	TestOneway(
		ctx context.Context,
		SecondsToSleep *int32,
		opts ...yarpc.CallOption,
	) (yarpc.Ack, error)

	TestSet(
		ctx context.Context,
		Thing map[int32]struct{},
		opts ...yarpc.CallOption,
	) (map[int32]struct{}, error)

	TestString(
		ctx context.Context,
		Thing *string,
		opts ...yarpc.CallOption,
	) (string, error)

	TestStringMap(
		ctx context.Context,
		Thing map[string]string,
		opts ...yarpc.CallOption,
	) (map[string]string, error)

	TestStruct(
		ctx context.Context,
		Thing *gauntlet.Xtruct,
		opts ...yarpc.CallOption,
	) (*gauntlet.Xtruct, error)

	TestTypedef(
		ctx context.Context,
		Thing *gauntlet.UserId,
		opts ...yarpc.CallOption,
	) (gauntlet.UserId, error)

	TestVoid(
		ctx context.Context,
		opts ...yarpc.CallOption,
	) error
}

// New builds a new client for the ThriftTest service.
//
// 	client := thrifttestclient.New(dispatcher.ClientConfig("thrifttest"))
func New(c transport.ClientConfig, opts ...thrift.ClientOption) Interface {
	return client{
		c: thrift.New(thrift.Config{
			Service:      "ThriftTest",
			ClientConfig: c,
		}, opts...),
		nwc: thrift.NewNoWire(thrift.Config{
			Service:      "ThriftTest",
			ClientConfig: c,
		}, opts...),
	}
}

func init() {
	yarpc.RegisterClientBuilder(
		func(c transport.ClientConfig, f reflect.StructField) Interface {
			return New(c, thrift.ClientBuilderOptions(c, f)...)
		},
	)
}

type client struct {
	c   thrift.Client
	nwc thrift.NoWireClient
}

func (c client) TestBinary(
	ctx context.Context,
	_Thing []byte,
	opts ...yarpc.CallOption,
) (success []byte, err error) {

	var result gauntlet.ThriftTest_TestBinary_Result
	args := gauntlet.ThriftTest_TestBinary_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestBinary_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestByte(
	ctx context.Context,
	_Thing *int8,
	opts ...yarpc.CallOption,
) (success int8, err error) {

	var result gauntlet.ThriftTest_TestByte_Result
	args := gauntlet.ThriftTest_TestByte_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestByte_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestDouble(
	ctx context.Context,
	_Thing *float64,
	opts ...yarpc.CallOption,
) (success float64, err error) {

	var result gauntlet.ThriftTest_TestDouble_Result
	args := gauntlet.ThriftTest_TestDouble_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestDouble_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestEnum(
	ctx context.Context,
	_Thing *gauntlet.Numberz,
	opts ...yarpc.CallOption,
) (success gauntlet.Numberz, err error) {

	var result gauntlet.ThriftTest_TestEnum_Result
	args := gauntlet.ThriftTest_TestEnum_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestEnum_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestException(
	ctx context.Context,
	_Arg *string,
	opts ...yarpc.CallOption,
) (err error) {

	var result gauntlet.ThriftTest_TestException_Result
	args := gauntlet.ThriftTest_TestException_Helper.Args(_Arg)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	err = gauntlet.ThriftTest_TestException_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestI32(
	ctx context.Context,
	_Thing *int32,
	opts ...yarpc.CallOption,
) (success int32, err error) {

	var result gauntlet.ThriftTest_TestI32_Result
	args := gauntlet.ThriftTest_TestI32_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestI32_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestI64(
	ctx context.Context,
	_Thing *int64,
	opts ...yarpc.CallOption,
) (success int64, err error) {

	var result gauntlet.ThriftTest_TestI64_Result
	args := gauntlet.ThriftTest_TestI64_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestI64_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestInsanity(
	ctx context.Context,
	_Argument *gauntlet.Insanity,
	opts ...yarpc.CallOption,
) (success map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, err error) {

	var result gauntlet.ThriftTest_TestInsanity_Result
	args := gauntlet.ThriftTest_TestInsanity_Helper.Args(_Argument)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestInsanity_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestList(
	ctx context.Context,
	_Thing []int32,
	opts ...yarpc.CallOption,
) (success []int32, err error) {

	var result gauntlet.ThriftTest_TestList_Result
	args := gauntlet.ThriftTest_TestList_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestList_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestMap(
	ctx context.Context,
	_Thing map[int32]int32,
	opts ...yarpc.CallOption,
) (success map[int32]int32, err error) {

	var result gauntlet.ThriftTest_TestMap_Result
	args := gauntlet.ThriftTest_TestMap_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestMap_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestMapMap(
	ctx context.Context,
	_Hello *int32,
	opts ...yarpc.CallOption,
) (success map[int32]map[int32]int32, err error) {

	var result gauntlet.ThriftTest_TestMapMap_Result
	args := gauntlet.ThriftTest_TestMapMap_Helper.Args(_Hello)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestMapMap_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestMulti(
	ctx context.Context,
	_Arg0 *int8,
	_Arg1 *int32,
	_Arg2 *int64,
	_Arg3 map[int16]string,
	_Arg4 *gauntlet.Numberz,
	_Arg5 *gauntlet.UserId,
	opts ...yarpc.CallOption,
) (success *gauntlet.Xtruct, err error) {

	var result gauntlet.ThriftTest_TestMulti_Result
	args := gauntlet.ThriftTest_TestMulti_Helper.Args(_Arg0, _Arg1, _Arg2, _Arg3, _Arg4, _Arg5)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestMulti_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestMultiException(
	ctx context.Context,
	_Arg0 *string,
	_Arg1 *string,
	opts ...yarpc.CallOption,
) (success *gauntlet.Xtruct, err error) {

	var result gauntlet.ThriftTest_TestMultiException_Result
	args := gauntlet.ThriftTest_TestMultiException_Helper.Args(_Arg0, _Arg1)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestMultiException_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestNest(
	ctx context.Context,
	_Thing *gauntlet.Xtruct2,
	opts ...yarpc.CallOption,
) (success *gauntlet.Xtruct2, err error) {

	var result gauntlet.ThriftTest_TestNest_Result
	args := gauntlet.ThriftTest_TestNest_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestNest_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestOneway(
	ctx context.Context,
	_SecondsToSleep *int32,
	opts ...yarpc.CallOption,
) (yarpc.Ack, error) {
	args := gauntlet.ThriftTest_TestOneway_Helper.Args(_SecondsToSleep)
	return c.c.CallOneway(ctx, args, opts...)
}

func (c client) TestSet(
	ctx context.Context,
	_Thing map[int32]struct{},
	opts ...yarpc.CallOption,
) (success map[int32]struct{}, err error) {

	var result gauntlet.ThriftTest_TestSet_Result
	args := gauntlet.ThriftTest_TestSet_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestSet_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestString(
	ctx context.Context,
	_Thing *string,
	opts ...yarpc.CallOption,
) (success string, err error) {

	var result gauntlet.ThriftTest_TestString_Result
	args := gauntlet.ThriftTest_TestString_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestString_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestStringMap(
	ctx context.Context,
	_Thing map[string]string,
	opts ...yarpc.CallOption,
) (success map[string]string, err error) {

	var result gauntlet.ThriftTest_TestStringMap_Result
	args := gauntlet.ThriftTest_TestStringMap_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestStringMap_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestStruct(
	ctx context.Context,
	_Thing *gauntlet.Xtruct,
	opts ...yarpc.CallOption,
) (success *gauntlet.Xtruct, err error) {

	var result gauntlet.ThriftTest_TestStruct_Result
	args := gauntlet.ThriftTest_TestStruct_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestStruct_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestTypedef(
	ctx context.Context,
	_Thing *gauntlet.UserId,
	opts ...yarpc.CallOption,
) (success gauntlet.UserId, err error) {

	var result gauntlet.ThriftTest_TestTypedef_Result
	args := gauntlet.ThriftTest_TestTypedef_Helper.Args(_Thing)

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	success, err = gauntlet.ThriftTest_TestTypedef_Helper.UnwrapResponse(&result)
	return
}

func (c client) TestVoid(
	ctx context.Context,
	opts ...yarpc.CallOption,
) (err error) {

	var result gauntlet.ThriftTest_TestVoid_Result
	args := gauntlet.ThriftTest_TestVoid_Helper.Args()

	if c.nwc != nil && c.nwc.Enabled() {
		if err = c.nwc.Call(ctx, args, &result, opts...); err != nil {
			return
		}
	} else {
		var body wire.Value
		if body, err = c.c.Call(ctx, args, opts...); err != nil {
			return
		}

		if err = result.FromWire(body); err != nil {
			return
		}
	}

	err = gauntlet.ThriftTest_TestVoid_Helper.UnwrapResponse(&result)
	return
}
