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

package yarpc

import (
	"errors"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet"
)

func resMetaFromReqMeta(reqMeta yarpc.ReqMeta) yarpc.ResMeta {
	return yarpc.NewResMeta(reqMeta.Context()).Headers(reqMeta.Headers())
}

// thriftTest implements the ThriftTest service.
type thriftTest struct{}

func (thriftTest) TestBinary(reqMeta yarpc.ReqMeta, thing []byte) ([]byte, yarpc.ResMeta, error) {
	return thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestByte(reqMeta yarpc.ReqMeta, thing *int8) (int8, yarpc.ResMeta, error) {
	return *thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestDouble(reqMeta yarpc.ReqMeta, thing *float64) (float64, yarpc.ResMeta, error) {
	return *thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestEnum(reqMeta yarpc.ReqMeta, thing *gauntlet.Numberz) (gauntlet.Numberz, yarpc.ResMeta, error) {
	return *thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestException(reqMeta yarpc.ReqMeta, arg *string) (yarpc.ResMeta, error) {
	switch *arg {
	case "Xception":
		code := int32(1001)
		return nil, &gauntlet.Xception{ErrorCode: &code, Message: arg}
	case "TException":
		// TODO raise TException once we support it. Meanwhile, return an
		// unexpected exception.
		return nil, errors.New("great sadness")
	default:
		return resMetaFromReqMeta(reqMeta), nil
	}
}

func (thriftTest) TestI32(reqMeta yarpc.ReqMeta, thing *int32) (int32, yarpc.ResMeta, error) {
	return *thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestI64(reqMeta yarpc.ReqMeta, thing *int64) (int64, yarpc.ResMeta, error) {
	return *thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestInsanity(reqMeta yarpc.ReqMeta, argument *gauntlet.Insanity) (map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, yarpc.ResMeta, error) {
	result := map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity{
		1: {
			gauntlet.NumberzTwo:   argument,
			gauntlet.NumberzThree: argument,
		},
		2: {
			gauntlet.NumberzSix: &gauntlet.Insanity{},
		},
	}
	return result, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestList(reqMeta yarpc.ReqMeta, thing []int32) ([]int32, yarpc.ResMeta, error) {
	return thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestMap(reqMeta yarpc.ReqMeta, thing map[int32]int32) (map[int32]int32, yarpc.ResMeta, error) {
	return thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestMapMap(reqMeta yarpc.ReqMeta, hello *int32) (map[int32]map[int32]int32, yarpc.ResMeta, error) {
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
	return result, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestMulti(reqMeta yarpc.ReqMeta, arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *gauntlet.Numberz, arg5 *gauntlet.UserId) (*gauntlet.Xtruct, yarpc.ResMeta, error) {
	hello := "Hello2"
	result := &gauntlet.Xtruct{
		StringThing: &hello,
		ByteThing:   arg0,
		I32Thing:    arg1,
		I64Thing:    arg2,
	}
	return result, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestMultiException(reqMeta yarpc.ReqMeta, arg0 *string, arg1 *string) (*gauntlet.Xtruct, yarpc.ResMeta, error) {
	structThing := &gauntlet.Xtruct{StringThing: arg1}
	switch *arg0 {
	case "Xception":
		code := int32(1001)
		message := "This is an Xception"
		return nil, nil, &gauntlet.Xception{ErrorCode: &code, Message: &message}
	case "Xception2":
		code := int32(2002)
		return nil, nil, &gauntlet.Xception2{
			ErrorCode:   &code,
			StructThing: structThing,
		}
	default:
		return structThing, resMetaFromReqMeta(reqMeta), nil
	}
}

func (thriftTest) TestNest(reqMeta yarpc.ReqMeta, thing *gauntlet.Xtruct2) (*gauntlet.Xtruct2, yarpc.ResMeta, error) {
	return thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestSet(reqMeta yarpc.ReqMeta, thing map[int32]struct{}) (map[int32]struct{}, yarpc.ResMeta, error) {
	return thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestString(reqMeta yarpc.ReqMeta, thing *string) (string, yarpc.ResMeta, error) {
	return *thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestStringMap(reqMeta yarpc.ReqMeta, thing map[string]string) (map[string]string, yarpc.ResMeta, error) {
	return thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestStruct(reqMeta yarpc.ReqMeta, thing *gauntlet.Xtruct) (*gauntlet.Xtruct, yarpc.ResMeta, error) {
	return thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestTypedef(reqMeta yarpc.ReqMeta, thing *gauntlet.UserId) (gauntlet.UserId, yarpc.ResMeta, error) {
	return *thing, resMetaFromReqMeta(reqMeta), nil
}

func (thriftTest) TestVoid(reqMeta yarpc.ReqMeta) (yarpc.ResMeta, error) {
	return resMetaFromReqMeta(reqMeta), nil
}
