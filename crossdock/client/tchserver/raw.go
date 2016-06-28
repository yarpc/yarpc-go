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
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/encoding/raw"

	"github.com/crossdock/crossdock-go"
	"golang.org/x/net/context"
)

func runRaw(t crossdock.T, rpc yarpc.RPC) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	// TODO headers should be at yarpc, not transport
	headers := yarpc.NewHeaders().With("hello", "raw")
	token := random.Bytes(5)

	resBody, resMeta, err := rawCall(rpc, headers, token)
	if skipOnConnRefused(t, err) {
		return
	}
	if checks.NoError(err, "raw: call failed") {
		assert.Equal(token, resBody, "body echoed")
		assert.Equal(headers, resMeta.Headers(), "headers echoed")
	}
}

func rawCall(rpc yarpc.RPC, headers yarpc.Headers, token []byte) ([]byte, yarpc.CallResMeta, error) {
	client := raw.New(rpc.Channel(serverName))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	reqMeta := yarpc.NewReqMeta(ctx).Procedure("echo/raw").Headers(headers)
	resBody, resMeta, err := client.Call(reqMeta, token)
	return resBody, resMeta, err
}
