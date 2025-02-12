// Copyright (c) 2025 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/internal"
)

func runJSON(t crossdock.T, dispatcher *yarpc.Dispatcher) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := map[string]string{"hello": "json"}
	token := random.String(5)

	resBody, resHeaders, err := jsonCall(dispatcher, headers, token)
	if skipOnConnRefused(t, err) {
		return
	}
	if checks.NoError(err, "json: call failed") {
		assert.Equal(token, resBody, "body echoed")
		internal.RemoveVariableMapKeys(resHeaders)
		assert.Equal(headers, resHeaders, "headers echoed")
	}
}

type jsonEcho struct {
	Token string `json:"token"`
}

func jsonCall(
	dispatcher *yarpc.Dispatcher,
	headers map[string]string,
	token string,
) (string, map[string]string, error) {
	client := json.New(dispatcher.ClientConfig(serverName))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var (
		opts       []yarpc.CallOption
		resHeaders map[string]string
		resBody    jsonEcho
	)

	for k, v := range headers {
		opts = append(opts, yarpc.WithHeader(k, v))
	}
	opts = append(opts, yarpc.ResponseHeaders(&resHeaders))

	reqBody := &jsonEcho{Token: token}
	err := client.Call(ctx, "echo", reqBody, &resBody, opts...)
	return resBody.Token, resHeaders, err
}
