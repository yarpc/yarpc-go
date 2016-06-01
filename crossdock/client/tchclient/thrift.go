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
	"time"

	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/client/gauntlet"
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

	// TODO assert headers
	headers := map[string]string{
		"hello": "gauntlet",
	}
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
		//    ErrorCode: int32p(1001),
		//    Message:   stringp("Xception"),
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
			Function: "TestString",
			Give:     []interface{}{token},
			Want:     token,
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
		ctx = thrift.WithHeaders(ctx, headers)
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

// TODO implement this
func isUnrecognizedProcedure(err error) bool {
	return false
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

// TODO these are reusable from gauntlet/behavior.go
func bytep(x int8) *int8         { return &x }
func int32p(x int32) *int32      { return &x }
func int64p(x int64) *int64      { return &x }
func doublep(x float64) *float64 { return &x }
func stringp(x string) *string   { return &x }
