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
	"github.com/yarpc/yarpc-go/crossdock/client/behavior"
	"github.com/yarpc/yarpc-go/crossdock/client/params"
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/yarpc/echoclient"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// headersEntry is an entry emitted by the headers behavior.
type headersEntry struct {
	crossdock.Entry

	Transport string `json:"transport"`
	Encoding  string `json:"encoding"`
	Server    string `json:"server"`
}

// headersSink wraps a sink to emit headersEntry entries.
type headersSink struct {
	crossdock.Sink

	Transport string
	Encoding  string
	Server    string
}

func (s headersSink) Put(e interface{}) {
	s.Sink.Put(headersEntry{
		Entry:     e.(crossdock.Entry),
		Transport: s.Transport,
		Encoding:  s.Encoding,
		Server:    s.Server,
	})
}

func createHeadersSink(s crossdock.Sink, ps crossdock.Params) crossdock.Sink {
	return headersSink{
		Sink:      s,
		Transport: ps.Param(params.Transport),
		Encoding:  ps.Param(params.Encoding),
		Server:    ps.Param(params.Server),
	}
}

// Run runs the headers behavior
func Run(s crossdock.Sink, ps crossdock.Params) {
	s = createHeadersSink(s, ps)
	rpc := behavior.CreateRPC(s, ps)

	fatals := crossdock.Fatals(s)
	assert := crossdock.Assert(s)
	checks := crossdock.Checks(s)

	var caller headerCaller
	encoding := ps.Param(params.Encoding)
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
	_, res, err := c.c.Call(&raw.Request{
		Context:   newTestContext(),
		Headers:   h,
		Procedure: "echo/raw",
		TTL:       time.Second, // TODO context contains the timeout
	}, []byte("hello"))

	if err != nil {
		return nil, err
	}
	return res.Headers, nil
}

type jsonCaller struct{ c json.Client }

func (c jsonCaller) Call(h transport.Headers) (transport.Headers, error) {
	var resBody interface{}
	res, err := c.c.Call(&json.Request{
		Context:   newTestContext(),
		Headers:   h,
		Procedure: "echo",
		TTL:       time.Second, // TODO context contains the timeout
	}, map[string]interface{}{}, &resBody)

	if err != nil {
		return nil, err
	}
	return res.Headers, nil
}

type thriftCaller struct{ c echoclient.Interface }

func (c thriftCaller) Call(h transport.Headers) (transport.Headers, error) {
	_, res, err := c.c.Echo(&thrift.Request{
		Context: newTestContext(),
		Headers: h,
		TTL:     time.Second,
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
