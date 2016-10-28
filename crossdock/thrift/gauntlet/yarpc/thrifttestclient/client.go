// Code generated by thriftrw-plugin-yarpc
// @generated

// Copyright (c) 2016 Uber Technologies, Inc.
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
	"go.uber.org/thriftrw/wire"
	"context"
	"go.uber.org/yarpc/crossdock/thrift/gauntlet"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/crossdock/thrift/gauntlet/service/thrifttest"
	"go.uber.org/yarpc"
)

// Interface is a client for the ThriftTest service.
type Interface interface {
	TestBinary(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing []byte,
	) ([]byte, yarpc.CallResMeta, error)

	TestByte(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *int8,
	) (int8, yarpc.CallResMeta, error)

	TestDouble(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *float64,
	) (float64, yarpc.CallResMeta, error)

	TestEnum(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *gauntlet.Numberz,
	) (gauntlet.Numberz, yarpc.CallResMeta, error)

	TestException(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Arg *string,
	) (yarpc.CallResMeta, error)

	TestI32(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *int32,
	) (int32, yarpc.CallResMeta, error)

	TestI64(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *int64,
	) (int64, yarpc.CallResMeta, error)

	TestInsanity(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Argument *gauntlet.Insanity,
	) (map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, yarpc.CallResMeta, error)

	TestList(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing []int32,
	) ([]int32, yarpc.CallResMeta, error)

	TestMap(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing map[int32]int32,
	) (map[int32]int32, yarpc.CallResMeta, error)

	TestMapMap(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Hello *int32,
	) (map[int32]map[int32]int32, yarpc.CallResMeta, error)

	TestMulti(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Arg0 *int8,
		Arg1 *int32,
		Arg2 *int64,
		Arg3 map[int16]string,
		Arg4 *gauntlet.Numberz,
		Arg5 *gauntlet.UserId,
	) (*gauntlet.Xtruct, yarpc.CallResMeta, error)

	TestMultiException(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Arg0 *string,
		Arg1 *string,
	) (*gauntlet.Xtruct, yarpc.CallResMeta, error)

	TestNest(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *gauntlet.Xtruct2,
	) (*gauntlet.Xtruct2, yarpc.CallResMeta, error)

	TestSet(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing map[int32]struct{},
	) (map[int32]struct{}, yarpc.CallResMeta, error)

	TestString(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *string,
	) (string, yarpc.CallResMeta, error)

	TestStringMap(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing map[string]string,
	) (map[string]string, yarpc.CallResMeta, error)

	TestStruct(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *gauntlet.Xtruct,
	) (*gauntlet.Xtruct, yarpc.CallResMeta, error)

	TestTypedef(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
		Thing *gauntlet.UserId,
	) (gauntlet.UserId, yarpc.CallResMeta, error)

	TestVoid(
		ctx context.Context,
		reqMeta yarpc.CallReqMeta,
	) (yarpc.CallResMeta, error)
}

// New builds a new client for the ThriftTest service.
//
// 	client := thrifttestclient.New(dispatcher.Channel("thrifttest"))
func New(c transport.Channel, opts ...thrift.ClientOption) Interface {
	return client{c: thrift.New(thrift.Config{
		Service: "ThriftTest",
		Channel: c,
	}, opts...)}
}

func init() {
	yarpc.RegisterClientBuilder(func(c transport.Channel) Interface {
		return New(c)
	})
}

type client struct{ c thrift.Client }

func (c client) TestBinary(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing []byte,
) (success []byte, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestBinaryHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestBinaryResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestBinaryHelper.UnwrapResponse(&result)
	return
}

func (c client) TestByte(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *int8,
) (success int8, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestByteHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestByteResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestByteHelper.UnwrapResponse(&result)
	return
}

func (c client) TestDouble(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *float64,
) (success float64, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestDoubleHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestDoubleResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestDoubleHelper.UnwrapResponse(&result)
	return
}

