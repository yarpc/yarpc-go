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

package headers

import (
	"context"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	disp "go.uber.org/yarpc/internal/crossdock/client/dispatcher"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/internal"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo/yarpc/echoclient"

	"github.com/crossdock/crossdock-go"
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
		give yarpc.Headers
		want yarpc.Headers
	}{
		{
			"valid headers",
			yarpc.NewHeaders().With("token1", token1).With("token2", token2),
			yarpc.NewHeaders().With("token1", token1).With("token2", token2),
		},
		{
			"non-string values",
			yarpc.NewHeaders().With("token", "42"),
			yarpc.NewHeaders().With("token", "42"),
		},
		{
			"empty strings",
			yarpc.NewHeaders().With("token", ""),
			yarpc.NewHeaders().With("token", ""),
		},
		{
			"no headers",
			yarpc.Headers{},
			yarpc.NewHeaders(),
		},
		{
			"empty map",
			yarpc.NewHeaders(),
			yarpc.NewHeaders(),
		},
		{
			"varying casing",
			yarpc.NewHeaders().With("ToKeN1", token1).With("tOkEn2", token2),
			yarpc.NewHeaders().With("token1", token1).With("token2", token2),
		},
		{
			"http header conflict",
			yarpc.NewHeaders().With("Rpc-Procedure", "does not exist"),
			yarpc.NewHeaders().With("rpc-procedure", "does not exist"),
		},
		{
			"mixed case value",
			yarpc.NewHeaders().With("token", "MIXED case Value"),
			yarpc.NewHeaders().With("token", "MIXED case Value"),
		},
	}

	for _, tt := range tests {
		got, err := caller.Call(tt.give)
		if checks.NoError(err, "%v: call failed", tt.desc) {
			gotHeaders := internal.RemoveVariableHeaderKeys(got)
			assert.Equal(tt.want, gotHeaders, "%v: returns valid headers", tt.desc)
		}
	}
}

type headerCaller interface {
	Call(yarpc.Headers) (yarpc.Headers, error)
}

type rawCaller struct{ c raw.Client }

func (c rawCaller) Call(h yarpc.Headers) (yarpc.Headers, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var (
		opts       []yarpc.CallOption
		resHeaders yarpc.Headers
	)
	for _, k := range h.Keys() {
		if v, ok := h.Get(k); ok {
			opts = append(opts, yarpc.WithHeader(k, v))
		}
	}
	opts = append(opts, yarpc.ResponseHeaders(&resHeaders))

	if _, err := c.c.Call(ctx, "echo/raw", []byte("hello"), opts...); err != nil {
		return yarpc.Headers{}, err
	}
	return resHeaders, nil
}

type jsonCaller struct{ c json.Client }

func (c jsonCaller) Call(h yarpc.Headers) (yarpc.Headers, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var resBody interface{}
	res, err := c.c.Call(
		ctx,
		yarpc.NewReqMeta().Headers(h).Procedure("echo"),
		map[string]interface{}{}, &resBody)

	if err != nil {
		return yarpc.Headers{}, err
	}
	return res.Headers(), nil
}

type thriftCaller struct{ c echoclient.Interface }

func (c thriftCaller) Call(h yarpc.Headers) (yarpc.Headers, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, res, err := c.c.Echo(
		ctx,
		yarpc.NewReqMeta().Headers(h),
		&echo.Ping{Beep: "hello"})

	if err != nil {
		return yarpc.Headers{}, err
	}
	return res.Headers(), nil
}
