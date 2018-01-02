// Copyright (c) 2018 Uber Technologies, Inc.
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
	"context"
	"time"

	"github.com/crossdock/crossdock-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/crossdock/client/gauntlet"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/internal"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo/echoclient"
)

func runThrift(t crossdock.T, dispatcher *yarpc.Dispatcher) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := map[string]string{"hello": "thrift"}
	token := random.String(5)

	resBody, resHeaders, err := thriftCall(dispatcher, headers, token)
	if skipOnConnRefused(t, err) {
		return
	}
	if checks.NoError(err, "thrift: call failed") {
		assert.Equal(token, resBody, "body echoed")
		internal.RemoveVariableMapKeys(resHeaders)
		assert.Equal(headers, resHeaders, "headers echoed")
	}

	t.Tag("server", t.Param(params.Server))
	gauntlet.RunGauntlet(t, gauntlet.Config{
		Dispatcher: dispatcher,
		ServerName: serverName,
	})
}

func thriftCall(
	dispatcher *yarpc.Dispatcher,
	headers map[string]string,
	token string,
) (string, map[string]string, error) {
	client := echoclient.New(dispatcher.ClientConfig(serverName))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var (
		opts       []yarpc.CallOption
		resHeaders map[string]string
	)
	for k, v := range headers {
		opts = append(opts, yarpc.WithHeader(k, v))
	}
	opts = append(opts, yarpc.ResponseHeaders(&resHeaders))

	ping := &echo.Ping{Beep: token}
	resBody, err := client.Echo(ctx, ping, opts...)
	if err != nil {
		return "", resHeaders, err
	}
	return resBody.Boop, resHeaders, err
}
