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

	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/behavior/params"
	"github.com/yarpc/yarpc-go/crossdock/behavior/random"
	"github.com/yarpc/yarpc-go/crossdock/behavior/rpc"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/yarpc/secondserviceclient"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gauntlet/yarpc/thrifttestclient"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

type gauntletEntry struct {
	crossdock.Entry

	Transport string `json:"transport"`
	Server    string `json:"server"`
}

type gauntletSink struct {
	crossdock.Sink

	Transport string
	Server    string
}

func (s gauntletSink) Put(e interface{}) {
	s.Sink.Put(gauntletEntry{
		Entry:     e.(crossdock.Entry),
		Transport: s.Transport,
		Server:    s.Server,
	})
}

func createGauntletSink(s crossdock.Sink, ps crossdock.Params) crossdock.Sink {
	return gauntletSink{
		Sink:      s,
		Transport: ps.Param(params.Transport),
		Server:    ps.Param(params.Server),
	}
}

// Run executes the thriftgauntlet behavior.
func Run(s crossdock.Sink, ps crossdock.Params) {
	s = createGauntletSink(s, ps)
	assert := crossdock.Assert(s)
	checks := crossdock.Checks(s)

	rpc := rpc.Create(s, ps)

	bytesToken := random.Bytes(10)
	tests := []struct {
		service  string        // thrift service name; defaults to ThriftTest
		function string        // name of the Go function on the client
		details  string        // optional extra details about what this test does
		give     []interface{} // arguments besides thrift.Request

		want          interface{} // expected response; nil for void
		wantError     error       // expected error
		wantErrorLike string      // for just matching error messages
	}{
		{
			function: "TestBinary",
			give:     []interface{}{bytesToken},
			want:     bytesToken,
		},
		{
			function: "TestByte",
			give:     []interface{}{bytep(42)},
			want:     int8(42),
		},
		{
			function: "TestDouble",
			give:     []interface{}{doublep(12.34)},
			want:     float64(12.34),
		},
		{
			function: "TestEnum",
			details:  "MyNumberz",
			give:     []interface{}{numberzp(gauntlet.MyNumberz)},
			want:     gauntlet.MyNumberz,
		},
		{
			function: "TestEnum",
			details:  "NumberzThree",
			give:     []interface{}{numberzp(gauntlet.NumberzThree)},
			want:     gauntlet.NumberzThree,
		},
		{
			function: "TestEnum",
			details:  "unrecognized Numberz",
			give:     []interface{}{numberzp(gauntlet.Numberz(42))},
			want:     gauntlet.Numberz(42),
		},
		{
			function: "TestException",
			details:  "Xception",
			give:     []interface{}{stringp("Xception")},
			wantError: &gauntlet.Xception{
				ErrorCode: int32p(1001),
				Message:   stringp("Xception"),
			},
		},
		{
			function:      "TestException",
			details:       "TException",
			give:          []interface{}{stringp("TException")},
			wantErrorLike: `UnexpectedError: error for procedure "ThriftTest::testException" of service "yarpc-test": great sadness`,
		},
		{
			function: "TestException",
			details:  "no error",
			give:     []interface{}{stringp("yolo")},
		},
		{
			function: "TestI32",
			give:     []interface{}{int32p(123)},
			want:     int32(123),
		},
		{
			function: "TestI64",
			give:     []interface{}{int64p(18934714)},
			want:     int64(18934714),
		},
		{
			function: "TestInsanity",
			give: []interface{}{
				&gauntlet.Insanity{
					UserMap: map[gauntlet.Numberz]gauntlet.UserId{
						gauntlet.NumberzThree: gauntlet.UserId(100),
						gauntlet.Numberz(100): gauntlet.UserId(200),
					},
					Xtructs: []*gauntlet.Xtruct{
						{StringThing: stringp("0")},
						{ByteThing: bytep(1)},
						{I32Thing: int32p(2)},
						{I64Thing: int64p(3)},
					},
				},
			},
			want: map[gauntlet.UserId]map[gauntlet.Numberz]*gauntlet.Insanity{
				1: {
					gauntlet.NumberzTwo: &gauntlet.Insanity{
						UserMap: map[gauntlet.Numberz]gauntlet.UserId{
							gauntlet.NumberzThree: gauntlet.UserId(100),
							gauntlet.Numberz(100): gauntlet.UserId(200),
						},
						Xtructs: []*gauntlet.Xtruct{
							{StringThing: stringp("0")},
							{ByteThing: bytep(1)},
							{I32Thing: int32p(2)},
							{I64Thing: int64p(3)},
						},
					},
					gauntlet.NumberzThree: &gauntlet.Insanity{
						UserMap: map[gauntlet.Numberz]gauntlet.UserId{
							gauntlet.NumberzThree: gauntlet.UserId(100),
							gauntlet.Numberz(100): gauntlet.UserId(200),
						},
						Xtructs: []*gauntlet.Xtruct{
							{StringThing: stringp("0")},
							{ByteThing: bytep(1)},
							{I32Thing: int32p(2)},
							{I64Thing: int64p(3)},
						},
					},
				},
				2: {
					gauntlet.NumberzSix: &gauntlet.Insanity{},
				},
			},
		},
		{
			function: "TestList",
			give:     []interface{}{[]int32{1, 2, 3}},
			want:     []int32{1, 2, 3},
		},
		{
			function: "TestMap",
			give:     []interface{}{map[int32]int32{1: 2, 3: 4, 5: 6}},
			want:     map[int32]int32{1: 2, 3: 4, 5: 6},
		},
		{
			function: "TestMapMap",
			give:     []interface{}{int32p(42)},
			want: map[int32]map[int32]int32{
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
			function: "TestMulti",
			give: []interface{}{
				bytep(100),
				int32p(200),
				int64p(300),
				map[int16]string{1: "1", 2: "2", 3: "3"},
				numberzp(gauntlet.NumberzEight),
				useridp(42),
			},
			want: &gauntlet.Xtruct{
				StringThing: stringp("Hello2"),
				ByteThing:   bytep(100),
				I32Thing:    int32p(200),
				I64Thing:    int64p(300),
			},
		},
		{
			function: "TestMultiException",
			details:  "Xception",
			give:     []interface{}{stringp("Xception"), stringp("foo")},
			wantError: &gauntlet.Xception{
				ErrorCode: int32p(1001),
				Message:   stringp("This is an Xception"),
			},
		},
		{
			function: "TestMultiException",
			details:  "Xception2",
			give:     []interface{}{stringp("Xception2"), stringp("foo")},
			wantError: &gauntlet.Xception2{
				ErrorCode:   int32p(2002),
				StructThing: &gauntlet.Xtruct{StringThing: stringp("foo")},
			},
		},
		{
			function: "TestMultiException",
			details:  "no error",
			give:     []interface{}{stringp("hello"), stringp("foo")},
			want:     &gauntlet.Xtruct{StringThing: stringp("foo")},
		},
		{
			function: "TestNest",
			give: []interface{}{
				&gauntlet.Xtruct2{
					ByteThing: bytep(-1),
					I32Thing:  int32p(-1234),
					StructThing: &gauntlet.Xtruct{
						StringThing: stringp("0"),
						ByteThing:   bytep(1),
						I32Thing:    int32p(2),
						I64Thing:    int64p(3),
					},
				},
			},
			want: &gauntlet.Xtruct2{
				ByteThing: bytep(-1),
				I32Thing:  int32p(-1234),
				StructThing: &gauntlet.Xtruct{
					StringThing: stringp("0"),
					ByteThing:   bytep(1),
					I32Thing:    int32p(2),
					I64Thing:    int64p(3),
				},
			},
		},
		{
			function: "TestSet",
			give: []interface{}{
				map[int32]struct{}{
					1:  struct{}{},
					2:  struct{}{},
					-1: struct{}{},
					-2: struct{}{},
				},
			},
			want: map[int32]struct{}{
				1:  struct{}{},
				2:  struct{}{},
				-1: struct{}{},
				-2: struct{}{},
			},
		},
		{
			function: "TestString",
			give:     []interface{}{stringp("hello")},
			want:     "hello",
		},
		{
			function: "TestStringMap",
			give: []interface{}{
				map[string]string{
					"foo":   "bar",
					"hello": "world",
				},
			},
			want: map[string]string{
				"foo":   "bar",
				"hello": "world",
			},
		},
		{
			function: "TestStruct",
			give: []interface{}{
				&gauntlet.Xtruct{
					StringThing: stringp("0"),
					ByteThing:   bytep(1),
					I32Thing:    int32p(2),
					I64Thing:    int64p(3),
				},
			},
			want: &gauntlet.Xtruct{
				StringThing: stringp("0"),
				ByteThing:   bytep(1),
				I32Thing:    int32p(2),
				I64Thing:    int64p(3),
			},
		},
		{
			function: "TestTypedef",
			give:     []interface{}{useridp(42)},
			want:     gauntlet.UserId(42),
		},
		{
			function: "TestVoid",
			give:     []interface{}{},
		},
		{
			service:  "SecondService",
			function: "BlahBlah",
			give:     []interface{}{},
		},
		{
			service:  "SecondService",
			function: "SecondtestString",
			give:     []interface{}{stringp("hello")},
			want:     "hello",
		},
	}

	for _, tt := range tests {
		// We log in one of the following formats,
		//
		// $function: $message
		// $function: $description: $message
		// $service: $function: $message
		// $service: $function: $description: $message
		desc := tt.function
		if tt.details != "" {
			desc = desc + ": " + tt.details
		}
		if tt.service != "" {
			desc = tt.service + ": " + desc
		}

		client := buildClient(s, desc, tt.service, rpc.Channel("yarpc-test"))
		f := client.MethodByName(tt.function)
		if !checks.True(f.IsValid(), "%v: invalid function", desc) {
			continue
		}

		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		req := thrift.Request{
			Context: ctx,
			TTL:     time.Second, // TODO context TTL should be enough
		}

		args := []reflect.Value{reflect.ValueOf(&req)}
		if give, ok := buildArgs(s, desc, f.Type(), tt.give); ok {
			args = append(args, give...)
		} else {
			continue
		}

		got, err := extractCallResponse(s, desc, f.Call(args))
		if isUnrecognizedProcedure(err) {
			crossdock.Skipf(s, "%v: procedure not defined", desc)
			continue
		}

		if tt.wantError != nil || tt.wantErrorLike != "" {
			if !checks.Error(err, "%v: expected failure but got: %v", desc, got) {
				continue
			}
			if tt.wantError != nil {
				assert.Equal(tt.wantError, err, "%v: server returned error: %v", desc, err)
			}
			if tt.wantErrorLike != "" {
				assert.Contains(err.Error(), tt.wantErrorLike, "%v: server returned error: %v", desc, err)
			}
		} else {
			if !checks.NoError(err, "%v: call failed", desc) {
				continue
			}

			if tt.want != nil {
				assert.Equal(tt.want, got, "%v: server returned: %v", desc, got)
			}
		}
	}
}

func isUnrecognizedProcedure(err error) bool {
	if _, isBadRequest := err.(transport.BadRequestError); isBadRequest {
		// TODO: Once all other languages implement the gauntlet test
		// subject, we can remove this check.
		return strings.Contains(err.Error(), "unrecognized procedure")
	}
	return false
}

func buildClient(s crossdock.Sink, desc string, service string, channel transport.Channel) reflect.Value {
	switch service {
	case "", "ThriftTest":
		return reflect.ValueOf(thrifttestclient.New(channel))
	case "SecondService":
		return reflect.ValueOf(secondserviceclient.New(channel))
	default:
		crossdock.Fatals(s).Fail("", "%v: unknown thrift service", desc)
		return reflect.Value{} // we'll never actually get here
	}
}

func extractCallResponse(s crossdock.Sink, desc string, returns []reflect.Value) (interface{}, error) {
	var (
		err error
		got interface{}
	)

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
		crossdock.Assert(s).Fail("",
			"%v: received unexpected number of return values: %v", desc, returns)
	}

	return got, err
}

func buildArgs(s crossdock.Sink, desc string, ft reflect.Type, give []interface{}) (_ []reflect.Value, ok bool) {
	check := crossdock.Checks(s)
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

func bytep(x int8) *int8         { return &x }
func int32p(x int32) *int32      { return &x }
func int64p(x int64) *int64      { return &x }
func doublep(x float64) *float64 { return &x }
func stringp(x string) *string   { return &x }

func numberzp(x gauntlet.Numberz) *gauntlet.Numberz { return &x }
func useridp(x gauntlet.UserId) *gauntlet.UserId    { return &x }
