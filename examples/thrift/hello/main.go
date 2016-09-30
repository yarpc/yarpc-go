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
	"fmt"
	"log"
	"time"

	"github.com/yarpc/yarpc-go/examples/thrift/hello/thrift/hello"
	"github.com/yarpc/yarpc-go/examples/thrift/hello/thrift/hello/yarpc/helloclient"
	"github.com/yarpc/yarpc-go/examples/thrift/hello/thrift/hello/yarpc/helloserver"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	"golang.org/x/net/context"
)

//go:generate thriftrw --out thrift --plugin=yarpc hello.thrift

func main() {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Inbounds: []transport.Inbound{
			http.NewInbound(":8086"),
		},
		Outbounds: transport.Outbounds{
			"hello": http.NewOutbound("http://127.0.0.1:8086"),
		},
	})

	thrift.Register(dispatcher, helloserver.New(&helloHandler{}))
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

func (h helloHandler) Echo(ctx context.Context, reqMeta yarpc.ReqMeta, echo *hello.EchoRequest) (*hello.EchoResponse, yarpc.ResMeta, error) {
	return &hello.EchoResponse{Message: echo.Message, Count: echo.Count + 1},
		yarpc.NewResMeta().Headers(reqMeta.Headers()),
		nil
}

func call(client helloclient.Interface, message string) (*hello.EchoResponse, yarpc.Headers) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resBody, resMeta, err := client.Echo(
		ctx,
		yarpc.NewReqMeta().Headers(yarpc.NewHeaders().With("from", "self")),
		&hello.EchoRequest{Message: message, Count: 1},
	)
	if err != nil {
		log.Fatal(err)
	}

	return resBody, resMeta.Headers()
}
