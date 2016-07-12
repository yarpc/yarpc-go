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

package gauntlet

import (
	"reflect"
	"strings"
	"time"

	"github.com/yarpc/yarpc-go"
	disp "github.com/yarpc/yarpc-go/crossdock/client/dispatcher"
	"github.com/yarpc/yarpc-go/crossdock/client/params"
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/yarpc/secondserviceclient"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/yarpc/thrifttestclient"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/crossdock/crossdock-go"
	"github.com/thriftrw/thriftrw-go/ptr"
	"golang.org/x/net/context"
)

const serverName = "yarpc-test"

func createGauntletT(t crossdock.T) crossdock.T {
	t.Tag("transport", t.Param(params.Transport))
	t.Tag("server", t.Param(params.Server))
	return t
}

// TT is the gauntlets table test struct
type TT struct {
	Service  string        // thrift service name; defaults to ThriftTest
	Function string        // name of the Go function on the client
	Details  string        // optional extra details about what this test does
	Give     []interface{} // arguments besides ReqMeta

	Want          interface{} // expected response; nil for void
	WantError     error       // expected error
	WantErrorLike string      // for just matching error messages
}

// Run executes the thriftgauntlet behavior.
func Run(t crossdock.T) {
	fatals := crossdock.Fatals(t)

	dispatcher := disp.Create(t)
	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	RunGauntlet(t, dispatcher, serverName)
}

