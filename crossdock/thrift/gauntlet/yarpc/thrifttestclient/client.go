// Code generated by thriftrw

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
	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	yarpc "github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/service/thrifttest"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
)

type Interface interface {
	TestBinary(reqMeta yarpc.CallReqMeta, thing []byte) ([]byte, yarpc.CallResMeta, error)
	TestByte(reqMeta yarpc.CallReqMeta, thing *int8) (int8, yarpc.CallResMeta, error)
	TestDouble(reqMeta yarpc.CallReqMeta, thing *float64) (float64, yarpc.CallResMeta, error)
	TestEnum(reqMeta yarpc.CallReqMeta, thing *gauntlet.Numberz) (gauntlet.Numberz, yarpc.CallResMeta, error)
	TestException(reqMeta yarpc.CallReqMeta, arg *string) (yarpc.CallResMeta, error)
	TestI32(reqMeta yarpc.CallReqMeta, thing *int32) (int32, yarpc.CallResMeta, error)
	TestI64(reqMeta yarpc.CallReqMeta, thing *int64) (int64, yarpc.CallResMeta, error)
	TestInsanity(reqMeta yarpc.CallReqMeta, argument *gauntlet.Insanity) (map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, yarpc.CallResMeta, error)
	TestList(reqMeta yarpc.CallReqMeta, thing []int32) ([]int32, yarpc.CallResMeta, error)
	TestMap(reqMeta yarpc.CallReqMeta, thing map[int32]int32) (map[int32]int32, yarpc.CallResMeta, error)
	TestMapMap(reqMeta yarpc.CallReqMeta, hello *int32) (map[int32]map[int32]int32, yarpc.CallResMeta, error)
	TestMulti(reqMeta yarpc.CallReqMeta, arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *gauntlet.Numberz, arg5 *gauntlet.UserId) (*gauntlet.Xtruct, yarpc.CallResMeta, error)
	TestMultiException(reqMeta yarpc.CallReqMeta, arg0 *string, arg1 *string) (*gauntlet.Xtruct, yarpc.CallResMeta, error)
	TestNest(reqMeta yarpc.CallReqMeta, thing *gauntlet.Xtruct2) (*gauntlet.Xtruct2, yarpc.CallResMeta, error)
	TestSet(reqMeta yarpc.CallReqMeta, thing map[int32]struct{}) (map[int32]struct{}, yarpc.CallResMeta, error)
	TestString(reqMeta yarpc.CallReqMeta, thing *string) (string, yarpc.CallResMeta, error)
	TestStringMap(reqMeta yarpc.CallReqMeta, thing map[string]string) (map[string]string, yarpc.CallResMeta, error)
	TestStruct(reqMeta yarpc.CallReqMeta, thing *gauntlet.Xtruct) (*gauntlet.Xtruct, yarpc.CallResMeta, error)
	TestTypedef(reqMeta yarpc.CallReqMeta, thing *gauntlet.UserId) (gauntlet.UserId, yarpc.CallResMeta, error)
	TestVoid(reqMeta yarpc.CallReqMeta) (yarpc.CallResMeta, error)
}

func New(c transport.Channel, opts ...thrift.ClientOption) Interface {
	return client{c: thrift.New(thrift.Config{Service: "ThriftTest", Channel: c, Protocol: protocol.Binary}, opts...)}
}

type client struct{ c thrift.Client }

func (c client) TestBinary(reqMeta yarpc.CallReqMeta, thing []byte) (success []byte, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestBinaryHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestByte(reqMeta yarpc.CallReqMeta, thing *int8) (success int8, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestByteHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestDouble(reqMeta yarpc.CallReqMeta, thing *float64) (success float64, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestDoubleHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestEnum(reqMeta yarpc.CallReqMeta, thing *gauntlet.Numberz) (success gauntlet.Numberz, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestEnumHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestException(reqMeta yarpc.CallReqMeta, arg *string) (resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestExceptionHelper.Args(arg)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestI32(reqMeta yarpc.CallReqMeta, thing *int32) (success int32, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestI32Helper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestI64(reqMeta yarpc.CallReqMeta, thing *int64) (success int64, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestI64Helper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestInsanity(reqMeta yarpc.CallReqMeta, argument *gauntlet.Insanity) (success map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestInsanityHelper.Args(argument)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestList(reqMeta yarpc.CallReqMeta, thing []int32) (success []int32, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestListHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestMap(reqMeta yarpc.CallReqMeta, thing map[int32]int32) (success map[int32]int32, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestMapHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestMapMap(reqMeta yarpc.CallReqMeta, hello *int32) (success map[int32]map[int32]int32, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestMapMapHelper.Args(hello)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestMulti(reqMeta yarpc.CallReqMeta, arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *gauntlet.Numberz, arg5 *gauntlet.UserId) (success *gauntlet.Xtruct, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestMultiHelper.Args(arg0, arg1, arg2, arg3, arg4, arg5)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestMultiException(reqMeta yarpc.CallReqMeta, arg0 *string, arg1 *string) (success *gauntlet.Xtruct, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestMultiExceptionHelper.Args(arg0, arg1)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestNest(reqMeta yarpc.CallReqMeta, thing *gauntlet.Xtruct2) (success *gauntlet.Xtruct2, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestNestHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestSet(reqMeta yarpc.CallReqMeta, thing map[int32]struct{}) (success map[int32]struct{}, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestSetHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestString(reqMeta yarpc.CallReqMeta, thing *string) (success string, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestStringHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestStringMap(reqMeta yarpc.CallReqMeta, thing map[string]string) (success map[string]string, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestStringMapHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestStruct(reqMeta yarpc.CallReqMeta, thing *gauntlet.Xtruct) (success *gauntlet.Xtruct, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestStructHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestTypedef(reqMeta yarpc.CallReqMeta, thing *gauntlet.UserId) (success gauntlet.UserId, resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestTypedefHelper.Args(thing)
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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

func (c client) TestVoid(reqMeta yarpc.CallReqMeta) (resMeta yarpc.CallResMeta, err error) {
	args := thrifttest.TestVoidHelper.Args()
	var body wire.Value
	body, resMeta, err = c.c.Call(reqMeta, args)
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
