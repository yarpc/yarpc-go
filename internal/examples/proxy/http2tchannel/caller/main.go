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
	"flag"
	"log"
	"time"

	"go.uber.org/yarpc/examples/proxy/http2tchannel/service/service/yarpc/serviceclient"

	"go.uber.org/yarpc"
	yhttp "go.uber.org/yarpc/transport/http"
)

func main() {
	flag.Parse()
	outbound := yhttp.NewOutbound(flag.Arg(0))

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "caller",
		Outbounds: yarpc.Outbounds{
			"service": {
				Unary: outbound,
			},
		},
	})

	if err := dispatcher.Start(); err != nil {
		log.Fatalf("failed to start dispatcher: %v", err)
	}
	defer dispatcher.Stop()

	client := serviceclient.New(dispatcher.Channel("service"))

	call := func() {
		log.Printf("sending request\n")
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, _, err := client.Procedure(ctx, nil, "request")
		if err != nil {
			log.Printf("received response error: %v\n", err)
		} else {
			log.Printf("received response\n")
		}
	}

	for {
		call()
		time.Sleep(time.Second)
	}
}
