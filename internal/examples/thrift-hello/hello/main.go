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

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/yarpc/internal/examples/thrift-hello/hello/echo"
	"go.uber.org/yarpc/internal/examples/thrift-hello/hello/echo/yarpc/helloclient"
	"go.uber.org/yarpc/internal/examples/thrift-hello/hello/echo/yarpc/helloserver"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/http"
)

//go:generate thriftrw --plugin=yarpc echo.thrift

func main() {
	// configure a dispatcher for the service "hello",
	// expose the service over an HTTP inbound on port 8086,
	// and configure outbound calls to service "hello" over HTTP port 8086 as well
	http := http.NewTransport()
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Inbounds: yarpc.Inbounds{
			http.NewInbound(":8086"),
		},
		Outbounds: yarpc.Outbounds{
			"hello": {
				Unary: http.NewSingleOutbound("http://127.0.0.1:8086"),
			},
		},
	})

	// register the Thrift handler which implements echo.thrift,
	// whose shapes are available in the hello/ dir, as generated by
	// the go:generate statement at the top of this file
	dispatcher.Register(helloserver.New(&helloHandler{}))

	// start the dispatcher, which enables requests to be sent and received
	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	// create a Thrift client configured to call the "hello" service
	// using the dispatcher and associated "hello" outbound
	client := helloclient.New(dispatcher.ClientConfig("hello"))

	// build a context with a 1 second deadline
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// use the Thrift client to call the Echo procedure using YARPC
	res, err := client.Echo(ctx, &echo.EchoRequest{Message: "Hello world", Count: 1})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	select {}
}

type helloHandler struct{}

func (h helloHandler) Echo(ctx context.Context, e *echo.EchoRequest) (*echo.EchoResponse, error) {
	return &echo.EchoResponse{Message: e.Message, Count: e.Count + 1}, nil
}