func (c client) TestEnum(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *gauntlet.Numberz,
) (success gauntlet.Numberz, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestEnumHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestEnumResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestEnumHelper.UnwrapResponse(&result)
	return
}

func (c client) TestException(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Arg *string,
) (resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestExceptionHelper.Args(_Arg)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestExceptionResult
	if err = result.FromWire(body); err != nil {
		return
	}

	err = thrifttest.TestExceptionHelper.UnwrapResponse(&result)
	return
}

func (c client) TestI32(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *int32,
) (success int32, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestI32Helper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestI32Result
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestI32Helper.UnwrapResponse(&result)
	return
}

func (c client) TestI64(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *int64,
) (success int64, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestI64Helper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestI64Result
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestI64Helper.UnwrapResponse(&result)
	return
}

func (c client) TestInsanity(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Argument *gauntlet.Insanity,
) (success map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestInsanityHelper.Args(_Argument)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestInsanityResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestInsanityHelper.UnwrapResponse(&result)
	return
}

func (c client) TestList(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing []int32,
) (success []int32, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestListHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestListResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestListHelper.UnwrapResponse(&result)
	return
}

func (c client) TestMap(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing map[int32]int32,
) (success map[int32]int32, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestMapHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestMapResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestMapHelper.UnwrapResponse(&result)
	return
}

func (c client) TestMapMap(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Hello *int32,
) (success map[int32]map[int32]int32, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestMapMapHelper.Args(_Hello)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestMapMapResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestMapMapHelper.UnwrapResponse(&result)
	return
}

func (c client) TestMulti(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Arg0 *int8,
	_Arg1 *int32,
	_Arg2 *int64,
	_Arg3 map[int16]string,
	_Arg4 *gauntlet.Numberz,
	_Arg5 *gauntlet.UserId,
) (success *gauntlet.Xtruct, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestMultiHelper.Args(_Arg0, _Arg1, _Arg2, _Arg3, _Arg4, _Arg5)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestMultiResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestMultiHelper.UnwrapResponse(&result)
	return
}

func (c client) TestMultiException(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Arg0 *string,
	_Arg1 *string,
) (success *gauntlet.Xtruct, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestMultiExceptionHelper.Args(_Arg0, _Arg1)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestMultiExceptionResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestMultiExceptionHelper.UnwrapResponse(&result)
	return
}

func (c client) TestNest(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *gauntlet.Xtruct2,
) (success *gauntlet.Xtruct2, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestNestHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestNestResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestNestHelper.UnwrapResponse(&result)
	return
}

func (c client) TestSet(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing map[int32]struct{},
) (success map[int32]struct{}, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestSetHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestSetResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestSetHelper.UnwrapResponse(&result)
	return
}

func (c client) TestString(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *string,
) (success string, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestStringHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestStringResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestStringHelper.UnwrapResponse(&result)
	return
}

func (c client) TestStringMap(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing map[string]string,
) (success map[string]string, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestStringMapHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestStringMapResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestStringMapHelper.UnwrapResponse(&result)
	return
}

func (c client) TestStruct(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *gauntlet.Xtruct,
) (success *gauntlet.Xtruct, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestStructHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestStructResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestStructHelper.UnwrapResponse(&result)
	return
}

func (c client) TestTypedef(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
	_Thing *gauntlet.UserId,
) (success gauntlet.UserId, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestTypedefHelper.Args(_Thing)

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestTypedefResult
	if err = result.FromWire(body); err != nil {
		return
	}

	success, err = thrifttest.TestTypedefHelper.UnwrapResponse(&result)
	return
}

func (c client) TestVoid(
	ctx context.Context,
	reqMeta yarpc.CallReqMeta,
) (resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestVoidHelper.Args()

	var body wire.Value
	body, resMeta, err = c.c.Call(ctx, reqMeta, args)
	if err != nil {
		return
	}

	var result thrifttest.TestVoidResult
	if err = result.FromWire(body); err != nil {
		return
	}

	err = thrifttest.TestVoidHelper.UnwrapResponse(&result)
	return
}
