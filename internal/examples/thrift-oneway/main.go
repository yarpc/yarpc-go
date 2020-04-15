// Copyright (c) 2020 Uber Technologies, Inc.
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

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/examples/thrift-oneway/sink"
	"go.uber.org/yarpc/internal/examples/thrift-oneway/sink/helloclient"
	"go.uber.org/yarpc/internal/examples/thrift-oneway/sink/helloserver"
	"go.uber.org/yarpc/transport/http"
)

// This example illustrates how to make oneway calls using different oneway
// transports.
func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	httpTransport := http.NewTransport()

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Inbounds: yarpc.Inbounds{
			httpTransport.NewInbound("127.0.0.1:8888"),
		},
		Outbounds: yarpc.Outbounds{
			"hello": {
				Oneway: httpTransport.NewSingleOutbound("http://127.0.0.1:8888"),
			},
		},
	})

	// register the server handler, implemented below
	dispatcher.Register(helloserver.New(&helloHandler{}))
	// create a client to talk to our server
	client := helloclient.New(dispatcher.ClientConfig("hello"))

	// start the dispatcher
	if err := dispatcher.Start(); err != nil {
		return err
	}
	defer dispatcher.Stop()

	// Make outbound call every 500ms
	for {
		time.Sleep(time.Second / 2)
		if _, err := client.Sink(context.Background(), &sink.SinkRequest{Message: "hello!"}); err != nil {
			log.Print(err)
		}
	}
}

type helloHandler struct{}

// Sink is our server-side handler implementation
func (h *helloHandler) Sink(ctx context.Context, snk *sink.SinkRequest) error {
	log.Println("received message:", snk.Message)
	return nil
}
