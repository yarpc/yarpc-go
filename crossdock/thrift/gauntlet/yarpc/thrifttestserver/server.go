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

package thrifttestserver

import (
	"go.uber.org/thriftrw/protocol"
	"golang.org/x/net/context"
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/service/thrifttest"
	"go.uber.org/thriftrw/wire"
)

// Interface is the server-side interface for the ThriftTest service.
type Interface interface {
	TestBinary(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing []byte,
	) ([]byte, yarpc.ResMeta, error)

	TestByte(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *int8,
	) (int8, yarpc.ResMeta, error)

	TestDouble(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *float64,
	) (float64, yarpc.ResMeta, error)

	TestEnum(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *gauntlet.Numberz,
	) (gauntlet.Numberz, yarpc.ResMeta, error)

	TestException(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Arg *string,
	) (yarpc.ResMeta, error)

	TestI32(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *int32,
	) (int32, yarpc.ResMeta, error)

	TestI64(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *int64,
	) (int64, yarpc.ResMeta, error)

	TestInsanity(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Argument *gauntlet.Insanity,
	) (map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, yarpc.ResMeta, error)

	TestList(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing []int32,
	) ([]int32, yarpc.ResMeta, error)

	TestMap(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing map[int32]int32,
	) (map[int32]int32, yarpc.ResMeta, error)

	TestMapMap(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Hello *int32,
	) (map[int32]map[int32]int32, yarpc.ResMeta, error)

	TestMulti(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Arg0 *int8,
		Arg1 *int32,
		Arg2 *int64,
		Arg3 map[int16]string,
		Arg4 *gauntlet.Numberz,
		Arg5 *gauntlet.UserId,
	) (*gauntlet.Xtruct, yarpc.ResMeta, error)

	TestMultiException(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Arg0 *string,
		Arg1 *string,
	) (*gauntlet.Xtruct, yarpc.ResMeta, error)

	TestNest(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *gauntlet.Xtruct2,
	) (*gauntlet.Xtruct2, yarpc.ResMeta, error)

	TestSet(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing map[int32]struct{},
	) (map[int32]struct{}, yarpc.ResMeta, error)

	TestString(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *string,
	) (string, yarpc.ResMeta, error)

	TestStringMap(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing map[string]string,
	) (map[string]string, yarpc.ResMeta, error)

	TestStruct(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *gauntlet.Xtruct,
	) (*gauntlet.Xtruct, yarpc.ResMeta, error)

	TestTypedef(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Thing *gauntlet.UserId,
	) (gauntlet.UserId, yarpc.ResMeta, error)

	TestVoid(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
	) (yarpc.ResMeta, error)
}

// New prepares an implementation of the ThriftTest service for
// registration.
//
// 	handler := ThriftTestHandler{}
// 	thrift.Register(dispatcher, thrifttestserver.New(handler))
func New(impl Interface) thrift.Service {
	return service{handler{impl}}
}

type service struct{ h handler }

func (service) Name() string {
	return "ThriftTest"
}

func (service) Protocol() protocol.Protocol {
	return protocol.Binary
}

func (s service) Handlers() map[string]thrift.Handler {
	return map[string]thrift.Handler{
		"testBinary": thrift.HandlerFunc(s.h.TestBinary),

		"testByte": thrift.HandlerFunc(s.h.TestByte),

		"testDouble": thrift.HandlerFunc(s.h.TestDouble),

		"testEnum": thrift.HandlerFunc(s.h.TestEnum),

		"testException": thrift.HandlerFunc(s.h.TestException),

		"testI32": thrift.HandlerFunc(s.h.TestI32),

		"testI64": thrift.HandlerFunc(s.h.TestI64),

		"testInsanity": thrift.HandlerFunc(s.h.TestInsanity),

		"testList": thrift.HandlerFunc(s.h.TestList),

		"testMap": thrift.HandlerFunc(s.h.TestMap),

		"testMapMap": thrift.HandlerFunc(s.h.TestMapMap),

		"testMulti": thrift.HandlerFunc(s.h.TestMulti),

		"testMultiException": thrift.HandlerFunc(s.h.TestMultiException),

		"testNest": thrift.HandlerFunc(s.h.TestNest),

		"testSet": thrift.HandlerFunc(s.h.TestSet),

		"testString": thrift.HandlerFunc(s.h.TestString),

		"testStringMap": thrift.HandlerFunc(s.h.TestStringMap),

		"testStruct": thrift.HandlerFunc(s.h.TestStruct),

		"testTypedef": thrift.HandlerFunc(s.h.TestTypedef),

		"testVoid": thrift.HandlerFunc(s.h.TestVoid),
	}
}

type handler struct{ impl Interface }

func (h handler) TestBinary(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestBinaryArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestBinary(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestBinaryHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestByte(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestByteArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestByte(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestByteHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestDouble(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestDoubleArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestDouble(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestDoubleHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestEnum(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestEnumArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestEnum(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestEnumHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestException(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestExceptionArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	resMeta, err := h.impl.TestException(ctx, reqMeta, args.Arg)

	hadError := err != nil
	result, err := thrifttest.TestExceptionHelper.WrapResponse(err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestI32(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestI32Args
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestI32(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestI32Helper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestI64(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestI64Args
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestI64(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestI64Helper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestInsanity(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestInsanityArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestInsanity(ctx, reqMeta, args.Argument)

	hadError := err != nil
	result, err := thrifttest.TestInsanityHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestList(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestListArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestList(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestListHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestMap(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestMapArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestMap(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestMapHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestMapMap(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestMapMapArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestMapMap(ctx, reqMeta, args.Hello)

	hadError := err != nil
	result, err := thrifttest.TestMapMapHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestMulti(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestMultiArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestMulti(ctx, reqMeta, args.Arg0, args.Arg1, args.Arg2, args.Arg3, args.Arg4, args.Arg5)

	hadError := err != nil
	result, err := thrifttest.TestMultiHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestMultiException(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestMultiExceptionArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestMultiException(ctx, reqMeta, args.Arg0, args.Arg1)

	hadError := err != nil
	result, err := thrifttest.TestMultiExceptionHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestNest(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestNestArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestNest(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestNestHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestSet(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestSetArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestSet(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestSetHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestString(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestStringArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestString(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestStringHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestStringMap(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestStringMapArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestStringMap(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestStringMapHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestStruct(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestStructArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestStruct(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestStructHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestTypedef(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestTypedefArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.TestTypedef(ctx, reqMeta, args.Thing)

	hadError := err != nil
	result, err := thrifttest.TestTypedefHelper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}

func (h handler) TestVoid(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args thrifttest.TestVoidArgs
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	resMeta, err := h.impl.TestVoid(ctx, reqMeta)

	hadError := err != nil
	result, err := thrifttest.TestVoidHelper.WrapResponse(err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}
