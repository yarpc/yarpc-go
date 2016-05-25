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
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet"
	"github.com/yarpc/yarpc-go/encoding/thrift"
)

// thriftTest implements the ThriftTest service.
type thriftTest struct{}

func (thriftTest) TestBinary(req *thrift.Request, thing []byte) ([]byte, *thrift.Response, error) {
	return thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestByte(req *thrift.Request, thing *int8) (int8, *thrift.Response, error) {
	return *thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestDouble(req *thrift.Request, thing *float64) (float64, *thrift.Response, error) {
	return *thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestEnum(req *thrift.Request, thing *gauntlet.Numberz) (gauntlet.Numberz, *thrift.Response, error) {
	return *thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestException(req *thrift.Request, arg *string) (*thrift.Response, error) {
	switch *arg {
	case "Xception":
		code := int32(1001)
		return nil, &gauntlet.Xception{ErrorCode: &code, Message: arg}
	case "TException":
		// TODO raise TException once we support it. Meanwhile, fallthrough.
		fallthrough
	default:
		return &thrift.Response{Headers: req.Headers}, nil
	}
}

func (thriftTest) TestI32(req *thrift.Request, thing *int32) (int32, *thrift.Response, error) {
	return *thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestI64(req *thrift.Request, thing *int64) (int64, *thrift.Response, error) {
	return *thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestInsanity(req *thrift.Request, argument *gauntlet.Insanity) (map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, *thrift.Response, error) {
	result := map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity{
		1: {
			gauntlet.NumberzTwo:   argument,
			gauntlet.NumberzThree: argument,
		},
		2: {
			gauntlet.NumberzSix: &gauntlet.Insanity{},
		},
	}
	return result, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestList(req *thrift.Request, thing []int32) ([]int32, *thrift.Response, error) {
	return thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestMap(req *thrift.Request, thing map[int32]int32) (map[int32]int32, *thrift.Response, error) {
	return thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestMapMap(req *thrift.Request, hello *int32) (map[int32]map[int32]int32, *thrift.Response, error) {
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
	return result, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestMulti(req *thrift.Request, arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *gauntlet.Numberz, arg5 *gauntlet.UserId) (*gauntlet.Xtruct, *thrift.Response, error) {
	hello := "Hello2"
	result := &gauntlet.Xtruct{
		StringThing: &hello,
		ByteThing:   arg0,
		I32Thing:    arg1,
		I64Thing:    arg2,
	}
	return result, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestMultiException(req *thrift.Request, arg0 *string, arg1 *string) (*gauntlet.Xtruct, *thrift.Response, error) {
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
		return structThing, &thrift.Response{Headers: req.Headers}, nil
	}
}

func (thriftTest) TestNest(req *thrift.Request, thing *gauntlet.Xtruct2) (*gauntlet.Xtruct2, *thrift.Response, error) {
	return thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestSet(req *thrift.Request, thing map[int32]struct{}) (map[int32]struct{}, *thrift.Response, error) {
	return thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestString(req *thrift.Request, thing *string) (string, *thrift.Response, error) {
	return *thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestStringMap(req *thrift.Request, thing map[string]string) (map[string]string, *thrift.Response, error) {
	return thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestStruct(req *thrift.Request, thing *gauntlet.Xtruct) (*gauntlet.Xtruct, *thrift.Response, error) {
	return thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestTypedef(req *thrift.Request, thing *gauntlet.UserId) (gauntlet.UserId, *thrift.Response, error) {
	return *thing, &thrift.Response{Headers: req.Headers}, nil
}

func (thriftTest) TestVoid(req *thrift.Request) (*thrift.Response, error) {
	return &thrift.Response{Headers: req.Headers}, nil
}
