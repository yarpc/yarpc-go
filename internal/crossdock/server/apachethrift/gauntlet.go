// Copyright (c) 2021 Uber Technologies, Inc.
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

package apachethrift

import (
	"errors"

	"go.uber.org/yarpc/internal/crossdock/thrift/gen-go/gauntlet_apache"
)

type thriftTestHandler struct{}

func (thriftTestHandler) TestVoid() (err error) {
	return nil
}

func (thriftTestHandler) TestString(thing string) (r string, err error) {
	return thing, nil
}

func (thriftTestHandler) TestByte(thing int8) (r int8, err error) {
	return thing, nil
}

func (thriftTestHandler) TestI32(thing int32) (r int32, err error) {
	return thing, nil
}

func (thriftTestHandler) TestI64(thing int64) (r int64, err error) {
	return thing, nil
}

func (thriftTestHandler) TestDouble(thing float64) (r float64, err error) {
	return thing, nil
}

func (thriftTestHandler) TestBinary(thing []byte) (r []byte, err error) {
	return thing, nil
}

func (thriftTestHandler) TestStruct(thing *gauntlet_apache.Xtruct) (r *gauntlet_apache.Xtruct, err error) {
	return thing, nil
}

func (thriftTestHandler) TestNest(thing *gauntlet_apache.Xtruct2) (r *gauntlet_apache.Xtruct2, err error) {
	return thing, nil
}

func (thriftTestHandler) TestMap(thing map[int32]int32) (r map[int32]int32, err error) {
	return thing, nil
}

func (thriftTestHandler) TestStringMap(thing map[string]string) (r map[string]string, err error) {
	return thing, nil
}

func (thriftTestHandler) TestSet(thing map[int32]bool) (r map[int32]bool, err error) {
	return thing, nil
}

func (thriftTestHandler) TestList(thing []int32) (r []int32, err error) {
	return thing, nil
}

func (thriftTestHandler) TestEnum(thing gauntlet_apache.Numberz) (r gauntlet_apache.Numberz, err error) {
	return thing, nil
}

func (thriftTestHandler) TestTypedef(thing gauntlet_apache.UserId) (r gauntlet_apache.UserId, err error) {
	return thing, nil
}

func (thriftTestHandler) TestOneway(seconds int32) (err error) {
	// time.Sleep(time.Duration(seconds) * time.Second)
	return nil
}

func (thriftTestHandler) TestMapMap(hello int32) (r map[int32]map[int32]int32, err error) {
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

func (thriftTestHandler) TestInsanity(argument *gauntlet_apache.Insanity) (r map[gauntlet_apache.UserId]map[gauntlet_apache.Numberz]*gauntlet_apache.Insanity, err error) {
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

func (thriftTestHandler) TestMulti(arg0 int8, arg1 int32, arg2 int64, arg3 map[int16]string, arg4 gauntlet_apache.Numberz, arg5 gauntlet_apache.UserId) (r *gauntlet_apache.Xtruct, err error) {
	hello := "Hello2"
	result := &gauntlet_apache.Xtruct{
		StringThing: &hello,
		ByteThing:   &arg0,
		I32Thing:    &arg1,
		I64Thing:    &arg2,
	}
	return result, nil
}

func (thriftTestHandler) TestException(arg string) (err error) {
	switch arg {
	case "Xception":
		code := int32(1001)
		return &gauntlet_apache.Xception{ErrorCode: &code, Message: &arg}
	case "TException":
		// unexpected exception.
		return errors.New("great sadness")
	default:
		return nil
	}
}

func (thriftTestHandler) TestMultiException(arg0 string, arg1 string) (r *gauntlet_apache.Xtruct, err error) {
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

type secondServiceHandler struct{}

func (secondServiceHandler) BlahBlah() (err error) {
	return nil
}

func (secondServiceHandler) SecondtestString(thing string) (r string, err error) {
	return thing, nil
}
