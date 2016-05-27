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
	"time"

	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/behavior/random"
	"github.com/yarpc/yarpc-go/crossdock/behavior/rpc"
	"github.com/yarpc/yarpc-go/encoding/json"

	"golang.org/x/net/context"
)

// jsonEcho contains an echo request or response for the JSON echo endpoint.
type jsonEcho struct {
	Token string `json:"token"`
}

// JSON implements the 'json' behavior.
func JSON(s crossdock.T, p crossdock.Params) {
	s = createEchoSink("json", s, p)
	rpc := rpc.Create(s, p)

	client := json.New(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	var response jsonEcho
	token := random.String(5)
	_, err := client.Call(
		&json.Request{
			Context:   ctx,
			Procedure: "echo",
			TTL:       time.Second, // TODO context already has timeout; use that
		},
		&jsonEcho{Token: token},
		&response,
	)
	crossdock.Fatals(s).NoError(err, "call to echo failed: %v", err)
	crossdock.Assert(s).Equal(token, response.Token, "server said: %v", response.Token)
}
