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
	"github.com/yarpc/yarpc-go/crossdock/client/gauntlet"
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/yarpc/echoclient"
	"github.com/yarpc/yarpc-go/encoding/thrift"

	"github.com/crossdock/crossdock-go"
	"golang.org/x/net/context"
)

func runThrift(t crossdock.T, dispatcher yarpc.Dispatcher) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := yarpc.NewHeaders().With("hello", "thrift")
	token := random.String(5)

	resBody, resMeta, err := thriftCall(dispatcher, headers, token)
	if skipOnConnRefused(t, err) {
		return
	}
	if checks.NoError(err, "thrift: call failed") {
		assert.Equal(token, resBody, "body echoed")
		assert.Equal(headers, resMeta.Headers(), "headers echoed")
	}

	gauntlet.RunGauntlet(t, dispatcher, serverName)
}

func thriftCall(dispatcher yarpc.Dispatcher, headers yarpc.Headers, token string) (string, yarpc.CallResMeta, error) {
	// NOTE(abg): Enveloping is disabled in old cross-language tests until the
	// other YARPC implementations catch up.
	client := echoclient.New(dispatcher.Channel(serverName), thrift.DisableEnveloping)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	reqMeta := yarpc.NewReqMeta(ctx).Headers(headers)
	ping := &echo.Ping{Beep: token}

	resBody, resMeta, err := client.Echo(reqMeta, ping)
	if err != nil {
		return "", nil, err
	}
	return resBody.Boop, resMeta, err
}
