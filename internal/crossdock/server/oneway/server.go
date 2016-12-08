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
	"fmt"
	"os"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/crossdock/thrift/oneway/yarpc/onewayserver"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/x/redis"
)

var dispatcher yarpc.Dispatcher

// Start starts the test server that clients will make requests to
func Start() {
	httpTransport := http.NewTransport()
	inbounds := []transport.Inbound{httpTransport.NewInbound(":8084")}

	if redisIsExpectedRunning() {
		rds := redis.NewInbound(
			redis.NewRedis5Client("redis:6379"),
			"yarpc/oneway",
			"yarpc/oneway/processing",
			time.Second,
		)
		inbounds = append(inbounds, rds)
	}

	dispatcher = yarpc.NewDispatcher(yarpc.Config{
		Name:     "oneway-server",
		Inbounds: inbounds,
	})

	// Echo procedures make an RPC back to the caller with the same context,
	// and body over http/raw
	h := &onewayHandler{http.NewTransport()}
	dispatcher.Register(raw.OnewayProcedure("echo/raw", h.EchoRaw))
	dispatcher.Register(json.OnewayProcedure("echo/json", h.EchoJSON))
	dispatcher.Register(onewayserver.New(h))

	if err := dispatcher.Start(); err != nil {
		fmt.Println("error:", err.Error())
	}
}

// Stop stops running the RPC test subject
func Stop() {
	if dispatcher == nil {
		return
	}
	if err := dispatcher.Stop(); err != nil {
		fmt.Println("failed to stop:", err.Error())
	}
}

// redisIsExpectedRunning checks to see if a redis server is expected to be
// available
func redisIsExpectedRunning() bool {
	return os.Getenv("REDIS") == "enabled"
}
