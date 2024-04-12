// Copyright (c) 2024 Uber Technologies, Inc.
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

package headers

import (
	"context"
	"time"

	"github.com/crossdock/crossdock-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	disp "go.uber.org/yarpc/internal/crossdock/client/dispatcher"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/internal"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo/echoclient"
)

func createHeadersT(t crossdock.T) crossdock.T {
	t.Tag("transport", t.Param(params.Transport))
	t.Tag("encoding", t.Param(params.Encoding))
	t.Tag("server", t.Param(params.Server))
	return t
}

// Run runs the headers behavior
func Run(t crossdock.T) {
	t = createHeadersT(t)

	fatals := crossdock.Fatals(t)
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	dispatcher := disp.Create(t)
	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	var caller headerCaller
	encoding := t.Param(params.Encoding)
	switch encoding {
	case "raw":
		caller = rawCaller{raw.New(dispatcher.ClientConfig("yarpc-test"))}
	case "json":
		caller = jsonCaller{json.New(dispatcher.ClientConfig("yarpc-test"))}
	case "thrift":
		caller = thriftCaller{echoclient.New(dispatcher.ClientConfig("yarpc-test"))}
	default:
		fatals.Fail("", "unknown encoding %q", encoding)
	}

	token1 := random.String(10)
	token2 := random.String(10)

	tests := []struct {
		desc string
		give map[string]string
		want map[string]string
	}{
		{
			"valid headers",
			map[string]string{"token1": token1, "token2": token2},
			map[string]string{"token1": token1, "token2": token2},
		},
		{
			"non-string values",
			map[string]string{"token": "42"},
			map[string]string{"token": "42"},
		},
		{
			"empty strings",
			map[string]string{"token": ""},
			map[string]string{"token": ""},
		},
		{
			"no headers",
			nil,
			nil,
		},
		{
			"empty map",
			map[string]string{},
			map[string]string{},
		},
		{
			"varying casing",
			map[string]string{"ToKeN1": token1, "tOkEn2": token2},
			map[string]string{"token1": token1, "token2": token2},
		},
		{
			"http header conflict",
			map[string]string{"Rpc-Procedure": "does not exist"},
			map[string]string{"rpc-procedure": "does not exist"},
		},
		{
			"mixed case value",
			map[string]string{"token": "MIXED case Value"},
			map[string]string{"token": "MIXED case Value"},
		},
	}

	for _, tt := range tests {
		gotHeaders, err := caller.Call(tt.give)
		if checks.NoError(err, "%v: call failed", tt.desc) {
			internal.RemoveVariableMapKeys(gotHeaders)

			// assert.Equal doesn't work with nil maps
			if len(tt.want) == 0 {
				assert.Empty(gotHeaders, "%v: returns valid headers", tt.desc)
			} else {
				assert.Equal(tt.want, gotHeaders, "%v: returns valid headers", tt.desc)
			}
		}
	}
}

type headerCaller interface {
	Call(map[string]string) (map[string]string, error)
}

type rawCaller struct{ c raw.Client }

func (c rawCaller) Call(h map[string]string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var (
		opts       []yarpc.CallOption
		resHeaders map[string]string
	)
	for k, v := range h {
		opts = append(opts, yarpc.WithHeader(k, v))
	}
	opts = append(opts, yarpc.ResponseHeaders(&resHeaders))

	_, err := c.c.Call(ctx, "echo/raw", []byte("hello"), opts...)
	return resHeaders, err
}

type jsonCaller struct{ c json.Client }

func (c jsonCaller) Call(h map[string]string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var (
		opts       []yarpc.CallOption
		resHeaders map[string]string
	)
	for k, v := range h {
		opts = append(opts, yarpc.WithHeader(k, v))
	}
	opts = append(opts, yarpc.ResponseHeaders(&resHeaders))

	var resBody interface{}
	err := c.c.Call(ctx, "echo", map[string]interface{}{}, &resBody, opts...)
	return resHeaders, err
}

type thriftCaller struct{ c echoclient.Interface }

func (c thriftCaller) Call(h map[string]string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var (
		opts       []yarpc.CallOption
		resHeaders map[string]string
	)
	for k, v := range h {
		opts = append(opts, yarpc.WithHeader(k, v))
	}
	opts = append(opts, yarpc.ResponseHeaders(&resHeaders))

	_, err := c.c.Echo(ctx, &echo.Ping{Beep: "hello"}, opts...)
	return resHeaders, err
}
