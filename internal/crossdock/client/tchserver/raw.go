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
	"strings"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/internal"
	"go.uber.org/yarpc/transport"

	"github.com/crossdock/crossdock-go"
)

func runRaw(t crossdock.T, dispatcher yarpc.Dispatcher) {
	hello(t, dispatcher)
	remoteTimeout(t, dispatcher)
}

func hello(t crossdock.T, dispatcher yarpc.Dispatcher) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	// TODO headers should be at yarpc, not transport
	headers := yarpc.NewHeaders().With("hello", "raw")
	token := random.Bytes(5)

	resBody, resMeta, err := rawCall(dispatcher, headers, "echo/raw", token)
	if skipOnConnRefused(t, err) {
		return
	}
	if checks.NoError(err, "raw: call failed") {
		assert.Equal(token, resBody, "body echoed")
		resHeaders := internal.RemoveVariableHeaderKeys(resMeta.Headers())
		assert.Equal(headers, resHeaders, "headers echoed")
	}
}

// remoteTimeout tests if a yarpc client returns a remote timeout error behind
// the TimeoutError interface when a remote tchannel handler returns a handler
// timeout.
func remoteTimeout(t crossdock.T, dispatcher yarpc.Dispatcher) {
	assert := crossdock.Assert(t)

	headers := yarpc.NewHeaders()
	token := random.Bytes(5)

	_, _, err := rawCall(dispatcher, headers, "handlertimeout/raw", token)
	if skipOnConnRefused(t, err) {
		return
	}
	if !assert.Error(err, "expected an error") {
		return
	}

	if transport.IsBadRequestError(err) {
		t.Skipf("handlertimeout/raw procedure not implemented: %v", err)
		return
	}

	assert.True(transport.IsTimeoutError(err), "returns a TimeoutError: %T", err)

	form := strings.HasPrefix(err.Error(),
		`Timeout: call to procedure "handlertimeout/raw" of service "service" from caller "caller" timed out after`)
	assert.True(form, "must be a remote handler timeout: %q", err.Error())
}

func rawCall(dispatcher yarpc.Dispatcher, headers yarpc.Headers, procedure string,
	token []byte) ([]byte, yarpc.CallResMeta, error) {
	client := raw.New(dispatcher.ClientConfig(serverName))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	reqMeta := yarpc.NewReqMeta().Procedure(procedure).Headers(headers)
	resBody, resMeta, err := client.Call(ctx, reqMeta, token)

	return resBody, resMeta, err
}
