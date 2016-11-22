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

	"github.com/uber/tchannel-go"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/examples/proxy/http2tchannel/service/service/yarpc/serviceserver"
	ytchannel "go.uber.org/yarpc/transport/tchannel"
)

func main() {
	flag.Parse()

	channel, err := tchannel.NewChannel("service", nil)
	if err != nil {
		log.Fatalf("failed to create channel %v", err)
	}

	inbound := ytchannel.NewInbound(channel, ytchannel.ListenAddr(flag.Arg(0)))

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     "service",
		Inbounds: yarpc.Inbounds{inbound},
	})

	dispatcher.Register(serviceserver.New(&handler{}))

	if err := dispatcher.Start(); err != nil {
		log.Fatalf("failed to start dispatcher: %v", err)
	}
	defer dispatcher.Stop()

	log.Printf("server started\n")
	select {}
}

type handler struct {
}

func (h *handler) Procedure(ctx context.Context, reqMeta yarpc.ReqMeta, argument string) (string, yarpc.ResMeta, error) {
	log.Printf("handling request\n")
	return argument, nil, nil
}
