// Copyright (c) 2020 Uber Technologies, Inc.
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

	"github.com/crossdock/crossdock-go"
	"github.com/uber/tchannel-go/thrift"
	"go.uber.org/thriftrw/ptr"
	"go.uber.org/yarpc/internal/crossdock/client/gauntlet"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/internal"
	"go.uber.org/yarpc/internal/crossdock/thrift/gen-go/echo"
	"go.uber.org/yarpc/internal/crossdock/thrift/gen-go/gauntlet_tchannel"
)

func runThrift(t crossdock.T, call call) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := map[string]string{
		"hello": "thrift",
	}
	token := random.String(5)

	client := thrift.NewClient(call.Channel, serverName, &thrift.ClientOptions{HostPort: call.ServerHostPort})

	resp, respHeaders, err := thriftCall(client, headers, token)
	if checks.NoError(err, "thrift: call failed") {
		assert.Equal(token, resp.Boop, "body echoed")
		internal.RemoveVariableMapKeys(respHeaders)
		assert.Equal(headers, respHeaders, "headers echoed")
	}

	runGauntlet(t, client)
}

func thriftCall(clientt thrift.TChanClient, headers map[string]string, token string) (*echo.Pong, map[string]string, error) {
	client := echo.NewTChanEchoClient(clientt)

	ctx, cancel := thrift.NewContext(time.Second)
	ctx = thrift.WithHeaders(ctx, headers)
	defer cancel()

	pong, err := client.Echo(ctx, &echo.Ping{Beep: token})
	return pong, ctx.ResponseHeaders(), err
}

