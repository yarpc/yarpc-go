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

package tchclient

import (
	"reflect"
	"strings"
	"time"

	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/client/gauntlet"
	"github.com/yarpc/yarpc-go/crossdock/client/pointer"
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gen-go/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gen-go/gauntlet_apache"

	"github.com/uber/tchannel-go/thrift"
)

func runThrift(t crossdock.T, call call) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := map[string]string{
		"hello": "thrift",
	}
	token := random.String(5)

	call.Channel.Peers().Add(call.ServerHostPort)

	resp, respHeaders, err := thriftCall(call, headers, token)
	if checks.NoError(err, "thrift: call failed") {
		assert.Equal(token, resp.Boop, "body echoed")
		assert.Equal(headers, respHeaders, "headers echoed")
	}

	runGauntlet(t, call)
}

func thriftCall(call call, headers map[string]string, token string) (*echo.Pong, map[string]string, error) {
	client := echo.NewTChanEchoClient(thrift.NewClient(call.Channel, serverName, nil))

	ctx, cancel := thrift.NewContext(time.Second)
	ctx = thrift.WithHeaders(ctx, headers)
	defer cancel()

	pong, err := client.Echo(ctx, &echo.Ping{Beep: token})
	return pong, ctx.ResponseHeaders(), err
}

