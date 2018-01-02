// Copyright (c) 2018 Uber Technologies, Inc.
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
	"time"

	"github.com/crossdock/crossdock-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
)

type jsonToken struct {
	Token string `json:"token"`
}

// JSON starts an http run using JSON encoding
func JSON(t crossdock.T, dispatcher *yarpc.Dispatcher, serverCalledBack <-chan []byte, callBackAddr string) {
	assert := crossdock.Assert(t)
	fatals := crossdock.Fatals(t)

	client := json.New(dispatcher.ClientConfig("oneway-server"))
	token := getRandomID()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ack, err := client.CallOneway(
		ctx,
		"echo/json",
		&jsonToken{Token: token},
		yarpc.WithHeader("callBackAddr", callBackAddr),
	)

	// ensure channel hasn't been filled yet
	select {
	case <-serverCalledBack:
		fatals.FailNow("oneway json test failed", "client waited for server to fill channel")
	default:
	}

	fatals.NoError(err, "call to oneway/json failed: %v", err)
	fatals.NotNil(ack, "ack is nil")

	serverToken := <-serverCalledBack
	assert.Equal(token, string(serverToken), "JSON token mismatch")
}