func runGauntlet(t crossdock.T, clientt thrift.TChanClient) {
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
			Give:     []interface{}{gauntlet_tchannel.Numberz(gauntlet_tchannel.MyNumberz)},
			Want:     gauntlet_tchannel.Numberz(gauntlet_tchannel.MyNumberz),
		},
		{
			Function: "TestEnum",
			Details:  "NumberzThree",
			Give:     []interface{}{gauntlet_tchannel.Numberz_THREE},
			Want:     gauntlet_tchannel.Numberz_THREE,
		},
		{
			Function: "TestEnum",
			Details:  "unrecognized Numberz",
			Give:     []interface{}{gauntlet_tchannel.Numberz(42)},
			Want:     gauntlet_tchannel.Numberz(42),
		},
		{
			Function: "TestException",
			Details:  "Xception",
			Give:     []interface{}{"Xception"},
			WantError: &gauntlet_tchannel.Xception{
				ErrorCode: ptr.Int32(1001),
				Message:   ptr.String("Xception"),
			},
		},
		{
			Function:      "TestException",
			Details:       "TException",
			Give:          []interface{}{"TException"},
			WantErrorLike: `ErrCodeUnexpected: great sadness`,
		},
		{
			Function: "TestException",
			Details:  "no error",
			Give:     []interface{}{"yolo"},
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
				&gauntlet_tchannel.Insanity{
					UserMap: map[gauntlet_tchannel.Numberz]gauntlet_tchannel.UserId{
						gauntlet_tchannel.Numberz_THREE: gauntlet_tchannel.UserId(100),
						gauntlet_tchannel.Numberz(100):  gauntlet_tchannel.UserId(200),
					},
					Xtructs: []*gauntlet_tchannel.Xtruct{
						{StringThing: ptr.String("0")},
						{ByteThing: ptr.Int8(1)},
						{I32Thing: ptr.Int32(2)},
						{I64Thing: ptr.Int64(3)},
					},
				},
			},
			Want: map[gauntlet_tchannel.UserId]map[gauntlet_tchannel.Numberz]*gauntlet_tchannel.Insanity{
				1: {
					gauntlet_tchannel.Numberz_TWO: &gauntlet_tchannel.Insanity{
						UserMap: map[gauntlet_tchannel.Numberz]gauntlet_tchannel.UserId{
							gauntlet_tchannel.Numberz_THREE: gauntlet_tchannel.UserId(100),
							gauntlet_tchannel.Numberz(100):  gauntlet_tchannel.UserId(200),
						},
						Xtructs: []*gauntlet_tchannel.Xtruct{
							{StringThing: ptr.String("0")},
							{ByteThing: ptr.Int8(1)},
							{I32Thing: ptr.Int32(2)},
							{I64Thing: ptr.Int64(3)},
						},
					},
					gauntlet_tchannel.Numberz_THREE: &gauntlet_tchannel.Insanity{
						UserMap: map[gauntlet_tchannel.Numberz]gauntlet_tchannel.UserId{
							gauntlet_tchannel.Numberz_THREE: gauntlet_tchannel.UserId(100),
							gauntlet_tchannel.Numberz(100):  gauntlet_tchannel.UserId(200),
						},
						Xtructs: []*gauntlet_tchannel.Xtruct{
							{StringThing: ptr.String("0")},
							{ByteThing: ptr.Int8(1)},
							{I32Thing: ptr.Int32(2)},
							{I64Thing: ptr.Int64(3)},
						},
					},
				},
				2: {
					gauntlet_tchannel.Numberz_SIX: &gauntlet_tchannel.Insanity{},
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
				gauntlet_tchannel.Numberz_EIGHT,
				gauntlet_tchannel.UserId(42),
			},
			Want: &gauntlet_tchannel.Xtruct{
				StringThing: ptr.String("Hello2"),
				ByteThing:   ptr.Int8(100),
				I32Thing:    ptr.Int32(200),
				I64Thing:    ptr.Int64(300),
			},
		},
		{
			Function: "TestMultiException",
			Details:  "Xception",
			Give:     []interface{}{"Xception", "foo"},
			WantError: &gauntlet_tchannel.Xception{
				ErrorCode: ptr.Int32(1001),
				Message:   ptr.String("This is an Xception"),
			},
		},
		{
			Function: "TestMultiException",
			Details:  "Xception2",
			Give:     []interface{}{"Xception2", "foo"},
			WantError: &gauntlet_tchannel.Xception2{
				ErrorCode:   ptr.Int32(2002),
				StructThing: &gauntlet_tchannel.Xtruct{StringThing: ptr.String("foo")},
			},
		},
		{
			Function: "TestMultiException",
			Details:  "no error",
			Give:     []interface{}{"hello", "foo"},
			Want:     &gauntlet_tchannel.Xtruct{StringThing: ptr.String("foo")},
		},
		{
			Function: "TestNest",
			Give: []interface{}{
				&gauntlet_tchannel.Xtruct2{
					ByteThing: ptr.Int8(-1),
					I32Thing:  ptr.Int32(-1234),
					StructThing: &gauntlet_tchannel.Xtruct{
						StringThing: ptr.String("0"),
						ByteThing:   ptr.Int8(1),
						I32Thing:    ptr.Int32(2),
						I64Thing:    ptr.Int64(3),
					},
				},
			},
			Want: &gauntlet_tchannel.Xtruct2{
				ByteThing: ptr.Int8(-1),
				I32Thing:  ptr.Int32(-1234),
				StructThing: &gauntlet_tchannel.Xtruct{
					StringThing: ptr.String("0"),
					ByteThing:   ptr.Int8(1),
					I32Thing:    ptr.Int32(2),
					I64Thing:    ptr.Int64(3),
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
				&gauntlet_tchannel.Xtruct{
					StringThing: ptr.String("0"),
					ByteThing:   ptr.Int8(1),
					I32Thing:    ptr.Int32(2),
					I64Thing:    ptr.Int64(3),
				},
			},
			Want: &gauntlet_tchannel.Xtruct{
				StringThing: ptr.String("0"),
				ByteThing:   ptr.Int8(1),
				I32Thing:    ptr.Int32(2),
				I64Thing:    ptr.Int64(3),
			},
		},
		{
			Function: "TestTypedef",
			Give:     []interface{}{gauntlet_tchannel.UserId(42)},
			Want:     gauntlet_tchannel.UserId(42),
		},
		{
			Function: "TestVoid",
			Give:     []interface{}{},
		},
		{
			Service:  "SecondService",
			Function: "BlahBlah",
			Give:     []interface{}{},
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

		client := buildClient(t, desc, tt.Service, clientt)
		f := client.MethodByName(tt.Function)
		if !checks.True(f.IsValid(), "%v: invalid function", desc) {
			continue
		}

		ctx, cancel := thrift.NewContext(time.Second)
		defer cancel()

		args := []reflect.Value{reflect.ValueOf(ctx)}
		if give, ok := gauntlet.BuildArgs(t, desc, f.Type(), tt.Give, 1); ok {
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
		client := gauntlet_tchannel.NewTChanThriftTestClient(client)
		return reflect.ValueOf(client)
	case "SecondService":
		client := gauntlet_tchannel.NewTChanSecondServiceClient(client)
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

func extractCallResponse(t crossdock.T, desc string, returns []reflect.Value) (got interface{}, err error) {
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
