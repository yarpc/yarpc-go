// Copyright (c) 2024 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/internal"
	"go.uber.org/yarpc/yarpcerrors"
)

func runRaw(t crossdock.T, dispatcher *yarpc.Dispatcher) {
	hello(t, dispatcher)
	remoteTimeout(t, dispatcher)
}

func hello(t crossdock.T, dispatcher *yarpc.Dispatcher) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	// TODO headers should be at yarpc, not transport
	headers := map[string]string{"hello": "raw"}
	token := random.Bytes(5)

	resBody, resHeaders, err := rawCall(dispatcher, headers, "echo/raw", token)
	if skipOnConnRefused(t, err) {
		return
	}
	if checks.NoError(err, "raw: call failed") {
		assert.Equal(token, resBody, "body echoed")
		internal.RemoveVariableMapKeys(resHeaders)
		assert.Equal(headers, resHeaders, "headers echoed")
	}
}

// remoteTimeout tests if a yarpc client returns a remote timeout error behind
// the TimeoutError interface when a remote tchannel handler returns a handler
// timeout.
func remoteTimeout(t crossdock.T, dispatcher *yarpc.Dispatcher) {
	assert := crossdock.Assert(t)

	token := random.Bytes(5)

	_, _, err := rawCall(dispatcher, nil, "handlertimeout/raw", token)
	if skipOnConnRefused(t, err) {
		return
	}
	if !assert.Error(err, "expected an error") {
		return
	}

	if yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInvalidArgument {
		t.Skipf("handlertimeout/raw procedure not implemented: %v", err)
		return
	}

	assert.Equal(yarpcerrors.CodeDeadlineExceeded, yarpcerrors.FromError(err).Code(), "is an error with code CodeDeadlineExceeded: %v", err)
}

func rawCall(
	dispatcher *yarpc.Dispatcher,
	headers map[string]string,
	procedure string,
	token []byte,
) ([]byte, map[string]string, error) {
	client := raw.New(dispatcher.ClientConfig(serverName))

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

	res, err := client.Call(ctx, procedure, token, opts...)
	return res, resHeaders, err
}
