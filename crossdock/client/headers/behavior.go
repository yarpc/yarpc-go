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
	"time"

	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/client/params"
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/crossdock/client/rpc"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/yarpc/echoclient"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
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
	rpc := rpc.Create(t)

	fatals := crossdock.Fatals(t)
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	var caller headerCaller
	encoding := t.Param(params.Encoding)
	switch encoding {
	case "raw":
		caller = rawCaller{raw.New(rpc.Channel("yarpc-test"))}
	case "json":
		caller = jsonCaller{json.New(rpc.Channel("yarpc-test"))}
	case "thrift":
		caller = thriftCaller{echoclient.New(rpc.Channel("yarpc-test"))}
	default:
		fatals.Fail("", "unknown encoding %q", encoding)
	}

	token1 := random.String(10)
	token2 := random.String(10)

	tests := []struct {
		desc string
		give transport.Headers
		want transport.Headers
	}{
		{
			"valid headers",
			transport.Headers{"token1": token1, "token2": token2},
			transport.Headers{"token1": token1, "token2": token2},
		},
		{
			"non-string values",
			transport.Headers{"token": "42"},
			transport.Headers{"token": "42"},
		},
		{
			"empty strings",
			transport.Headers{"token": ""},
			transport.Headers{"token": ""},
		},
		{
			"no headers",
			nil,
			transport.Headers{},
		},
		{
			"empty map",
			transport.Headers{},
			transport.Headers{},
		},
		{
			"varying casing",
			transport.Headers{"ToKeN1": token1, "tOkEn2": token2},
			transport.Headers{"token1": token1, "token2": token2},
		},
		{
			"http header conflict",
			transport.Headers{"Rpc-Procedure": "does not exist"},
			transport.Headers{"rpc-procedure": "does not exist"},
		},
		{
			"mixed case value",
			transport.Headers{"token": "MIXED case Value"},
			transport.Headers{"token": "MIXED case Value"},
		},
	}

	for _, tt := range tests {
		got, err := caller.Call(tt.give)
		if checks.NoError(err, "%v: call failed", tt.desc) {
			assert.Equal(tt.want, got, "%v: returns valid headers", tt.desc)
		}
	}
}

type headerCaller interface {
	Call(transport.Headers) (transport.Headers, error)
}

type rawCaller struct{ c raw.Client }

func (c rawCaller) Call(h transport.Headers) (transport.Headers, error) {
	_, res, err := c.c.Call(&raw.ReqMeta{
		Context:   newTestContext(),
		Headers:   h,
		Procedure: "echo/raw",
	}, []byte("hello"))

	if err != nil {
		return nil, err
	}
	return res.Headers, nil
}

type jsonCaller struct{ c json.Client }

func (c jsonCaller) Call(h transport.Headers) (transport.Headers, error) {
	var resBody interface{}
	res, err := c.c.Call(&json.ReqMeta{
		Context:   newTestContext(),
		Headers:   h,
		Procedure: "echo",
	}, map[string]interface{}{}, &resBody)

	if err != nil {
		return nil, err
	}
	return res.Headers, nil
}

type thriftCaller struct{ c echoclient.Interface }

func (c thriftCaller) Call(h transport.Headers) (transport.Headers, error) {
	_, res, err := c.c.Echo(&thrift.ReqMeta{
		Context: newTestContext(),
		Headers: h,
	}, &echo.Ping{Beep: "hello"})

	if err != nil {
		return nil, err
	}
	return res.Headers, nil
}

func newTestContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	return ctx
}
