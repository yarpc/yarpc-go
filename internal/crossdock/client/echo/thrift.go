// Copyright (c) 2017 Uber Technologies, Inc.
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
	"context"
	"time"

	disp "go.uber.org/yarpc/internal/crossdock/client/dispatcher"
	"go.uber.org/yarpc/internal/crossdock/client/random"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo/echoclient"

	"github.com/crossdock/crossdock-go"
)

// Thrift implements the 'thrift' behavior.
func Thrift(t crossdock.T) {
	t = createEchoT("thrift", t)
	fatals := crossdock.Fatals(t)

	dispatcher := disp.Create(t)
	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	client := echoclient.New(dispatcher.ClientConfig("yarpc-test"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	token := random.String(5)

	pong, err := client.Echo(ctx, &echo.Ping{Beep: token})

	crossdock.Fatals(t).NoError(err, "call to Echo::echo failed: %v", err)
	crossdock.Assert(t).Equal(token, pong.Boop, "server said: %v", pong.Boop)
}
