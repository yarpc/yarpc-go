// Copyright (c) 2019 Uber Technologies, Inc.
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
	"context"
	"errors"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/crossdock/thrift/gauntlet"
)

func copyRequestHeaders(ctx context.Context) {
	call := yarpc.CallFromContext(ctx)
	for _, k := range call.HeaderNames() {
		if err := call.WriteResponseHeader(k, call.Header(k)); err != nil {
			panic(err)
		}
	}
}

// thriftTest implements the ThriftTest service.
type thriftTest struct{}

func (thriftTest) TestBinary(ctx context.Context, thing []byte) ([]byte, error) {
	copyRequestHeaders(ctx)
	return thing, nil
}

func (thriftTest) TestByte(ctx context.Context, thing *int8) (int8, error) {
	copyRequestHeaders(ctx)
	return *thing, nil
}

func (thriftTest) TestDouble(ctx context.Context, thing *float64) (float64, error) {
	copyRequestHeaders(ctx)
	return *thing, nil
}

func (thriftTest) TestEnum(ctx context.Context, thing *gauntlet.Numberz) (gauntlet.Numberz, error) {
	copyRequestHeaders(ctx)
	return *thing, nil
}

func (thriftTest) TestException(ctx context.Context, arg *string) error {
	switch *arg {
	case "Xception":
		code := int32(1001)
		return &gauntlet.Xception{ErrorCode: &code, Message: arg}
	case "TException":
		// TODO raise TException once we support it. Meanwhile, return an
		// unexpected exception.
		return errors.New("great sadness")
	default:
		copyRequestHeaders(ctx)
		return nil
	}
}

func (thriftTest) TestI32(ctx context.Context, thing *int32) (int32, error) {
	copyRequestHeaders(ctx)
	return *thing, nil
}

func (thriftTest) TestI64(ctx context.Context, thing *int64) (int64, error) {
	copyRequestHeaders(ctx)
	return *thing, nil
}

func (thriftTest) TestInsanity(ctx context.Context, argument *gauntlet.Insanity) (map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity, error) {
	result := map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity{
		1: {
			gauntlet.NumberzTwo:   argument,
			gauntlet.NumberzThree: argument,
		},
		2: {
			gauntlet.NumberzSix: &gauntlet.Insanity{},
		},
	}
	copyRequestHeaders(ctx)
	return result, nil
}

func (thriftTest) TestList(ctx context.Context, thing []int32) ([]int32, error) {
	copyRequestHeaders(ctx)
	return thing, nil
}

func (thriftTest) TestMap(ctx context.Context, thing map[int32]int32) (map[int32]int32, error) {
	copyRequestHeaders(ctx)
	return thing, nil
}

func (thriftTest) TestMapMap(ctx context.Context, hello *int32) (map[int32]map[int32]int32, error) {
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
	copyRequestHeaders(ctx)
	return result, nil
}

func (thriftTest) TestMulti(ctx context.Context, arg0 *int8, arg1 *int32, arg2 *int64, arg3 map[int16]string, arg4 *gauntlet.Numberz, arg5 *gauntlet.UserId) (*gauntlet.Xtruct, error) {
	hello := "Hello2"
	result := &gauntlet.Xtruct{
		StringThing: &hello,
		ByteThing:   arg0,
		I32Thing:    arg1,
		I64Thing:    arg2,
	}
	copyRequestHeaders(ctx)
	return result, nil
}

func (thriftTest) TestMultiException(ctx context.Context, arg0 *string, arg1 *string) (*gauntlet.Xtruct, error) {
	structThing := &gauntlet.Xtruct{StringThing: arg1}
	switch *arg0 {
	case "Xception":
		code := int32(1001)
		message := "This is an Xception"
		return nil, &gauntlet.Xception{ErrorCode: &code, Message: &message}
	case "Xception2":
		code := int32(2002)
		return nil, &gauntlet.Xception2{
			ErrorCode:   &code,
			StructThing: structThing,
		}
	default:
		copyRequestHeaders(ctx)
		return structThing, nil
	}
}

func (thriftTest) TestNest(ctx context.Context, thing *gauntlet.Xtruct2) (*gauntlet.Xtruct2, error) {
	copyRequestHeaders(ctx)
	return thing, nil
}

func (thriftTest) TestOneway(ctx context.Context, seconds *int32) error {
	time.Sleep(time.Duration(*seconds) * time.Second)
	return nil
}

func (thriftTest) TestSet(ctx context.Context, thing map[int32]struct{}) (map[int32]struct{}, error) {
	copyRequestHeaders(ctx)
	return thing, nil
}

func (thriftTest) TestString(ctx context.Context, thing *string) (string, error) {
	copyRequestHeaders(ctx)
	return *thing, nil
}

func (thriftTest) TestStringMap(ctx context.Context, thing map[string]string) (map[string]string, error) {
	copyRequestHeaders(ctx)
	return thing, nil
}

func (thriftTest) TestStruct(ctx context.Context, thing *gauntlet.Xtruct) (*gauntlet.Xtruct, error) {
	copyRequestHeaders(ctx)
	return thing, nil
}

func (thriftTest) TestTypedef(ctx context.Context, thing *gauntlet.UserId) (gauntlet.UserId, error) {
	copyRequestHeaders(ctx)
	return *thing, nil
}

func (thriftTest) TestVoid(ctx context.Context) error {
	copyRequestHeaders(ctx)
	return nil
}
