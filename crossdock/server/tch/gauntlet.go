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

package tch

import (
	"errors"

	"github.com/uber/tchannel-go/thrift"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gen-go/gauntlet_apache"
)

type thriftTestHandler struct{}

func (thriftTestHandler) TestVoid(ctx thrift.Context) error {
	ctx.SetResponseHeaders(ctx.Headers())
	return nil
}

func (thriftTestHandler) TestString(ctx thrift.Context, thing string) (string, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestByte(ctx thrift.Context, thing int8) (int8, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestI32(ctx thrift.Context, thing int32) (int32, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestI64(ctx thrift.Context, thing int64) (int64, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestDouble(ctx thrift.Context, thing float64) (float64, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestBinary(ctx thrift.Context, thing []byte) ([]byte, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestStruct(ctx thrift.Context, thing *gauntlet_apache.Xtruct) (*gauntlet_apache.Xtruct, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestNest(ctx thrift.Context, thing *gauntlet_apache.Xtruct2) (*gauntlet_apache.Xtruct2, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestMap(ctx thrift.Context, thing map[int32]int32) (map[int32]int32, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestStringMap(ctx thrift.Context, thing map[string]string) (map[string]string, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestSet(ctx thrift.Context, thing map[int32]bool) (map[int32]bool, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestList(ctx thrift.Context, thing []int32) ([]int32, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestEnum(ctx thrift.Context, thing gauntlet_apache.Numberz) (gauntlet_apache.Numberz, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestTypedef(ctx thrift.Context, thing gauntlet_apache.UserId) (gauntlet_apache.UserId, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return thing, nil
}

func (thriftTestHandler) TestMapMap(ctx thrift.Context, hello int32) (map[int32]map[int32]int32, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	result := map[int32]map[int32]int32{
		-4: {
			-4: -4,
			-3: -3,
			-2: -2,
			-1: -1,
		},
		4: {
			1: 1,
			2: 2,
			3: 3,
			4: 4,
		},
	}
	return result, nil
}

func (thriftTestHandler) TestInsanity(ctx thrift.Context, argument *gauntlet_apache.Insanity) (
	map[gauntlet_apache.UserId]map[gauntlet_apache.Numberz]*gauntlet_apache.Insanity, error) {

	ctx.SetResponseHeaders(ctx.Headers())
	result := map[gauntlet_apache.UserId]map[gauntlet_apache.Numberz]*gauntlet_apache.Insanity{
		1: {
			gauntlet_apache.Numberz_TWO:   argument,
			gauntlet_apache.Numberz_THREE: argument,
		},
		2: {
			gauntlet_apache.Numberz_SIX: &gauntlet_apache.Insanity{},
		},
	}
	return result, nil
}

func (thriftTestHandler) TestMulti(
	ctx thrift.Context,
	arg0 int8,
	arg1 int32,
	arg2 int64,
	arg3 map[int16]string,
	arg4 gauntlet_apache.Numberz,
	arg5 gauntlet_apache.UserId,
) (*gauntlet_apache.Xtruct, error) {

	ctx.SetResponseHeaders(ctx.Headers())
	hello := "Hello2"
	result := &gauntlet_apache.Xtruct{
		StringThing: &hello,
		ByteThing:   &arg0,
		I32Thing:    &arg1,
		I64Thing:    &arg2,
	}
	return result, nil
}

func (thriftTestHandler) TestException(ctx thrift.Context, arg string) error {
	ctx.SetResponseHeaders(ctx.Headers())
	switch arg {
	case "Xception":
		code := int32(1001)
		return &gauntlet_apache.Xception{ErrorCode: &code, Message: &arg}
	case "TException":
		// TODO is there something better I can raise here?
		// unexpected exception.
		return errors.New("great sadness")
	default:
		return nil
	}
}

func (thriftTestHandler) TestMultiException(ctx thrift.Context, arg0 string, arg1 string) (*gauntlet_apache.Xtruct, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	structThing := &gauntlet_apache.Xtruct{StringThing: &arg1}
	switch arg0 {
	case "Xception":
		code := int32(1001)
		message := "This is an Xception"
		return nil, &gauntlet_apache.Xception{ErrorCode: &code, Message: &message}
	case "Xception2":
		code := int32(2002)
		return nil, &gauntlet_apache.Xception2{ErrorCode: &code, StructThing: structThing}
	default:
		return structThing, nil
	}
}
