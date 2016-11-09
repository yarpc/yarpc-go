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

package oneway

import (
	"context"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/transport"

	"github.com/crossdock/crossdock-go"
)

type jsonToken struct {
	Token string `json:"token"`
}

// JSON starts an http run using JSON encoding
func JSON(t crossdock.T, dispatcher yarpc.Dispatcher) {
	fatals := crossdock.Fatals(t)

	client := json.New(dispatcher.Channel("oneway-test"))

	token := getRandomID()

	var ack transport.Ack
	ack, err := client.CallOneway(
		context.Background(),
		yarpc.NewReqMeta().Procedure("echo/json"),
		&jsonToken{Token: token},
		&ack)

	// ensure channel hasn't been filled yet
	select {
	case <-serverCalledBack:
		fatals.FailNow("oneway json test failed", "client waited for server to fill channel")
	default:
	}

	fatals.NoError(err, "call to oneway/json failed: %v", err)
	fatals.NotNil(ack, "ack is nil")

	serverToken := <-serverCalledBack
	fatals.Equal(token, string(serverToken), "Client/Server token mismatch")
}
