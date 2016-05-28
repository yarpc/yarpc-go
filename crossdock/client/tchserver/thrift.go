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

package tchserver

import (
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/yarpc/echoclient"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

func runThrift(t crossdock.T, rpc yarpc.RPC) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := transport.Headers{
		"hello": "thrift",
	}
	token := random.String(5)

	resBody, resMeta, err := thriftCall(rpc, headers, token)
	if skipOnConnRefused(t, err) {
		return
	}
	if checks.NoError(err, "thrift: call failed") {
		assert.Equal(token, resBody, "body echoed")
		assert.Equal(headers, resMeta.Headers, "headers echoed")
	}
}

func thriftCall(rpc yarpc.RPC, headers transport.Headers, token string) (string, *thrift.Response, error) {
	client := echoclient.New(rpc.Channel(serverName))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	reqMeta := &thrift.Request{
		Context: ctx,
		TTL:     time.Second,
		Headers: headers,
	}
	ping := &echo.Ping{Beep: token}

	resBody, resMeta, err := client.Echo(reqMeta, ping)
	if err != nil {
		return "", nil, err
	}
	return resBody.Boop, resMeta, err
}
