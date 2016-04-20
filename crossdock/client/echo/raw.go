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

	"github.com/yarpc/yarpc-go/crossdock/client/behavior"
	"github.com/yarpc/yarpc-go/crossdock/client/random"
	"github.com/yarpc/yarpc-go/encoding/raw"

	"golang.org/x/net/context"
)

// Raw implements the 'raw' behavior.
func Raw(s behavior.Sink, p behavior.Params) {
	s = createEchoSink("raw", s, p)
	rpc := createRPC(s, p)

	client := raw.New(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	token := random.Bytes(5)
	resBody, _, err := client.Call(&raw.Request{
		Context:   ctx,
		Procedure: "echo/raw",
		TTL:       time.Second, // TODO context already has timeout; use that
	}, token)

	if err != nil {
		behavior.Fatalf(s, "call to echo/raw failed: %v", err)
	}

	if !bytes.Equal(token, resBody) {
		behavior.Fatalf(s, "expected %v, got %v", token, resBody)
	}

	behavior.Successf(s, "server said: %v", resBody)
}
