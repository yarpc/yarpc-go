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
	"log"
	"time"

	"go.uber.org/yarpc/internal/examples/redis/sink"
	"go.uber.org/yarpc/internal/examples/redis/sink/yarpc/helloclient"
	"go.uber.org/yarpc/internal/examples/redis/sink/yarpc/helloserver"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/x/redis"
)

//go:generate thriftrw --plugin=yarpc sink.thrift

func main() {
	httpTransport := http.NewTransport()

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Inbounds: yarpc.Inbounds{
			httpTransport.NewInbound(":8086"),
			redis.NewInbound(
				redis.NewRedis5Client("localhost:6379"),
				"yarpc-queue",
				"yarpc-queue-processing",
				time.Duration(time.Second),
			),
		},
		Outbounds: yarpc.Outbounds{
			"hello": {
				Oneway: httpTransport.NewSingleOutbound("http://127.0.0.1:8080"),
				// Oneway: redis.NewOnewayOutbound(
				// 	redis.NewRedis5Client("localhost:6379"),
				// 	"yarpc-queue",
				// ),
			},
		},
	})

	dispatcher.Register(helloserver.New(&helloHandler{}))
	client := helloclient.New(dispatcher.ClientConfig("hello"))

	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()

	for {
		time.Sleep(time.Second / 2)
		client.Sink(context.Background(), nil, &sink.SinkRequest{Message: "yo", Count: 1})
	}
}

type helloHandler struct{}

func (h *helloHandler) Sink(ctx context.Context, reqMeta yarpc.ReqMeta, snk *sink.SinkRequest) error {
	log.Println("got message: ", snk.Message)
	return nil
}