// RunGauntlet takes an rpc object and runs the gauntlet
func RunGauntlet(t crossdock.T, dispatcher yarpc.Dispatcher, serverName string) {
	t = createGauntletT(t)
	checks := crossdock.Checks(t)

	bytesToken := random.Bytes(10)
	tests := []TT{
		{
			Function: "TestBinary",
			Give:     []interface{}{bytesToken},
			Want:     bytesToken,
		},
		{
			Function: "TestByte",
			Give:     []interface{}{ptr.Int8(42)},
			Want:     int8(42),
		},
		{
			Function: "TestDouble",
			Give:     []interface{}{ptr.Float64(12.34)},
			Want:     float64(12.34),
		},
		{
			Function: "TestEnum",
			Details:  "MyNumberz",
			Give:     []interface{}{numberzp(gauntlet.MyNumberz)},
			Want:     gauntlet.MyNumberz,
		},
		{
			Function: "TestEnum",
			Details:  "NumberzThree",
			Give:     []interface{}{numberzp(gauntlet.NumberzThree)},
			Want:     gauntlet.NumberzThree,
		},
		{
			Function: "TestEnum",
			Details:  "unrecognized Numberz",
			Give:     []interface{}{numberzp(gauntlet.Numberz(42))},
			Want:     gauntlet.Numberz(42),
		},
		{
			Function: "TestException",
			Details:  "Xception",
			Give:     []interface{}{ptr.String("Xception")},
			WantError: &gauntlet.Xception{
				ErrorCode: ptr.Int32(1001),
				Message:   ptr.String("Xception"),
			},
		},
		{
			Function:      "TestException",
			Details:       "TException",
			Give:          []interface{}{ptr.String("TException")},
			WantErrorLike: "great sadness",
		},
		{
			Function: "TestException",
			Details:  "no error",
			Give:     []interface{}{ptr.String("yolo")},
		},
		{
			Function: "TestI32",
			Give:     []interface{}{ptr.Int32(123)},
			Want:     int32(123),
		},
		{
			Function: "TestI64",
			Give:     []interface{}{ptr.Int64(18934714)},
			Want:     int64(18934714),
		},
		{
			Function: "TestInsanity",
			Give: []interface{}{
				&gauntlet.Insanity{
					UserMap: map[gauntlet.Numberz]gauntlet.UserId{
						gauntlet.NumberzThree: gauntlet.UserId(100),
						gauntlet.Numberz(100): gauntlet.UserId(200),
					},
					Xtructs: []*gauntlet.Xtruct{
						{StringThing: ptr.String("0")},
						{ByteThing: ptr.Int8(1)},
						{I32Thing: ptr.Int32(2)},
						{I64Thing: ptr.Int64(3)},
					},
				},
			},
			Want: map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity{
				1: {
					gauntlet.NumberzTwo: &gauntlet.Insanity{
						UserMap: map[gauntlet.Numberz]gauntlet.UserId{
							gauntlet.NumberzThree: gauntlet.UserId(100),
							gauntlet.Numberz(100): gauntlet.UserId(200),
						},
						Xtructs: []*gauntlet.Xtruct{
							{StringThing: ptr.String("0")},
							{ByteThing: ptr.Int8(1)},
							{I32Thing: ptr.Int32(2)},
							{I64Thing: ptr.Int64(3)},
						},
					},
					gauntlet.NumberzThree: &gauntlet.Insanity{
						UserMap: map[gauntlet.Numberz]gauntlet.UserId{
							gauntlet.NumberzThree: gauntlet.UserId(100),
							gauntlet.Numberz(100): gauntlet.UserId(200),
						},
						Xtructs: []*gauntlet.Xtruct{
							{StringThing: ptr.String("0")},
							{ByteThing: ptr.Int8(1)},
							{I32Thing: ptr.Int32(2)},
							{I64Thing: ptr.Int64(3)},
						},
					},
				},
				2: {
					gauntlet.NumberzSix: &gauntlet.Insanity{},
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
			Give:     []interface{}{ptr.Int32(42)},
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
				ptr.Int8(100),
				ptr.Int32(200),
				ptr.Int64(300),
				map[int16]string{1: "1", 2: "2", 3: "3"},
				numberzp(gauntlet.NumberzEight),
				useridp(42),
			},
			Want: &gauntlet.Xtruct{
				StringThing: ptr.String("Hello2"),
				ByteThing:   ptr.Int8(100),
				I32Thing:    ptr.Int32(200),
				I64Thing:    ptr.Int64(300),
			},
		},
		{
			Function: "TestMultiException",
			Details:  "Xception",
			Give:     []interface{}{ptr.String("Xception"), ptr.String("foo")},
			WantError: &gauntlet.Xception{
				ErrorCode: ptr.Int32(1001),
				Message:   ptr.String("This is an Xception"),
			},
		},
		{
			Function: "TestMultiException",
			Details:  "Xception2",
			Give:     []interface{}{ptr.String("Xception2"), ptr.String("foo")},
			WantError: &gauntlet.Xception2{
				ErrorCode:   ptr.Int32(2002),
				StructThing: &gauntlet.Xtruct{StringThing: ptr.String("foo")},
			},
		},
		{
			Function: "TestMultiException",
			Details:  "no error",
			Give:     []interface{}{ptr.String("hello"), ptr.String("foo")},
			Want:     &gauntlet.Xtruct{StringThing: ptr.String("foo")},
		},
		{
			Function: "TestNest",
			Give: []interface{}{
				&gauntlet.Xtruct2{
					ByteThing: ptr.Int8(-1),
					I32Thing:  ptr.Int32(-1234),
					StructThing: &gauntlet.Xtruct{
						StringThing: ptr.String("0"),
						ByteThing:   ptr.Int8(1),
						I32Thing:    ptr.Int32(2),
						I64Thing:    ptr.Int64(3),
					},
				},
			},
			Want: &gauntlet.Xtruct2{
				ByteThing: ptr.Int8(-1),
				I32Thing:  ptr.Int32(-1234),
				StructThing: &gauntlet.Xtruct{
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
				map[int32]struct{}{
					1:  {},
					2:  {},
					-1: {},
					-2: {},
				},
			},
			Want: map[int32]struct{}{
				1:  {},
				2:  {},
				-1: {},
				-2: {},
			},
		},
		{
			Function: "TestString",
			Give:     []interface{}{ptr.String("hello")},
			Want:     "hello",
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
				&gauntlet.Xtruct{
					StringThing: ptr.String("0"),
					ByteThing:   ptr.Int8(1),
					I32Thing:    ptr.Int32(2),
					I64Thing:    ptr.Int64(3),
				},
			},
			Want: &gauntlet.Xtruct{
				StringThing: ptr.String("0"),
				ByteThing:   ptr.Int8(1),
				I32Thing:    ptr.Int32(2),
				I64Thing:    ptr.Int64(3),
			},
		},
		{
			Function: "TestTypedef",
			Give:     []interface{}{useridp(42)},
			Want:     gauntlet.UserId(42),
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
			Give:     []interface{}{ptr.String("hello")},
			Want:     "hello",
		},
	}

	for _, tt := range tests {
		t.Tag("service", tt.Service)
		t.Tag("function", tt.Function)

		desc := BuildDesc(tt)

		client := buildClient(t, desc, tt.Service, dispatcher.Channel(serverName))
		f := client.MethodByName(tt.Function)
		if !checks.True(f.IsValid(), "%v: invalid function", desc) {
			continue
		}

		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		args := []reflect.Value{reflect.ValueOf(yarpc.NewReqMeta(ctx))}
		if give, ok := BuildArgs(t, desc, f.Type(), tt.Give); ok {
			args = append(args, give...)
		} else {
			continue
		}

		got, err := extractCallResponse(t, desc, f.Call(args))
		if isUnrecognizedProcedure(err) {
			t.Skipf("%v: procedure not defined", desc)
			continue
		}
		Assert(t, tt, desc, got, err)
	}
}

// BuildDesc creates a logging string for the test
//
// We log in one of the following formats,
//
// $Function: $message
// $Function: $description: $message
// $Service: $function: $message
// $Service: $function: $description: $message
//
func BuildDesc(tt TT) string {
	desc := tt.Function
	if tt.Details != "" {
		desc = desc + ": " + tt.Details
	}
	if tt.Service != "" {
		desc = tt.Service + ": " + desc
	}
	return desc
}

func buildClient(t crossdock.T, desc string, service string, channel transport.Channel) reflect.Value {
	switch service {
	case "", "ThriftTest":
		client := thrifttestclient.New(channel)
		return reflect.ValueOf(client)
	case "SecondService":
		client := secondserviceclient.New(channel)
		return reflect.ValueOf(client)
	default:
		crossdock.Fatals(t).Fail("", "%v: unknown thrift service", desc)
		return reflect.Value{} // we'll never actually get here
	}
}

// BuildArgs creates an args slice than can be used to make a f.Call(args)
func BuildArgs(t crossdock.T, desc string, ft reflect.Type, give []interface{}) (_ []reflect.Value, ok bool) {
	check := crossdock.Checks(t)
	wantIn := len(give) + 1
	if !check.Equal(wantIn, ft.NumIn(), "%v: should accept %d arguments", desc, wantIn) {
		return nil, false
	}

	var args []reflect.Value
	for i, v := range give {
		var val reflect.Value
		vt := ft.In(i + 1)
		if v == nil {
			// nil is an invalid argument to ValueOf. For nil, use the zero
			// value for that argument.
			val = reflect.Zero(vt)
		} else {
			val = reflect.ValueOf(v)
		}
		if !check.Equal(vt, val.Type(), "%v: argument %v type mismatch", desc, i) {
			return nil, false
		}
		args = append(args, val)
	}

	return args, true
}

func isUnrecognizedProcedure(err error) bool {
	if transport.IsBadRequestError(err) {
		// TODO: Once all other languages implement the gauntlet test
		// subject, we can remove this check.
		return strings.Contains(err.Error(), "unrecognized procedure")
	}
	return false
}

func extractCallResponse(t crossdock.T, desc string, returns []reflect.Value) (got interface{}, err error) {
	switch len(returns) {
	case 2:
		e := returns[1].Interface()
		if e != nil {
			err = e.(error)
		}
	case 3:
		got = returns[0].Interface()
		e := returns[2].Interface()
		if e != nil {
			err = e.(error)
		}
	default:
		crossdock.Assert(t).Fail("",
			"%v: received unexpected number of return values: %v", desc, returns)
	}
	return got, err
}

// Assert verifies the call response against TT
func Assert(t crossdock.T, tt TT, desc string, got interface{}, err error) {
	checks := crossdock.Checks(t)
	assert := crossdock.Assert(t)

	if tt.WantError != nil || tt.WantErrorLike != "" {
		if !checks.Error(err, "%v: expected failure but got: %v", desc, got) {
			return
		}
		if tt.WantError != nil {
			assert.Equal(tt.WantError, err, "%v: server returned error: %v", desc, err)
		}
		if tt.WantErrorLike != "" {
			assert.Contains(err.Error(), tt.WantErrorLike, "%v: server returned error: %v", desc, err)
		}
	} else {
		if !checks.NoError(err, "%v: call failed", desc) {
			return
		}
		if tt.Want != nil {
			assert.Equal(tt.Want, got, "%v: server returned: %v", desc, got)
		}
	}
}

func numberzp(x gauntlet.Numberz) *gauntlet.Numberz { return &x }
func useridp(x gauntlet.UserId) *gauntlet.UserId    { return &x }
