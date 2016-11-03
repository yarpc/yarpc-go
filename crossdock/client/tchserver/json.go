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
	"context"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/crossdock/client/random"
	"go.uber.org/yarpc/crossdock/internal"
	"go.uber.org/yarpc/encoding/json"

	"github.com/crossdock/crossdock-go"
)

func runJSON(t crossdock.T, dispatcher yarpc.Dispatcher) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := yarpc.NewHeaders().With("hello", "json")
	token := random.String(5)

	resBody, resMeta, err := jsonCall(dispatcher, headers, token)
	if skipOnConnRefused(t, err) {
		return
	}
	if checks.NoError(err, "json: call failed") {
		assert.Equal(token, resBody, "body echoed")
		resHeaders := internal.RemoveVariableHeaderKeys(resMeta.Headers())
		assert.Equal(headers, resHeaders, "headers echoed")
	}
}

type jsonEcho struct {
	Token string `json:"token"`
}

func jsonCall(dispatcher yarpc.Dispatcher, headers yarpc.Headers, token string) (string, yarpc.CallResMeta, error) {
	client := json.New(dispatcher.Channel(serverName))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	reqMeta := yarpc.NewReqMeta().Procedure("echo").Headers(headers)
	reqBody := &jsonEcho{Token: token}

	var resBody jsonEcho
	resMeta, err := client.Call(ctx, reqMeta, reqBody, &resBody)
	return resBody.Token, resMeta, err
}
