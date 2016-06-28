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

package echo

import (
	"bytes"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/crossdock/client/rpc"
	"github.com/yarpc/yarpc-go/encoding/raw"

	"github.com/crossdock/crossdock-go"
	"golang.org/x/net/context"
)

// Raw implements the 'raw' behavior.
func Raw(t crossdock.T) {
	t = createEchoT("raw", t)
	rpc := rpc.Create(t)

	client := raw.New(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	token := random.Bytes(5)
	resBody, _, err := client.Call(
		yarpc.NewReqMeta(ctx).Procedure("echo/raw"),
		token)

	crossdock.Fatals(t).NoError(err, "call to echo/raw failed: %v", err)
	crossdock.Assert(t).True(bytes.Equal(token, resBody), "server said: %v", resBody)
}