func runGauntlet(t crossdock.T, call call) {
	checks := crossdock.Checks(t)

	token := random.String(5)
	bytesToken := random.Bytes(1)

	tests := []gauntlet.TT{
		{
			Function: "TestBinary",
			Give:     []interface{}{bytesToken},
			Want:     bytesToken,
		},
		{
			Function: "TestByte",
			Give:     []interface{}{int8(42)},
			Want:     int8(42),
		},
		{
			Function: "TestDouble",
			Give:     []interface{}{float64(12.34)},
			Want:     float64(12.34),
		},
		{
			Function: "TestEnum",
			Details:  "MyNumberz",
			// TODO is this a thrift-gen bug?
			// Needing to cast to Numberz, since MyNumberz doesnt have a type for some reason,
			// this is odd since gauntlet_apche.Numberz_THREE has a type of Numberz already
			Give: []interface{}{gauntlet_apache.Numberz(gauntlet_apache.MyNumberz)},
			Want: gauntlet_apache.Numberz(gauntlet_apache.MyNumberz),
		},
		{
			Function: "TestEnum",
			Details:  "NumberzThree",
			Give:     []interface{}{gauntlet_apache.Numberz_THREE},
			Want:     gauntlet_apache.Numberz_THREE,
		},
		{
			Function: "TestEnum",
			Details:  "unrecognized Numberz",
			Give:     []interface{}{gauntlet_apache.Numberz(42)},
			Want:     gauntlet_apache.Numberz(42),
		},
		// TODO getting nil instead of Xception right now
		// @see https://github.com/yarpc/yarpc-go/issues/163
		//{
		//  Function: "TestException",
		//  Details:  "Xception",
		//  Give:     []interface{}{"Xception"},
		//  WantError: &gauntlet_apache.Xception{
		//    ErrorCode: pointer.Int32(1001),
		//    Message:   pointer.String("Xception"),
		//  },
		//},
		{
			Function:      "TestException",
			Details:       "TException",
			Give:          []interface{}{"TException"},
			WantErrorLike: `UnexpectedError: error for procedure "ThriftTest::testException" of service "yarpc-test": great sadness`,
		},
		{
			Function: "TestException",
			Details:  "no error",
			Give:     []interface{}{"yolo"},
			// TODO no Want here?
		},
		{
			Function: "TestI32",
			Give:     []interface{}{int32(123)},
			Want:     int32(123),
		},
		{
			Function: "TestI64",
			Give:     []interface{}{int64(18934714)},
			Want:     int64(18934714),
		},
		{
			Function: "TestInsanity",
			Give: []interface{}{
				&gauntlet_apache.Insanity{
					UserMap: map[gauntlet_apache.Numberz]gauntlet_apache.UserId{
						gauntlet_apache.Numberz_THREE: gauntlet_apache.UserId(100),
						gauntlet_apache.Numberz(100):  gauntlet_apache.UserId(200),
					},
					Xtructs: []*gauntlet_apache.Xtruct{
						{StringThing: pointer.String("0")},
						{ByteThing: pointer.Byte(1)},
						{I32Thing: pointer.Int32(2)},
						{I64Thing: pointer.Int64(3)},
					},
				},
			},
			Want: map[gauntlet_apache.UserId]map[gauntlet_apache.Numberz]*gauntlet_apache.Insanity{
				1: {
					gauntlet_apache.Numberz_TWO: &gauntlet_apache.Insanity{
						UserMap: map[gauntlet_apache.Numberz]gauntlet_apache.UserId{
							gauntlet_apache.Numberz_THREE: gauntlet_apache.UserId(100),
							gauntlet_apache.Numberz(100):  gauntlet_apache.UserId(200),
						},
						Xtructs: []*gauntlet_apache.Xtruct{
							{StringThing: pointer.String("0")},
							{ByteThing: pointer.Byte(1)},
							{I32Thing: pointer.Int32(2)},
							{I64Thing: pointer.Int64(3)},
						},
					},
					gauntlet_apache.Numberz_THREE: &gauntlet_apache.Insanity{
						UserMap: map[gauntlet_apache.Numberz]gauntlet_apache.UserId{
							gauntlet_apache.Numberz_THREE: gauntlet_apache.UserId(100),
							gauntlet_apache.Numberz(100):  gauntlet_apache.UserId(200),
						},
						Xtructs: []*gauntlet_apache.Xtruct{
							{StringThing: pointer.String("0")},
							{ByteThing: pointer.Byte(1)},
							{I32Thing: pointer.Int32(2)},
							{I64Thing: pointer.Int64(3)},
						},
					},
				},
				2: {
					gauntlet_apache.Numberz_SIX: &gauntlet_apache.Insanity{},
				},
			},
		},
		{
			Function: "TestList",
			Give:     []interface{}{[]int32{1, 2, 3}},
			Want:     []int32{1, 2, 3},
		},
		{
			Function: "TestMap",
			Give:     []interface{}{map[int32]int32{1: 2, 3: 4, 5: 6}},
			Want:     map[int32]int32{1: 2, 3: 4, 5: 6},
		},
		{
			Function: "TestMapMap",
			Give:     []interface{}{int32(42)},
			Want: map[int32]map[int32]int32{
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
			},
		},
		{
			Function: "TestMulti",
			Give: []interface{}{
				int8(100),
				int32(200),
				int64(300),
				map[int16]string{1: "1", 2: "2", 3: "3"},
				gauntlet_apache.Numberz_EIGHT,
				gauntlet_apache.UserId(42),
			},
			Want: &gauntlet_apache.Xtruct{
				StringThing: pointer.String("Hello2"),
				ByteThing:   pointer.Byte(100),
				I32Thing:    pointer.Int32(200),
				I64Thing:    pointer.Int64(300),
			},
		},
		// TODO TestMultiException Xception & Xception2
		// @see https://github.com/yarpc/yarpc-go/issues/163
		{
			Function: "TestMultiException",
			Details:  "no error",
			Give:     []interface{}{"hello", "foo"},
			Want:     &gauntlet_apache.Xtruct{StringThing: pointer.String("foo")},
		},
		{
			Function: "TestNest",
			Give: []interface{}{
				&gauntlet_apache.Xtruct2{
					ByteThing: pointer.Byte(-1),
					I32Thing:  pointer.Int32(-1234),
					StructThing: &gauntlet_apache.Xtruct{
						StringThing: pointer.String("0"),
						ByteThing:   pointer.Byte(1),
						I32Thing:    pointer.Int32(2),
						I64Thing:    pointer.Int64(3),
					},
				},
			},
			Want: &gauntlet_apache.Xtruct2{
				ByteThing: pointer.Byte(-1),
				I32Thing:  pointer.Int32(-1234),
				StructThing: &gauntlet_apache.Xtruct{
					StringThing: pointer.String("0"),
					ByteThing:   pointer.Byte(1),
					I32Thing:    pointer.Int32(2),
					I64Thing:    pointer.Int64(3),
				},
			},
		},
		{
			Function: "TestSet",
			Give: []interface{}{
				map[int32]bool{
					1:  true,
					2:  true,
					-1: true,
					-2: true,
				},
			},
			Want: map[int32]bool{
				1:  true,
				2:  true,
				-1: true,
				-2: true,
			},
		},
		{
			Function: "TestString",
			Give:     []interface{}{token},
			Want:     token,
		},
		{
			Function: "TestStringMap",
			Give: []interface{}{
				map[string]string{
					"foo":   "bar",
					"hello": "world",
				},
			},
			Want: map[string]string{
				"foo":   "bar",
				"hello": "world",
			},
		},
		{
			Function: "TestStruct",
			Give: []interface{}{
				&gauntlet_apache.Xtruct{
					StringThing: pointer.String("0"),
					ByteThing:   pointer.Byte(1),
					I32Thing:    pointer.Int32(2),
					I64Thing:    pointer.Int64(3),
				},
			},
			Want: &gauntlet_apache.Xtruct{
				StringThing: pointer.String("0"),
				ByteThing:   pointer.Byte(1),
				I32Thing:    pointer.Int32(2),
				I64Thing:    pointer.Int64(3),
			},
		},
		{
			Function: "TestTypedef",
			Give:     []interface{}{gauntlet_apache.UserId(42)},
			Want:     gauntlet_apache.UserId(42),
		},
		{
			Function: "TestVoid",
			Give:     []interface{}{},
		},
		{
			Service:  "SecondService",
			Function: "BlahBlah",
			Give:     []interface{}{},
			// TODO no Want here?
		},
		{
			Service:  "SecondService",
			Function: "SecondtestString",
			Give:     []interface{}{"hello"},
			Want:     "hello",
		},
	}

	for _, tt := range tests {
		desc := gauntlet.BuildDesc(tt)

		client := buildClient(t, desc, tt.Service, thrift.NewClient(call.Channel, serverName, nil))
		f := client.MethodByName(tt.Function)
		if !checks.True(f.IsValid(), "%v: invalid function", desc) {
			continue
		}

		ctx, cancel := thrift.NewContext(time.Second)
		defer cancel()

		args := []reflect.Value{reflect.ValueOf(ctx)}
		if give, ok := gauntlet.BuildArgs(t, desc, f.Type(), tt.Give); ok {
			args = append(args, give...)
		} else {
			continue
		}

		got, err := extractCallResponse(t, desc, f.Call(args))
		if isUnrecognizedProcedure(err) {
			t.Skipf("%v: procedure not defined", desc)
			continue
		}
		gauntlet.Assert(t, tt, desc, got, err)
	}
}

func buildClient(t crossdock.T, desc string, service string, client thrift.TChanClient) reflect.Value {
	switch service {
	case "", "ThriftTest":
		client := gauntlet_apache.NewTChanThriftTestClient(client)
		return reflect.ValueOf(client)
	case "SecondService":
		client := gauntlet_apache.NewTChanSecondServiceClient(client)
		return reflect.ValueOf(client)
	default:
		crossdock.Fatals(t).Fail("", "%v: unknown thrift service", desc)
		return reflect.Value{} // we'll never actually get here
	}
}

// TODO once other servers implement the gauntlet, this func should be removed
func isUnrecognizedProcedure(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "unrecognized procedure")
}

func extractCallResponse(t crossdock.T, desc string, returns []reflect.Value) (interface{}, error) {
	var (
		err error
		got interface{}
	)

	switch len(returns) {
	case 1:
		e := returns[0].Interface()
		if e != nil {
			err = e.(error)
		}
	case 2:
		got = returns[0].Interface()
		e := returns[1].Interface()
		if e != nil {
			err = e.(error)
		}
	default:
		crossdock.Assert(t).Fail("",
			"%v: received unexpected number of return values: %v", desc, returns)
	}

	return got, err
}
