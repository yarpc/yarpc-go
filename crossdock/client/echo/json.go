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

	"go.uber.org/yarpc"
	disp "go.uber.org/yarpc/crossdock/client/dispatcher"
	"go.uber.org/yarpc/crossdock/client/random"
	"go.uber.org/yarpc/encoding/json"

	"context"

	"github.com/crossdock/crossdock-go"
)

// jsonEcho contains an echo request or response for the JSON echo endpoint.
type jsonEcho struct {
	Token string `json:"token"`
}

// JSON implements the 'json' behavior.
func JSON(t crossdock.T) {
	t = createEchoT("json", t)
	fatals := crossdock.Fatals(t)

	dispatcher := disp.Create(t)
	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	client := json.New(dispatcher.Channel("yarpc-test"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var response jsonEcho
	token := random.String(5)
	_, err := client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echo"),
		&jsonEcho{Token: token},
		&response,
	)
	crossdock.Fatals(t).NoError(err, "call to echo failed: %v", err)
	crossdock.Assert(t).Equal(token, response.Token, "server said: %v", response.Token)
}
