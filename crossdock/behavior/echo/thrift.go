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
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/yarpc/echoclient"
	"github.com/yarpc/yarpc-go/encoding/thrift"

	"golang.org/x/net/context"
)

// Thrift implements the 'thrift' behavior.
func Thrift(t crossdock.T, p crossdock.Params) {
	t = createEchoT("thrift", t, p)
	rpc := rpc.Create(t, p)

	client := echoclient.New(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	token := random.String(5)
	pong, _, err := client.Echo(
		&thrift.Request{
			Context: ctx,
			TTL:     time.Second, // TODO context already has timeout; use that
		},
		&echo.Ping{Beep: token},
	)

	crossdock.Fatals(t).NoError(err, "call to Echo::echo failed: %v", err)
	crossdock.Assert(t).Equal(token, pong.Boop, "server said: %v", pong.Boop)
}
