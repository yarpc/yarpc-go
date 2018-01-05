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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/examples/thrift-hello-binary/hellobinary/echobinary"
	"go.uber.org/yarpc/internal/examples/thrift-hello-binary/hellobinary/echobinary/hellobinaryclient"
	"go.uber.org/yarpc/internal/examples/thrift-hello-binary/hellobinary/echobinary/hellobinaryserver"
	"go.uber.org/yarpc/transport/http"
)

var flagWait = flag.Bool("wait", false, "Wait for a signal to exit")

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	flag.Parse()
	// configure a YARPC dispatcher for the service "hello",
	// expose the service over an HTTP inbound on port 8086,
	// and configure outbound calls to service "hello" over HTTP port 8086 as well
	http := http.NewTransport()
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "hello",
		Inbounds: yarpc.Inbounds{
			http.NewInbound("127.0.0.1:8086"),
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
	dispatcher.Register(hellobinaryserver.New(&helloHandler{}))

	// start the dispatcher, which enables requests to be sent and received
	if err := dispatcher.Start(); err != nil {
		return err
	}
	defer dispatcher.Stop()

	// create a Thrift client configured to call the "hello" service
	// using the dispatcher and associated "hello" outbound
	client := hellobinaryclient.New(dispatcher.ClientConfig("hello"))

	// build a context with a 1 second deadline
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// use the Thrift client to call the Echo procedure using YARPC
	res, err := client.Echo(ctx, &echobinary.EchoBinaryRequest{Message: []byte("Hello world"), Count: 1})
	if err != nil {
		return err
	}
	fmt.Printf("%T{Message: %s, Count: %d}", res, string(res.Message), res.Count)

	if *flagWait {
		// Gracefully shut down if we receive an interrupt (^C) or a kill signal.
		// Upon returning, this will unravel cancel() to abort the outbound request
		// and dispatcher.Stop() which will block until graceful shutdown.
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		<-signals
	}
	return nil
}

type helloHandler struct{}

func (h helloHandler) Echo(ctx context.Context, e *echobinary.EchoBinaryRequest) (*echobinary.EchoBinaryResponse, error) {
	return &echobinary.EchoBinaryResponse{Message: e.Message, Count: e.Count + 1}, nil
}
