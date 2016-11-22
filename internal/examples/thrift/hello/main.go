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

	"go.uber.org/yarpc/internal/examples/thrift/hello/echo"
	"go.uber.org/yarpc/internal/examples/thrift/hello/echo/yarpc/helloclient"
	"go.uber.org/yarpc/internal/examples/thrift/hello/echo/yarpc/helloserver"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/http"
)

//go:generate thriftrw --plugin=yarpc echo.thrift

func main() {

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Inbounds: yarpc.Inbounds{
			http.NewInbound(":8086"),
		},
		Outbounds: yarpc.Outbounds{
			"hello": {
				Unary: http.NewOutbound("http://127.0.0.1:8086"),
			},
		},
	})

	dispatcher.Register(helloserver.New(&helloHandler{}))
	client := helloclient.New(dispatcher.Channel("hello"))

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	response, headers := call(client, "Hi There")
	fmt.Println(response, headers)

	select {}
}

type helloHandler struct{}

func (h helloHandler) Echo(ctx context.Context, reqMeta yarpc.ReqMeta, e *echo.EchoRequest) (*echo.EchoResponse, yarpc.ResMeta, error) {
	return &echo.EchoResponse{Message: e.Message, Count: e.Count + 1},
		yarpc.NewResMeta().Headers(reqMeta.Headers()),
		nil
}

func call(client helloclient.Interface, message string) (*echo.EchoResponse, yarpc.Headers) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resBody, resMeta, err := client.Echo(
		ctx,
		yarpc.NewReqMeta().Headers(yarpc.NewHeaders().With("from", "self")),
		&echo.EchoRequest{Message: message, Count: 1},
	)
	if err != nil {
		log.Fatal(err)
	}

	return resBody, resMeta.Headers()
}
