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
	"time"

	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/behavior/random"
	"github.com/yarpc/yarpc-go/crossdock/thrift/gen-go/echo"

	"github.com/uber/tchannel-go/thrift"
)

func runThrift(t crossdock.T, call call) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := map[string]string{
		"hello": "thrift",
	}
	token := random.String(5)

	resp, respHeaders, err := thriftCall(call, headers, token)
	if checks.NoError(err, "thrift: call failed") {
		assert.Equal(token, resp.Boop, "body echoed")
		assert.Equal(headers, respHeaders, "headers echoed")
	}
}

func thriftCall(call call, headers map[string]string, token string) (*echo.Pong, map[string]string, error) {
	call.Channel.Peers().Add(call.ServerHostPort)

	ctx, cancel := thrift.NewContext(time.Second)
	ctx = thrift.WithHeaders(ctx, headers)
	defer cancel()

	client := echo.NewTChanEchoClient(thrift.NewClient(call.Channel, serverName, nil))
	pong, err := client.Echo(ctx, &echo.Ping{Beep: token})
	return pong, ctx.ResponseHeaders(), err
}
